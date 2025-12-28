// Ticket: 003-ANALYZERS — validate template parameter variations (baudrates, dropdowns) on a real Logic 2 server
//
// Context
// -------
// We generate analyzer templates from saved `.sal` sessions. We also support "typed overrides" (effectively
// changing settings programmatically). This script validates that setting variations are accepted by the
// server AND persist into the saved `.sal` → `meta.json`, which gives us a feedback loop:
//
//	expected settings -> AddAnalyzer -> SaveCapture(.sal) -> extract meta.json -> compare(meta, expected)
//
// We intentionally source dropdown option strings from an existing `.sal` (meta.json options list),
// so we don't have to guess the exact UI-visible strings.
//
// Usage
// -----
//
//	go run ./ttmp/2025/12/24/003-ANALYZERS--analyzers-add-remove-settings-templates/scripts/06-real-validate-template-variations/main.go \
//	  --host 127.0.0.1 --port 10430 --timeout 25s \
//	  --source-sal "/tmp/Session 6.sal" \
//	  --templates-dir "/home/manuel/workspaces/2025-12-27/salad-pass/salad/configs/analyzers" \
//	  --out-sal "/tmp/session-validate-variations.sal"
package main

import (
	"archive/zip"
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	pb "github.com/go-go-golems/salad/gen/saleae/automation"
	"github.com/go-go-golems/salad/internal/config"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

type Meta struct {
	Data struct {
		Name      string         `json:"name"`
		Analyzers []MetaAnalyzer `json:"analyzers"`
	} `json:"data"`
}

type MetaAnalyzer struct {
	NodeID   uint64           `json:"nodeId"`
	Type     string           `json:"type"`
	Name     string           `json:"name"`
	Settings []MetaSettingRow `json:"settings"`
}

type MetaSettingRow struct {
	Title   string          `json:"title"`
	Setting MetaSettingSpec `json:"setting"`
}

type MetaSettingSpec struct {
	Type    string              `json:"type"`
	Value   any                 `json:"value"`
	Options []MetaSettingOption `json:"options"`
}

type MetaSettingOption struct {
	DropdownText string `json:"dropdownText"`
	Value        any    `json:"value"`
}

type OptionsIndex map[string]map[string][]MetaSettingOption // analyzerType -> title -> options

type Created struct {
	VariationName string
	AnalyzerID    uint64
	Expected      map[string]*pb.AnalyzerSettingValue
}

func main() {
	var (
		host         = flag.String("host", "127.0.0.1", "Logic 2 automation host")
		port         = flag.Int("port", 10430, "Logic 2 automation port")
		timeout      = flag.Duration("timeout", 25*time.Second, "Dial/RPC timeout")
		sourceSAL    = flag.String("source-sal", "/tmp/Session 6.sal", "Source .sal used to learn dropdown option strings (meta.json options)")
		templatesDir = flag.String("templates-dir", "", "Directory containing generated session templates (e.g. configs/analyzers)")
		outSAL       = flag.String("out-sal", "/tmp/session-validate-variations.sal", "Where to save the validation .sal (on Logic host filesystem)")
		prefix       = flag.String("prefix", "session6-", "Template filename prefix to use for base templates")
		digital      = flag.String("digital", "0,1,2,3,4,5,6,7", "Digital channels to enable (comma-separated)")
		digitalHz    = flag.Uint64("digital-hz", 1_000_000, "Digital sample rate (Hz)")
		bufferMB     = flag.Uint64("buffer-mb", 16, "Capture buffer size (MB)")
	)
	flag.Parse()

	if *templatesDir == "" {
		_, _ = fmt.Fprintln(os.Stderr, "--templates-dir is required")
		os.Exit(2)
	}

	// IMPORTANT: Use per-RPC timeouts instead of a single "overall" deadline.
	// A single top-level deadline makes later RPCs fail immediately once elapsed.
	baseCtx := context.Background()

	addr := fmt.Sprintf("%s:%d", *host, *port)
	dialCtx, cancelDial := context.WithTimeout(baseCtx, *timeout)
	conn, err := dialAndWaitReady(dialCtx, addr)
	cancelDial()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "dial: %v\n", err)
		os.Exit(2)
	}
	defer func() { _ = conn.Close() }()
	manager := pb.NewManagerClient(conn)

	ctx, cancel := context.WithTimeout(baseCtx, *timeout)
	deviceID, err := firstPhysicalDeviceID(ctx, manager)
	cancel()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "device: %v\n", err)
		os.Exit(2)
	}

	digitalChannels, err := parseUint32CSV(*digital)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "parse --digital: %v\n", err)
		os.Exit(2)
	}
	if len(digitalChannels) == 0 {
		_, _ = fmt.Fprintln(os.Stderr, "at least one digital channel must be enabled")
		os.Exit(2)
	}

	// Load dropdown option strings from source .sal
	sourceMeta, err := readMetaFromSAL(*sourceSAL)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "read source meta: %v\n", err)
		os.Exit(2)
	}
	opts := buildOptionsIndex(sourceMeta)

	// Base templates
	baseAsyncSerial := findTemplateByType(*templatesDir, *prefix, "Async Serial")
	baseSPI := findTemplateByType(*templatesDir, *prefix, "SPI")
	if baseAsyncSerial == "" {
		_, _ = fmt.Fprintf(os.Stderr, "could not find base template for type %q (prefix %q) in %s\n", "Async Serial", *prefix, *templatesDir)
		os.Exit(2)
	}
	if baseSPI == "" {
		_, _ = fmt.Fprintf(os.Stderr, "could not find base template for type %q (prefix %q) in %s\n", "SPI", *prefix, *templatesDir)
		os.Exit(2)
	}

	// Start capture
	ctx, cancel = context.WithTimeout(baseCtx, *timeout)
	captureID, err := startManualCapture(ctx, manager, deviceID, digitalChannels, uint32(*digitalHz), uint32(*bufferMB))
	cancel()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "start capture: %v\n", err)
		os.Exit(2)
	}
	fmt.Printf("capture_id=%d\n", captureID)

	// Ensure stopped for stable analyzer ops + SaveCapture.
	ctx, cancel = context.WithTimeout(baseCtx, *timeout)
	_, _ = manager.StopCapture(ctx, &pb.StopCaptureRequest{CaptureId: captureID})
	cancel()

	type Variation struct {
		Name         string
		Analyzer     string
		BaseTemplate string
		Apply        func(expected map[string]*pb.AnalyzerSettingValue) (map[string]*pb.AnalyzerSettingValue, error)
	}

	variations := []Variation{
		{
			Name:         "async-serial-baud-9600",
			Analyzer:     "Async Serial",
			BaseTemplate: baseAsyncSerial,
			Apply: func(expected map[string]*pb.AnalyzerSettingValue) (map[string]*pb.AnalyzerSettingValue, error) {
				return withInt(expected, "Bit Rate (Bits/s)", 9600), nil
			},
		},
		{
			Name:         "async-serial-baud-57600",
			Analyzer:     "Async Serial",
			BaseTemplate: baseAsyncSerial,
			Apply: func(expected map[string]*pb.AnalyzerSettingValue) (map[string]*pb.AnalyzerSettingValue, error) {
				return withInt(expected, "Bit Rate (Bits/s)", 57600), nil
			},
		},
		{
			Name:         "async-serial-baud-230400",
			Analyzer:     "Async Serial",
			BaseTemplate: baseAsyncSerial,
			Apply: func(expected map[string]*pb.AnalyzerSettingValue) (map[string]*pb.AnalyzerSettingValue, error) {
				return withInt(expected, "Bit Rate (Bits/s)", 230400), nil
			},
		},
		{
			Name:         "spi-mode3-bits16",
			Analyzer:     "SPI",
			BaseTemplate: baseSPI,
			Apply: func(expected map[string]*pb.AnalyzerSettingValue) (map[string]*pb.AnalyzerSettingValue, error) {
				// Use dropdownText values from source meta.json options.
				phase, err := optionText(opts, "SPI", "Clock Phase", 1) // CPHA=1
				if err != nil {
					return nil, err
				}
				state, err := optionText(opts, "SPI", "Clock State", 1) // CPOL=1
				if err != nil {
					return nil, err
				}
				bits, err := optionText(opts, "SPI", "Bits per Transfer", 16)
				if err != nil {
					return nil, err
				}
				out := withString(expected, "Clock Phase", phase)
				out = withString(out, "Clock State", state)
				out = withString(out, "Bits per Transfer", bits)
				return out, nil
			},
		},
	}

	var created []Created
	var addFailures []string
	for _, v := range variations {
		baseSettings, err := config.LoadAnalyzerSettings(v.BaseTemplate)
		if err != nil {
			addFailures = append(addFailures, fmt.Sprintf("%s: load base template %s: %v", v.Name, v.BaseTemplate, err))
			continue
		}
		exp, err := v.Apply(baseSettings)
		if err != nil {
			addFailures = append(addFailures, fmt.Sprintf("%s: apply variation: %v", v.Name, err))
			continue
		}

		label := "var:" + v.Name
		ctx, cancel = context.WithTimeout(baseCtx, *timeout)
		reply, err := manager.AddAnalyzer(ctx, &pb.AddAnalyzerRequest{
			CaptureId:     captureID,
			AnalyzerName:  v.Analyzer,
			AnalyzerLabel: label,
			Settings:      exp,
		})
		cancel()
		if err != nil {
			addFailures = append(addFailures, fmt.Sprintf("%s: AddAnalyzer(name=%q): %v", v.Name, v.Analyzer, err))
			continue
		}
		analyzerID := reply.GetAnalyzerId()
		fmt.Printf("added %s analyzer_id=%d\n", v.Name, analyzerID)
		created = append(created, Created{VariationName: v.Name, AnalyzerID: analyzerID, Expected: exp})
	}

	if len(addFailures) > 0 {
		fmt.Println("ADD FAILURES:")
		for _, f := range addFailures {
			fmt.Printf("- %s\n", f)
		}
		ctx, cancel = context.WithTimeout(baseCtx, *timeout)
		_ = cleanupCapture(ctx, manager, captureID, ids(created))
		cancel()
		os.Exit(1)
	}

	// Save capture to outSAL (on Logic host filesystem).
	ctx, cancel = context.WithTimeout(baseCtx, *timeout)
	if _, err := manager.SaveCapture(ctx, &pb.SaveCaptureRequest{CaptureId: captureID, Filepath: *outSAL}); err != nil {
		cancel()
		ctx2, cancel2 := context.WithTimeout(baseCtx, *timeout)
		_ = cleanupCapture(ctx2, manager, captureID, ids(created))
		cancel2()
		_, _ = fmt.Fprintf(os.Stderr, "SaveCapture: %v\n", err)
		os.Exit(2)
	}
	cancel()
	fmt.Printf("saved_sal=%s\n", *outSAL)

	// Read meta.json back and compare.
	savedMeta, err := readMetaFromSAL(*outSAL)
	if err != nil {
		ctx2, cancel2 := context.WithTimeout(baseCtx, *timeout)
		_ = cleanupCapture(ctx2, manager, captureID, ids(created))
		cancel2()
		_, _ = fmt.Fprintf(os.Stderr, "read saved meta from %s: %v\n", *outSAL, err)
		os.Exit(2)
	}

	var mismatches []string
	for _, c := range created {
		an, ok := findMetaAnalyzer(savedMeta, c.AnalyzerID)
		if !ok {
			mismatches = append(mismatches, fmt.Sprintf("%s: analyzer_id=%d not found in saved meta.json", c.VariationName, c.AnalyzerID))
			continue
		}
		got, err := extractSettingsFromMeta(an)
		if err != nil {
			mismatches = append(mismatches, fmt.Sprintf("%s: extract meta settings: %v", c.VariationName, err))
			continue
		}
		if diff := subsetDiff(c.Expected, got); diff != "" {
			mismatches = append(mismatches, fmt.Sprintf("%s (analyzer_id=%d):\n%s", c.VariationName, c.AnalyzerID, diff))
		}
	}

	ctx, cancel = context.WithTimeout(baseCtx, *timeout)
	if err := cleanupCapture(ctx, manager, captureID, ids(created)); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "cleanup: %v\n", err)
	}
	cancel()

	if len(mismatches) > 0 {
		fmt.Println("MISMATCHES:")
		for _, m := range mismatches {
			fmt.Println("----")
			fmt.Println(m)
		}
		os.Exit(1)
	}

	fmt.Printf("all ok (%d variations)\n", len(created))
}

