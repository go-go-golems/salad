// Ticket: 003-ANALYZERS — analyzer settings templates feedback loop
//
// Context
// -------
// We can apply analyzer settings via the gRPC AddAnalyzer settings map, but the automation API doesn't
// let us read back the analyzer settings from the running app. However, a saved `.sal` contains a
// `meta.json` which includes analyzer settings (including dropdown UI strings).
//
// This script compares the settings for one analyzer in `meta.json` (selected by nodeId) against a
// YAML/JSON template file that `salad analyzer add --settings-*` would consume.
//
// Primary use: regression/verification loop
//   template -> AddAnalyzer -> SaveCapture -> extract meta.json -> compare(meta.json, template)
//
// Usage
// -----
//   go run ./ttmp/.../scripts/03-compare-meta-json-to-template/main.go \
//     --meta /tmp/meta.json \
//     --node-id 10038 \
//     --template /abs/path/to/configs/analyzers/spi-from-session6.yaml
//
// Notes
// -----
// - Dropdown values in meta.json include both numeric codes and dropdownText strings; we compare using
//   dropdownText strings (matches Saleae automation docs and our template style).
// - meta.json contains non-gRPC fields (showInDataTable/streamToTerminal); those are ignored.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	pb "github.com/go-go-golems/salad/gen/saleae/automation"
	"github.com/go-go-golems/salad/internal/config"
	"github.com/pkg/errors"
)

type Meta struct {
	Version int `json:"version"`
	Data    struct {
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
	Type    string             `json:"type"`
	Value   any                `json:"value"`
	Options []MetaSettingOption `json:"options"`
}

type MetaSettingOption struct {
	DropdownText    string `json:"dropdownText"`
	DropdownTooltip string `json:"dropdownTooltip"`
	Value           any    `json:"value"`
}

func main() {
	var (
		metaPath     = flag.String("meta", "", "Path to meta.json extracted from a .sal")
		nodeID       = flag.Uint64("node-id", 0, "Analyzer nodeId to compare (usually equals analyzer_id)")
		templatePath = flag.String("template", "", "Path to analyzer settings template (.yaml/.yml/.json)")
	)
	flag.Parse()

	if *metaPath == "" {
		_, _ = fmt.Fprintln(os.Stderr, "--meta is required")
		os.Exit(2)
	}
	if *nodeID == 0 {
		_, _ = fmt.Fprintln(os.Stderr, "--node-id is required")
		os.Exit(2)
	}
	if *templatePath == "" {
		_, _ = fmt.Fprintln(os.Stderr, "--template is required")
		os.Exit(2)
	}

	meta, err := loadMeta(*metaPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "load meta: %v\n", err)
		os.Exit(2)
	}

	an, ok := findAnalyzer(meta, *nodeID)
	if !ok {
		_, _ = fmt.Fprintf(os.Stderr, "analyzer nodeId=%d not found in %s\n", *nodeID, *metaPath)
		os.Exit(2)
	}

	fromMeta, err := extractSettingsFromMeta(an)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "extract meta settings: %v\n", err)
		os.Exit(2)
	}

	fromTemplate, err := config.LoadAnalyzerSettings(*templatePath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "load template settings: %v\n", err)
		os.Exit(2)
	}

	diff := diffSettings(fromTemplate, fromMeta)
	if diff == "" {
		fmt.Println("ok")
		return
	}

	fmt.Println("DIFF (template vs meta.json):")
	fmt.Print(diff)
	os.Exit(1)
}

func loadMeta(path string) (*Meta, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "read %s", path)
	}
	var m Meta
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, errors.Wrapf(err, "decode %s", path)
	}
	return &m, nil
}

func findAnalyzer(m *Meta, nodeID uint64) (*MetaAnalyzer, bool) {
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
			return nil, errors.New("meta setting row has empty title")
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
		// channel index is numeric
		n, ok := asInt64(s.Value)
		if !ok {
			return nil, errors.Errorf("Channel value is not numeric (%T)", s.Value)
		}
		return &pb.AnalyzerSettingValue{Value: &pb.AnalyzerSettingValue_Int64Value{Int64Value: n}}, nil
	case "NumberList":
		// Compare using dropdownText (string) for current selection if possible
		cur := s.Value
		for _, opt := range s.Options {
			if valuesEqual(opt.Value, cur) && strings.TrimSpace(opt.DropdownText) != "" {
				return &pb.AnalyzerSettingValue{Value: &pb.AnalyzerSettingValue_StringValue{StringValue: opt.DropdownText}}, nil
			}
		}
		// Fallback: numeric
		if n, ok := asInt64(cur); ok {
			return &pb.AnalyzerSettingValue{Value: &pb.AnalyzerSettingValue_Int64Value{Int64Value: n}}, nil
		}
		return nil, errors.Errorf("NumberList value has no matching dropdownText and is not numeric (%T)", cur)
	default:
		return nil, errors.Errorf("unsupported meta setting type %q", s.Type)
	}
}

func diffSettings(template, meta map[string]*pb.AnalyzerSettingValue) string {
	var keys []string
	seen := map[string]bool{}
	for k := range template {
		keys = append(keys, k)
		seen[k] = true
	}
	for k := range meta {
		if !seen[k] {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		tv, tok := template[k]
		mv, mok := templateValueOr(meta, k)
		if !tok {
			fmt.Fprintf(&b, "+ meta-only %q = %s\n", k, fmtPB(mv))
			continue
		}
		if !mok {
			fmt.Fprintf(&b, "- template-only %q = %s\n", k, fmtPB(tv))
			continue
		}
		if !pbEqual(tv, mv) {
			fmt.Fprintf(&b, "~ %q\n  template: %s\n  meta:     %s\n", k, fmtPB(tv), fmtPB(mv))
		}
	}
	return b.String()
}

func templateValueOr(m map[string]*pb.AnalyzerSettingValue, k string) (*pb.AnalyzerSettingValue, bool) {
	v, ok := m[k]
	return v, ok
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

func asInt64(v any) (int64, bool) {
	switch t := v.(type) {
	case float64:
		// JSON numbers decode as float64
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

func valuesEqual(a, b any) bool {
	// The JSON decoder yields float64 for numbers; compare via int64 when possible.
	if ai, ok := asInt64(a); ok {
		if bi, ok := asInt64(b); ok {
			return ai == bi
		}
	}
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}


