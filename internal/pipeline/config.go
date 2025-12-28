package pipeline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// Config defines the v1 pipeline config that can be executed with `salad run --config ...`.
//
// NOTE: This is intentionally scoped to what the current codebase can do today:
// - capture.load
// - add LLA analyzers
// - export raw-csv/raw-binary/table-csv
// - close capture
type Config struct {
	Version int `json:"version" yaml:"version"`

	Capture   CaptureConfig    `json:"capture" yaml:"capture"`
	Analyzers []AnalyzerConfig `json:"analyzers" yaml:"analyzers"`
	Exports   []ExportConfig   `json:"exports" yaml:"exports"`
	Cleanup   CleanupConfig    `json:"cleanup" yaml:"cleanup"`
}

type CaptureConfig struct {
	Load *CaptureLoadConfig `json:"load,omitempty" yaml:"load,omitempty"`
}

type CaptureLoadConfig struct {
	Filepath string `json:"filepath" yaml:"filepath"`
}

type AnalyzerConfig struct {
	// Name must match the Logic 2 analyzer UI name exactly (e.g. "SPI", "I2C", "Async Serial").
	Name string `json:"name" yaml:"name"`

	// Label is user-facing name for the analyzer and is used as the default ref for exports.
	Label string `json:"label,omitempty" yaml:"label,omitempty"`

	// Settings file path (optional). Only one of these may be provided.
	SettingsYAML string `json:"settings_yaml,omitempty" yaml:"settings_yaml,omitempty"`
	SettingsJSON string `json:"settings_json,omitempty" yaml:"settings_json,omitempty"`

	// Typed overrides (same shapes as the analyzer CLI).
	Set      []string `json:"set,omitempty" yaml:"set,omitempty"`
	SetBool  []string `json:"set_bool,omitempty" yaml:"set_bool,omitempty"`
	SetInt   []string `json:"set_int,omitempty" yaml:"set_int,omitempty"`
	SetFloat []string `json:"set_float,omitempty" yaml:"set_float,omitempty"`
}

type CleanupConfig struct {
	// CloseCapture closes the capture at the end of the run (best-effort).
	// Default: true.
	CloseCapture *bool `json:"close_capture,omitempty" yaml:"close_capture,omitempty"`
}

type ExportConfig struct {
	// Type is one of: raw-csv, raw-binary, table-csv
	Type string `json:"type" yaml:"type"`

	// raw-csv/raw-binary
	Directory             string   `json:"directory,omitempty" yaml:"directory,omitempty"`
	DigitalChannels       []uint32 `json:"digital,omitempty" yaml:"digital,omitempty"`
	AnalogChannels        []uint32 `json:"analog,omitempty" yaml:"analog,omitempty"`
	AnalogDownsampleRatio uint64   `json:"analog_downsample_ratio,omitempty" yaml:"analog_downsample_ratio,omitempty"`
	Iso8601Timestamp      bool     `json:"iso8601_timestamp,omitempty" yaml:"iso8601_timestamp,omitempty"`

	// table-csv
	Filepath  string             `json:"filepath,omitempty" yaml:"filepath,omitempty"`
	Analyzers []TableAnalyzerRef `json:"analyzers,omitempty" yaml:"analyzers,omitempty"`
	Columns   []string           `json:"columns,omitempty" yaml:"columns,omitempty"`
	Filter    *TableFilterConfig `json:"filter,omitempty" yaml:"filter,omitempty"`
}

type TableAnalyzerRef struct {
	// Ref references an analyzer created earlier in the pipeline (by label).
	Ref string `json:"ref" yaml:"ref"`
	// Radix is one of: hex|dec|bin|ascii
	Radix string `json:"radix" yaml:"radix"`
}

type TableFilterConfig struct {
	Query   string   `json:"query" yaml:"query"`
	Columns []string `json:"columns,omitempty" yaml:"columns,omitempty"`
}

func Load(path string) (*Config, error) {
	if path == "" {
		return nil, errors.New("pipeline config path is required")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "read pipeline config %s", path)
	}

	cfg := &Config{}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(b, cfg); err != nil {
			return nil, errors.Wrapf(err, "decode pipeline config yaml %s", path)
		}
	case ".json":
		if err := json.Unmarshal(b, cfg); err != nil {
			return nil, errors.Wrapf(err, "decode pipeline config json %s", path)
		}
	default:
		return nil, errors.Errorf("unsupported pipeline config extension %q (expected .yaml/.yml/.json)", ext)
	}

	if cfg.Version == 0 {
		cfg.Version = 1
	}
	if cfg.Version != 1 {
		return nil, errors.Errorf("unsupported pipeline config version %d", cfg.Version)
	}

	return cfg, nil
}

func pickBool(p *bool, fallback bool) bool {
	if p == nil {
		return fallback
	}
	return *p
}