func ids(cs []Created) []uint64 {
	out := make([]uint64, 0, len(cs))
	for _, c := range cs {
		out = append(out, c.AnalyzerID)
	}
	return out
}

func readMetaFromSAL(path string) (*Meta, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, errors.Wrapf(err, "open zip %s", path)
	}
	defer func() { _ = r.Close() }()

	var metaFile *zip.File
	for _, f := range r.File {
		if f.Name == "meta.json" {
			metaFile = f
			break
		}
	}
	if metaFile == nil {
		return nil, errors.Errorf("meta.json not found in %s", path)
	}

	rc, err := metaFile.Open()
	if err != nil {
		return nil, errors.Wrap(err, "open meta.json")
	}
	defer func() { _ = rc.Close() }()

	b, err := io.ReadAll(rc)
	if err != nil {
		return nil, errors.Wrap(err, "read meta.json")
	}
	var m Meta
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, errors.Wrap(err, "decode meta.json")
	}
	return &m, nil
}

func buildOptionsIndex(m *Meta) OptionsIndex {
	out := OptionsIndex{}
	if m == nil {
		return out
	}
	for _, a := range m.Data.Analyzers {
		if strings.TrimSpace(a.Type) == "" {
			continue
		}
		if _, ok := out[a.Type]; !ok {
			out[a.Type] = map[string][]MetaSettingOption{}
		}
		for _, s := range a.Settings {
			title := strings.TrimSpace(s.Title)
			if title == "" {
				continue
			}
			if strings.TrimSpace(s.Setting.Type) != "NumberList" {
				continue
			}
			// Store first occurrence (options are typically identical across analyzers of same type)
			if _, ok := out[a.Type][title]; ok {
				continue
			}
			out[a.Type][title] = s.Setting.Options
		}
	}
	return out
}

