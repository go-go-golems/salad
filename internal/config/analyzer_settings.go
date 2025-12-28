package config

import (
	"encoding/json"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	pb "github.com/go-go-golems/salad/gen/saleae/automation"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// LoadAnalyzerSettingsJSON loads analyzer settings from a JSON file.
// Supported shapes:
// - {"key": <scalar>, ...}
// - {"settings": {"key": <scalar>, ...}}
//
// Scalars can be: string, bool, integer, float.
func LoadAnalyzerSettingsJSON(path string) (map[string]*pb.AnalyzerSettingValue, error) {
	if path == "" {
		return map[string]*pb.AnalyzerSettingValue{}, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "read analyzer settings json %s", path)
	}

	var doc any
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, errors.Wrapf(err, "decode analyzer settings json %s", path)
	}

	m, err := normalizeSettingsDocument(doc)
	if err != nil {
		return nil, errors.Wrapf(err, "parse analyzer settings json %s", path)
	}
	return coerceAnalyzerSettings(m)
}

// LoadAnalyzerSettingsYAML loads analyzer settings from a YAML file.
// Supported shapes:
//   - key: <scalar>
//   - settings:
//     key: <scalar>
func LoadAnalyzerSettingsYAML(path string) (map[string]*pb.AnalyzerSettingValue, error) {
	if path == "" {
		return map[string]*pb.AnalyzerSettingValue{}, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrapf(err, "open analyzer settings yaml %s", path)
	}
	defer func() { _ = f.Close() }()

	decoder := yaml.NewDecoder(f)
	decoder.KnownFields(false) // settings files are free-form by design

	var doc any
	if err := decoder.Decode(&doc); err != nil {
		return nil, errors.Wrapf(err, "decode analyzer settings yaml %s", path)
	}

	m, err := normalizeSettingsDocument(doc)
	if err != nil {
		return nil, errors.Wrapf(err, "parse analyzer settings yaml %s", path)
	}
	return coerceAnalyzerSettings(m)
}

// LoadAnalyzerSettings loads analyzer settings from JSON or YAML based on file extension.
func LoadAnalyzerSettings(path string) (map[string]*pb.AnalyzerSettingValue, error) {
	if path == "" {
		return map[string]*pb.AnalyzerSettingValue{}, nil
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		return LoadAnalyzerSettingsJSON(path)
	case ".yaml", ".yml":
		return LoadAnalyzerSettingsYAML(path)
	default:
		return nil, errors.Errorf("unsupported analyzer settings file extension %q (expected .json/.yaml/.yml)", filepath.Ext(path))
	}
}

func normalizeSettingsDocument(doc any) (map[string]any, error) {
	m, ok := doc.(map[string]any)
	if !ok {
		// yaml may decode into map[interface{}]interface{} if we use yaml.Unmarshal directly,
		// but we decode via Decoder into 'any' which yields map[string]any in practice.
		return nil, errors.Errorf("expected mapping at top-level, got %T", doc)
	}
	if raw, ok := m["settings"]; ok {
		settings, ok := raw.(map[string]any)
		if !ok {
			return nil, errors.Errorf("expected settings to be a mapping, got %T", raw)
		}
		return settings, nil
	}
	return m, nil
}

func coerceAnalyzerSettings(m map[string]any) (map[string]*pb.AnalyzerSettingValue, error) {
	out := make(map[string]*pb.AnalyzerSettingValue, len(m))
	for k, v := range m {
		key := strings.TrimSpace(k)
		if key == "" {
			return nil, errors.New("settings contains empty key")
		}
		if v == nil {
			return nil, errors.Errorf("settings[%q] is null", key)
		}

		sv, err := toAnalyzerSettingValue(v)
		if err != nil {
			return nil, errors.Wrapf(err, "settings[%q]", key)
		}
		out[key] = sv
	}
	return out, nil
}

func toAnalyzerSettingValue(v any) (*pb.AnalyzerSettingValue, error) {
	switch t := v.(type) {
	case string:
		return &pb.AnalyzerSettingValue{Value: &pb.AnalyzerSettingValue_StringValue{StringValue: t}}, nil
	case bool:
		return &pb.AnalyzerSettingValue{Value: &pb.AnalyzerSettingValue_BoolValue{BoolValue: t}}, nil
	case int:
		return &pb.AnalyzerSettingValue{Value: &pb.AnalyzerSettingValue_Int64Value{Int64Value: int64(t)}}, nil
	case int64:
		return &pb.AnalyzerSettingValue{Value: &pb.AnalyzerSettingValue_Int64Value{Int64Value: t}}, nil
	case uint64:
		if t > uint64(math.MaxInt64) {
			return nil, errors.Errorf("integer %d overflows int64", t)
		}
		return &pb.AnalyzerSettingValue{Value: &pb.AnalyzerSettingValue_Int64Value{Int64Value: int64(t)}}, nil
	case float64:
		// JSON numbers come through as float64. If it’s integral, treat as int64.
		if math.IsNaN(t) || math.IsInf(t, 0) {
			return nil, errors.Errorf("invalid float %v", t)
		}
		if math.Trunc(t) == t && t >= float64(math.MinInt64) && t <= float64(math.MaxInt64) {
			return &pb.AnalyzerSettingValue{Value: &pb.AnalyzerSettingValue_Int64Value{Int64Value: int64(t)}}, nil
		}
		return &pb.AnalyzerSettingValue{Value: &pb.AnalyzerSettingValue_DoubleValue{DoubleValue: t}}, nil
	case float32:
		return &pb.AnalyzerSettingValue{Value: &pb.AnalyzerSettingValue_DoubleValue{DoubleValue: float64(t)}}, nil
	default:
		return nil, errors.Errorf("unsupported value type %T", v)
	}
}

