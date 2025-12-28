package config

import (
	"strings"
	"testing"

	pb "github.com/go-go-golems/salad/gen/saleae/automation"
)

func TestLoadAnalyzerSettingsFromReader_JSON_TopLevelMap(t *testing.T) {
	settings, err := LoadAnalyzerSettingsFromReader(strings.NewReader(`{
  "mode": "SPI",
  "enabled": true,
  "baud": 115200,
  "threshold": 0.25
}`), "json")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if got := settings["mode"].GetStringValue(); got != "SPI" {
		t.Fatalf("mode: expected %q, got %q", "SPI", got)
	}
	if got := settings["enabled"].GetBoolValue(); got != true {
		t.Fatalf("enabled: expected %v, got %v", true, got)
	}
	if got := settings["baud"].GetInt64Value(); got != 115200 {
		t.Fatalf("baud: expected %d, got %d", int64(115200), got)
	}
	if got := settings["threshold"].GetDoubleValue(); got < 0.249999 || got > 0.250001 {
		t.Fatalf("threshold: expected ~%f, got %f", 0.25, got)
	}
}

func TestLoadAnalyzerSettingsFromReader_JSON_SettingsWrapper(t *testing.T) {
	settings, err := LoadAnalyzerSettingsFromReader(strings.NewReader(`{
  "settings": {
    "a": 1,
    "b": "x"
  }
}`), "json")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got := settings["a"].GetInt64Value(); got != 1 {
		t.Fatalf("a: expected %d, got %d", int64(1), got)
	}
	if got := settings["b"].GetStringValue(); got != "x" {
		t.Fatalf("b: expected %q, got %q", "x", got)
	}
}

func TestLoadAnalyzerSettingsFromReader_YAML_TopLevelMap(t *testing.T) {
	settings, err := LoadAnalyzerSettingsFromReader(strings.NewReader(`
mode: SPI
enabled: true
baud: 115200
threshold: 0.25
`), "yaml")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if got := settings["mode"].GetStringValue(); got != "SPI" {
		t.Fatalf("mode: expected %q, got %q", "SPI", got)
	}
	if got := settings["enabled"].GetBoolValue(); got != true {
		t.Fatalf("enabled: expected %v, got %v", true, got)
	}
	if got := settings["baud"].GetInt64Value(); got != 115200 {
		t.Fatalf("baud: expected %d, got %d", int64(115200), got)
	}
	if got := settings["threshold"].GetDoubleValue(); got < 0.249999 || got > 0.250001 {
		t.Fatalf("threshold: expected ~%f, got %f", 0.25, got)
	}
}

func TestApplyAnalyzerSettingOverrides(t *testing.T) {
	base := map[string]*pb.AnalyzerSettingValue{
		"a": {Value: &pb.AnalyzerSettingValue_StringValue{StringValue: "x"}},
		"b": {Value: &pb.AnalyzerSettingValue_Int64Value{Int64Value: 1}},
	}

	out, err := ApplyAnalyzerSettingOverrides(
		base,
		[]string{"a=override"},
		[]string{"flag=true"},
		[]string{"b=42"},
		[]string{"f=3.14"},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if got := out["a"].GetStringValue(); got != "override" {
		t.Fatalf("a: expected %q, got %q", "override", got)
	}
	if got := out["b"].GetInt64Value(); got != 42 {
		t.Fatalf("b: expected %d, got %d", int64(42), got)
	}
	if got := out["flag"].GetBoolValue(); got != true {
		t.Fatalf("flag: expected %v, got %v", true, got)
	}
	if got := out["f"].GetDoubleValue(); got < 3.139999 || got > 3.140001 {
		t.Fatalf("f: expected ~%f, got %f", 3.14, got)
	}
}