func optionText(opts OptionsIndex, analyzerType, title string, numericValue int) (string, error) {
	m, ok := opts[analyzerType]
	if !ok {
		return "", errors.Errorf("no options for analyzer type %q", analyzerType)
	}
	list, ok := m[title]
	if !ok || len(list) == 0 {
		return "", errors.Errorf("no options for %q/%q", analyzerType, title)
	}
	for _, o := range list {
		if valuesEqual(o.Value, numericValue) && strings.TrimSpace(o.DropdownText) != "" {
			return o.DropdownText, nil
		}
	}
	return "", errors.Errorf("no dropdownText match for %q/%q value=%d", analyzerType, title, numericValue)
}

func findTemplateByType(dir, prefix, analyzerType string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	var candidates []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if prefix != "" && !strings.HasPrefix(name, prefix) {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(name), ".yaml") && !strings.HasSuffix(strings.ToLower(name), ".yml") {
			continue
		}
		full := filepath.Join(dir, name)
		t, err := parseAnalyzerTypeFromHeader(full)
		if err != nil {
			continue
		}
		if t == analyzerType {
			candidates = append(candidates, full)
		}
	}
	sort.Strings(candidates)
	if len(candidates) == 0 {
		return ""
	}
	return candidates[0]
}

var reAnalyzerType = regexp.MustCompile(`type=(?:'([^']+)'|"([^"]+)")`)

