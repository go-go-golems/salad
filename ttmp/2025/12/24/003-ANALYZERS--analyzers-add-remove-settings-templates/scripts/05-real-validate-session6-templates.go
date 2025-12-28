// Ticket: 003-ANALYZERS — validate generated analyzer templates against a real Logic 2 session
//
// Context
// -------
// We generate analyzer templates from a saved .sal (meta.json) to capture the exact UI setting keys and
// dropdown strings. This script performs an end-to-end validation against a real Logic 2 automation server:
//
//   - Start a new capture (enable digital channels)
//   - Stop the capture (avoid "switch sessions while recording" issues)
//   - For each template file matching a prefix (default: "session6-"), determine analyzer name,
//     load settings, call AddAnalyzer, record analyzer_id
//   - Remove all created analyzers, then stop/close capture
//
// Usage
// -----
//   go run ./ttmp/2025/12/24/003-ANALYZERS--analyzers-add-remove-settings-templates/scripts/05-real-validate-session6-templates.go \
//     --host 127.0.0.1 --port 10430 --timeout 12s \
//     --templates-dir /home/manuel/workspaces/2025-12-27/salad-pass/salad/configs/analyzers \
//     --prefix session6-
//
// Notes
// -----
// - Analyzer "name" must match UI name exactly (e.g. "Async Serial", "DMX-512", "1-Wire").
//   The bulk template generator writes this in a header line: `# Analyzer: ... type='X' ...`.
// - This script reads that header to select the analyzer name; it does not guess from filename.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
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

func main() {
	var (
		host         = flag.String("host", "127.0.0.1", "Logic 2 automation host")
		port         = flag.Int("port", 10430, "Logic 2 automation port")
		timeout      = flag.Duration("timeout", 12*time.Second, "Dial/RPC timeout")
		templatesDir = flag.String("templates-dir", "", "Directory containing YAML templates to validate")
		prefix       = flag.String("prefix", "session6-", "Only validate templates whose filename starts with this prefix")
		digital      = flag.String("digital", "0,1,2,3,4,5,6,7", "Digital channels to enable (comma-separated)")
		digitalHz    = flag.Uint64("digital-hz", 1_000_000, "Digital sample rate (Hz)")
		bufferMB     = flag.Uint64("buffer-mb", 16, "Capture buffer size (MB)")
	)
	flag.Parse()

	if *templatesDir == "" {
		_, _ = fmt.Fprintln(os.Stderr, "--templates-dir is required")
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	addr := fmt.Sprintf("%s:%d", *host, *port)
	conn, err := dialAndWaitReady(ctx, addr)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "dial: %v\n", err)
		os.Exit(2)
	}
	defer func() { _ = conn.Close() }()

	manager := pb.NewManagerClient(conn)

	deviceID, err := firstPhysicalDeviceID(ctx, manager)
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

	captureID, err := startManualCapture(ctx, manager, deviceID, digitalChannels, uint32(*digitalHz), uint32(*bufferMB))
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "start capture: %v\n", err)
		os.Exit(2)
	}
	fmt.Printf("capture_id=%d\n", captureID)

	// Stop immediately so analyzer operations don't depend on recording state.
	if _, err := manager.StopCapture(ctx, &pb.StopCaptureRequest{CaptureId: captureID}); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "warning: StopCapture: %v\n", err)
	}

	templates, err := listTemplates(*templatesDir, *prefix)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "list templates: %v\n", err)
		_ = cleanupCapture(ctx, manager, captureID, nil)
		os.Exit(2)
	}
	if len(templates) == 0 {
		_, _ = fmt.Fprintf(os.Stderr, "no templates found in %s matching prefix %q\n", *templatesDir, *prefix)
		_ = cleanupCapture(ctx, manager, captureID, nil)
		os.Exit(1)
	}

	var (
		createdIDs []uint64
		failures   []string
	)

	for _, t := range templates {
		analyzerName, err := parseAnalyzerTypeFromHeader(t)
		if err != nil {
			failures = append(failures, fmt.Sprintf("template=%s: parse analyzer type: %v", t, err))
			continue
		}

		settings, err := config.LoadAnalyzerSettings(t)
		if err != nil {
			failures = append(failures, fmt.Sprintf("template=%s: load settings: %v", t, err))
			continue
		}

		label := "tmpl:" + filepath.Base(t)
		reply, err := manager.AddAnalyzer(ctx, &pb.AddAnalyzerRequest{
			CaptureId:     captureID,
			AnalyzerName:  analyzerName,
			AnalyzerLabel: label,
			Settings:      settings,
		})
		if err != nil {
			failures = append(failures, fmt.Sprintf("template=%s name=%q: AddAnalyzer: %v", t, analyzerName, err))
			continue
		}
		analyzerID := reply.GetAnalyzerId()
		createdIDs = append(createdIDs, analyzerID)
		fmt.Printf("ok template=%s name=%q analyzer_id=%d\n", filepath.Base(t), analyzerName, analyzerID)
	}

	if err := cleanupCapture(ctx, manager, captureID, createdIDs); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "cleanup: %v\n", err)
		// continue; we still want to report failures
	}

	if len(failures) > 0 {
		fmt.Println("FAILED:")
		for _, f := range failures {
			fmt.Printf("- %s\n", f)
		}
		os.Exit(1)
	}

	fmt.Printf("all ok (%d templates)\n", len(templates))
}

func listTemplates(dir, prefix string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, errors.Wrapf(err, "ReadDir(%s)", dir)
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".yaml") && !strings.HasSuffix(strings.ToLower(name), ".yml") {
			continue
		}
		if prefix != "" && !strings.HasPrefix(name, prefix) {
			continue
		}
		out = append(out, filepath.Join(dir, name))
	}
	sort.Strings(out)
	return out, nil
}

var reAnalyzerType = regexp.MustCompile(`type=(?:'([^']+)'|"([^"]+)")`)

func parseAnalyzerTypeFromHeader(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", errors.Wrapf(err, "open %s", path)
	}
	defer func() { _ = f.Close() }()

	sc := bufio.NewScanner(f)
	// Scan first ~60 lines for the header.
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