type KVOverrideType int

const (
	OverrideString KVOverrideType = iota
	OverrideBool
	OverrideInt
	OverrideFloat
)

// ApplyAnalyzerSettingOverrides merges typed overrides into an existing settings map.
// Overrides win over values from files.
//
// Supported override formats:
// - string: "key=value"
// - bool:   "key=true"
// - int:    "key=123"
// - float:  "key=12.34"
func ApplyAnalyzerSettingOverrides(
	base map[string]*pb.AnalyzerSettingValue,
	stringOverrides []string,
	boolOverrides []string,
	intOverrides []string,
	floatOverrides []string,
) (map[string]*pb.AnalyzerSettingValue, error) {
	out := make(map[string]*pb.AnalyzerSettingValue, len(base))
	for k, v := range base {
		out[k] = v
	}

	if err := applyKVOverrides(out, OverrideString, stringOverrides); err != nil {
		return nil, err
	}
	if err := applyKVOverrides(out, OverrideBool, boolOverrides); err != nil {
		return nil, err
	}
	if err := applyKVOverrides(out, OverrideInt, intOverrides); err != nil {
		return nil, err
	}
	if err := applyKVOverrides(out, OverrideFloat, floatOverrides); err != nil {
		return nil, err
	}

	return out, nil
}

func applyKVOverrides(dst map[string]*pb.AnalyzerSettingValue, kind KVOverrideType, pairs []string) error {
	for _, raw := range pairs {
		k, v, ok := strings.Cut(raw, "=")
		if !ok {
			return errors.Errorf("invalid override %q (expected key=value)", raw)
		}
		key := strings.TrimSpace(k)
		if key == "" {
			return errors.Errorf("invalid override %q (empty key)", raw)
		}

		switch kind {
		case OverrideString:
			dst[key] = &pb.AnalyzerSettingValue{Value: &pb.AnalyzerSettingValue_StringValue{StringValue: v}}
		case OverrideBool:
			parsed, err := strconv.ParseBool(strings.TrimSpace(v))
			if err != nil {
				return errors.Wrapf(err, "parse bool override %q", raw)
			}
			dst[key] = &pb.AnalyzerSettingValue{Value: &pb.AnalyzerSettingValue_BoolValue{BoolValue: parsed}}
		case OverrideInt:
			parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
			if err != nil {
				return errors.Wrapf(err, "parse int override %q", raw)
			}
			dst[key] = &pb.AnalyzerSettingValue{Value: &pb.AnalyzerSettingValue_Int64Value{Int64Value: parsed}}
		case OverrideFloat:
			parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
			if err != nil {
				return errors.Wrapf(err, "parse float override %q", raw)
			}
			dst[key] = &pb.AnalyzerSettingValue{Value: &pb.AnalyzerSettingValue_DoubleValue{DoubleValue: parsed}}
		default:
			return errors.Errorf("unknown override kind %d", kind)
		}
	}
	return nil
}

// LoadAnalyzerSettingsFromReader is a small helper for tests.
func LoadAnalyzerSettingsFromReader(r io.Reader, format string) (map[string]*pb.AnalyzerSettingValue, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "json":
		var doc any
		if err := json.NewDecoder(r).Decode(&doc); err != nil {
			return nil, errors.Wrap(err, "decode analyzer settings json")
		}
		m, err := normalizeSettingsDocument(doc)
		if err != nil {
			return nil, err
		}
		return coerceAnalyzerSettings(m)
	case "yaml":
		decoder := yaml.NewDecoder(r)
		var doc any
		if err := decoder.Decode(&doc); err != nil {
			return nil, errors.Wrap(err, "decode analyzer settings yaml")
		}
		m, err := normalizeSettingsDocument(doc)
		if err != nil {
			return nil, err
		}
		return coerceAnalyzerSettings(m)
	default:
		return nil, errors.Errorf("unknown analyzer settings format %q", format)
	}
}