func parseAnalyzerTypeFromHeader(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", errors.Wrapf(err, "open %s", path)
	}
	defer func() { _ = f.Close() }()

	sc := bufio.NewScanner(f)
	for i := 0; i < 60 && sc.Scan(); i++ {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "#") {
			continue
		}
		m := reAnalyzerType.FindStringSubmatch(line)
		if len(m) == 3 {
			if m[1] != "" {
				return m[1], nil
			}
			if m[2] != "" {
				return m[2], nil
			}
		}
	}
	if err := sc.Err(); err != nil {
		return "", errors.Wrapf(err, "scan %s", path)
	}
	return "", errors.Errorf("could not find analyzer type in header comments of %s", path)
}

func dialAndWaitReady(ctx context.Context, addr string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, errors.Wrapf(err, "grpc.NewClient(%s)", addr)
	}
	conn.Connect()
	for {
		state := conn.GetState()
		if state == connectivity.Ready {
			return conn, nil
		}
		if state == connectivity.Shutdown {
			_ = conn.Close()
			return nil, errors.Errorf("grpc connection shutdown while connecting to %s", addr)
		}
		if !conn.WaitForStateChange(ctx, state) {
			_ = conn.Close()
			return nil, errors.Wrapf(ctx.Err(), "connect to %s", addr)
		}
	}
}

func firstPhysicalDeviceID(ctx context.Context, client pb.ManagerClient) (string, error) {
	reply, err := client.GetDevices(ctx, &pb.GetDevicesRequest{IncludeSimulationDevices: false})
	if err != nil {
		return "", errors.Wrap(err, "GetDevices RPC")
	}
	for _, d := range reply.GetDevices() {
		if d == nil || d.GetIsSimulation() {
			continue
		}
		if d.GetDeviceId() == "" {
			continue
		}
		return d.GetDeviceId(), nil
	}
	return "", errors.New("no physical devices found")
}

func startManualCapture(
	ctx context.Context,
	client pb.ManagerClient,
	deviceID string,
	digitalChannels []uint32,
	digitalHz uint32,
	bufferMB uint32,
) (uint64, error) {
	req := &pb.StartCaptureRequest{
		DeviceId: deviceID,
		DeviceConfiguration: &pb.StartCaptureRequest_LogicDeviceConfiguration{
			LogicDeviceConfiguration: &pb.LogicDeviceConfiguration{
				EnabledChannels: &pb.LogicDeviceConfiguration_LogicChannels{
					LogicChannels: &pb.LogicChannels{
						DigitalChannels: digitalChannels,
					},
				},
				DigitalSampleRate: digitalHz,
			},
		},
		CaptureConfiguration: &pb.CaptureConfiguration{
			BufferSizeMegabytes: bufferMB,
			CaptureMode: &pb.CaptureConfiguration_ManualCaptureMode{
				ManualCaptureMode: &pb.ManualCaptureMode{TrimDataSeconds: 0},
			},
		},
	}
	reply, err := client.StartCapture(ctx, req)
	if err != nil {
		return 0, errors.Wrap(err, "StartCapture RPC")
	}
	if reply.GetCaptureInfo() == nil {
		return 0, errors.New("StartCapture: reply.capture_info is nil")
	}
	return reply.GetCaptureInfo().GetCaptureId(), nil
}

func cleanupCapture(ctx context.Context, client pb.ManagerClient, captureID uint64, analyzerIDs []uint64) error {
	var firstErr error
	for _, id := range analyzerIDs {
		_, err := client.RemoveAnalyzer(ctx, &pb.RemoveAnalyzerRequest{
			CaptureId:  captureID,
			AnalyzerId: id,
		})
		if err != nil && firstErr == nil {
			firstErr = errors.Wrapf(err, "RemoveAnalyzer(capture_id=%d, analyzer_id=%d)", captureID, id)
		}
	}
	if _, err := client.StopCapture(ctx, &pb.StopCaptureRequest{CaptureId: captureID}); err != nil && firstErr == nil {
		firstErr = errors.Wrapf(err, "StopCapture(capture_id=%d)", captureID)
	}
	if _, err := client.CloseCapture(ctx, &pb.CloseCaptureRequest{CaptureId: captureID}); err != nil && firstErr == nil {
		firstErr = errors.Wrapf(err, "CloseCapture(capture_id=%d)", captureID)
	}
	return firstErr
}

func parseUint32CSV(s string) ([]uint32, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	parts := strings.Split(s, ",")
	out := make([]uint32, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		v, err := strconv.ParseUint(part, 10, 32)
		if err != nil {
			return nil, errors.Wrapf(err, "parse uint32 %q", part)
		}
		out = append(out, uint32(v))
	}
	return out, nil
}

func findMetaAnalyzer(m *Meta, nodeID uint64) (*MetaAnalyzer, bool) {
	if m == nil {
		return nil, false
	}
	for i := range m.Data.Analyzers {
		if m.Data.Analyzers[i].NodeID == nodeID {
			return &m.Data.Analyzers[i], true
		}
	}
	return nil, false
}

func extractSettingsFromMeta(a *MetaAnalyzer) (map[string]*pb.AnalyzerSettingValue, error) {
	out := map[string]*pb.AnalyzerSettingValue{}
	if a == nil {
		return out, nil
	}
	for _, row := range a.Settings {
		key := strings.TrimSpace(row.Title)
		if key == "" {
			continue
		}
		sv, err := metaValueToPB(row.Setting)
		if err != nil {
			return nil, errors.Wrapf(err, "setting %q", key)
		}
		out[key] = sv
	}
	return out, nil
}

func metaValueToPB(s MetaSettingSpec) (*pb.AnalyzerSettingValue, error) {
	switch strings.TrimSpace(s.Type) {
	case "Channel":
		n, ok := asInt64(s.Value)
		if !ok {
			return nil, errors.Errorf("Channel value is not numeric (%T)", s.Value)
		}
		return &pb.AnalyzerSettingValue{Value: &pb.AnalyzerSettingValue_Int64Value{Int64Value: n}}, nil
	case "NumberList":
		cur := s.Value
		for _, opt := range s.Options {
			if valuesEqual(opt.Value, cur) && strings.TrimSpace(opt.DropdownText) != "" {
				return &pb.AnalyzerSettingValue{Value: &pb.AnalyzerSettingValue_StringValue{StringValue: opt.DropdownText}}, nil
			}
		}
		// Fallback numeric
		if n, ok := asInt64(cur); ok {
			return &pb.AnalyzerSettingValue{Value: &pb.AnalyzerSettingValue_Int64Value{Int64Value: n}}, nil
		}
		return nil, errors.Errorf("NumberList value has no matching dropdownText and is not numeric (%T)", cur)
	default:
		// Best-effort scalar
		switch v := s.Value.(type) {
		case string:
			return &pb.AnalyzerSettingValue{Value: &pb.AnalyzerSettingValue_StringValue{StringValue: v}}, nil
		case bool:
			return &pb.AnalyzerSettingValue{Value: &pb.AnalyzerSettingValue_BoolValue{BoolValue: v}}, nil
		default:
			if n, ok := asInt64(s.Value); ok {
				return &pb.AnalyzerSettingValue{Value: &pb.AnalyzerSettingValue_Int64Value{Int64Value: n}}, nil
			}
		}
		return nil, errors.Errorf("unsupported meta setting type %q", s.Type)
	}
}

func subsetDiff(expected, got map[string]*pb.AnalyzerSettingValue) string {
	var keys []string
	for k := range expected {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		ev := expected[k]
		gv, ok := got[k]
		if !ok {
			fmt.Fprintf(&b, "- missing key %q (expected %s)\n", k, fmtPB(ev))
			continue
		}
		if !pbEqual(ev, gv) {
			fmt.Fprintf(&b, "~ %q\n  expected: %s\n  got:      %s\n", k, fmtPB(ev), fmtPB(gv))
		}
	}
	return b.String()
}

func pbEqual(a, b *pb.AnalyzerSettingValue) bool {
	if a == nil || b == nil {
		return a == b
	}
	switch av := a.Value.(type) {
	case *pb.AnalyzerSettingValue_StringValue:
		bv, ok := b.Value.(*pb.AnalyzerSettingValue_StringValue)
		return ok && av.StringValue == bv.StringValue
	case *pb.AnalyzerSettingValue_Int64Value:
		bv, ok := b.Value.(*pb.AnalyzerSettingValue_Int64Value)
		return ok && av.Int64Value == bv.Int64Value
	case *pb.AnalyzerSettingValue_BoolValue:
		bv, ok := b.Value.(*pb.AnalyzerSettingValue_BoolValue)
		return ok && av.BoolValue == bv.BoolValue
	case *pb.AnalyzerSettingValue_DoubleValue:
		bv, ok := b.Value.(*pb.AnalyzerSettingValue_DoubleValue)
		return ok && av.DoubleValue == bv.DoubleValue
	default:
		return false
	}
}

func fmtPB(v *pb.AnalyzerSettingValue) string {
	if v == nil {
		return "<nil>"
	}
	switch tv := v.Value.(type) {
	case *pb.AnalyzerSettingValue_StringValue:
		return fmt.Sprintf("%q", tv.StringValue)
	case *pb.AnalyzerSettingValue_Int64Value:
		return fmt.Sprintf("%d", tv.Int64Value)
	case *pb.AnalyzerSettingValue_BoolValue:
		if tv.BoolValue {
			return "true"
		}
		return "false"
	case *pb.AnalyzerSettingValue_DoubleValue:
		return fmt.Sprintf("%v", tv.DoubleValue)
	default:
		return "<unknown>"
	}
}

func withInt(base map[string]*pb.AnalyzerSettingValue, key string, v int64) map[string]*pb.AnalyzerSettingValue {
	out := cloneMap(base)
	out[key] = &pb.AnalyzerSettingValue{Value: &pb.AnalyzerSettingValue_Int64Value{Int64Value: v}}
	return out
}

func withString(base map[string]*pb.AnalyzerSettingValue, key string, v string) map[string]*pb.AnalyzerSettingValue {
	out := cloneMap(base)
	out[key] = &pb.AnalyzerSettingValue{Value: &pb.AnalyzerSettingValue_StringValue{StringValue: v}}
	return out
}

func cloneMap(m map[string]*pb.AnalyzerSettingValue) map[string]*pb.AnalyzerSettingValue {
	out := make(map[string]*pb.AnalyzerSettingValue, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func asInt64(v any) (int64, bool) {
	switch t := v.(type) {
	case float64:
		return int64(t), true
	case int64:
		return t, true
	case int:
		return int64(t), true
	case uint64:
		return int64(t), true
	default:
		return 0, false
	}
}

func valuesEqual(a any, b any) bool {
	if ai, ok := asInt64(a); ok {
		if bi, ok := asInt64(b); ok {
			return ai == bi
		}
	}
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}
