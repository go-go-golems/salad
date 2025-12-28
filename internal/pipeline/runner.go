package pipeline

import (
	"context"
	"strings"

	pb "github.com/go-go-golems/salad/gen/saleae/automation"
	saladconfig "github.com/go-go-golems/salad/internal/config"
	"github.com/go-go-golems/salad/internal/saleae"
	"github.com/pkg/errors"
)

type Runner struct {
	SaleaeConfig saleae.Config
}

type Result struct {
	CaptureID uint64
	Analyzers map[string]uint64 // label -> analyzer_id
	Artifacts []string          // file/directory paths written by exports (best-effort tracking)
}

func (r *Runner) Run(ctx context.Context, cfg *Config) (*Result, error) {
	if cfg == nil {
		return nil, errors.New("pipeline config is nil")
	}
	if cfg.Capture.Load == nil || strings.TrimSpace(cfg.Capture.Load.Filepath) == "" {
		return nil, errors.New("pipeline.capture.load.filepath is required (StartCapture is not implemented yet)")
	}

	c, err := saleae.New(ctx, r.SaleaeConfig)
	if err != nil {
		return nil, err
	}
	defer func() { _ = c.Close() }()

	res := &Result{
		Analyzers: make(map[string]uint64),
	}

	captureID, err := c.LoadCapture(ctx, cfg.Capture.Load.Filepath)
	if err != nil {
		return nil, err
	}
	res.CaptureID = captureID

	// Best-effort cleanup.
	defer func() {
		if pickBool(cfg.Cleanup.CloseCapture, true) && res.CaptureID != 0 {
			_ = c.CloseCapture(ctx, res.CaptureID)
		}
	}()

	// 1) Add analyzers (LLA only for now).
	for i, a := range cfg.Analyzers {
		name := strings.TrimSpace(a.Name)
		if name == "" {
			return nil, errors.Errorf("pipeline.analyzers[%d].name is required", i)
		}

		if a.SettingsJSON != "" && a.SettingsYAML != "" {
			return nil, errors.Errorf("pipeline.analyzers[%d]: only one of settings_yaml/settings_json may be set", i)
		}

		var settings map[string]*pb.AnalyzerSettingValue
		switch {
		case a.SettingsYAML != "":
			settings, err = saladconfig.LoadAnalyzerSettingsYAML(a.SettingsYAML)
		case a.SettingsJSON != "":
			settings, err = saladconfig.LoadAnalyzerSettingsJSON(a.SettingsJSON)
		default:
			settings = map[string]*pb.AnalyzerSettingValue{}
		}
		if err != nil {
			return nil, err
		}

		settings, err = saladconfig.ApplyAnalyzerSettingOverrides(settings, a.Set, a.SetBool, a.SetInt, a.SetFloat)
		if err != nil {
			return nil, err
		}

		label := strings.TrimSpace(a.Label)
		analyzerID, err := c.AddAnalyzer(ctx, res.CaptureID, name, label, settings)
		if err != nil {
			return nil, errors.Wrapf(err, "AddAnalyzer(name=%q,label=%q)", name, label)
		}

		ref := label
		if ref == "" {
			ref = name
		}
		if _, exists := res.Analyzers[ref]; exists {
			return nil, errors.Errorf("duplicate analyzer ref %q (labels must be unique)", ref)
		}
		res.Analyzers[ref] = analyzerID
	}

	// 2) Exports.
	for i, e := range cfg.Exports {
		switch strings.ToLower(strings.TrimSpace(e.Type)) {
		case "raw-csv":
			ch := &pb.LogicChannels{
				DigitalChannels: e.DigitalChannels,
				AnalogChannels:  e.AnalogChannels,
			}
			if len(ch.DigitalChannels) == 0 && len(ch.AnalogChannels) == 0 {
				return nil, errors.Errorf("pipeline.exports[%d] raw-csv: at least one of digital/analog must be set", i)
			}
			if err := c.ExportRawDataCsv(ctx, res.CaptureID, e.Directory, ch, e.AnalogDownsampleRatio, e.Iso8601Timestamp); err != nil {
				return nil, err
			}
			res.Artifacts = append(res.Artifacts, e.Directory)

		case "raw-binary":
			ch := &pb.LogicChannels{
				DigitalChannels: e.DigitalChannels,
				AnalogChannels:  e.AnalogChannels,
			}
			if len(ch.DigitalChannels) == 0 && len(ch.AnalogChannels) == 0 {
				return nil, errors.Errorf("pipeline.exports[%d] raw-binary: at least one of digital/analog must be set", i)
			}
			if err := c.ExportRawDataBinary(ctx, res.CaptureID, e.Directory, ch, e.AnalogDownsampleRatio); err != nil {
				return nil, err
			}
			res.Artifacts = append(res.Artifacts, e.Directory)

		case "table-csv":
			if strings.TrimSpace(e.Filepath) == "" {
				return nil, errors.Errorf("pipeline.exports[%d] table-csv: filepath is required", i)
			}
			if len(e.Analyzers) == 0 {
				return nil, errors.Errorf("pipeline.exports[%d] table-csv: analyzers is required", i)
			}

			analyzers, err := resolveTableAnalyzers(e.Analyzers, res.Analyzers)
			if err != nil {
				return nil, err
			}

			var filter *pb.DataTableFilter
			if e.Filter != nil && strings.TrimSpace(e.Filter.Query) != "" {
				filter = &pb.DataTableFilter{
					Query:   e.Filter.Query,
					Columns: e.Filter.Columns,
				}
			}

			if err := c.ExportDataTableCsv(ctx, res.CaptureID, e.Filepath, analyzers, e.Iso8601Timestamp, e.Columns, filter); err != nil {
				return nil, err
			}
			res.Artifacts = append(res.Artifacts, e.Filepath)

		default:
			return nil, errors.Errorf("pipeline.exports[%d]: unknown type %q (expected raw-csv|raw-binary|table-csv)", i, e.Type)
		}
	}

	return res, nil
}

func resolveTableAnalyzers(specs []TableAnalyzerRef, created map[string]uint64) ([]*pb.DataTableAnalyzerConfiguration, error) {
	out := make([]*pb.DataTableAnalyzerConfiguration, 0, len(specs))
	for i, s := range specs {
		ref := strings.TrimSpace(s.Ref)
		if ref == "" {
			return nil, errors.Errorf("table analyzers[%d].ref is required", i)
		}
		analyzerID, ok := created[ref]
		if !ok {
			return nil, errors.Errorf("table analyzers[%d]: unknown ref %q (no such analyzer label)", i, ref)
		}
		radix, err := parseRadixType(strings.TrimSpace(s.Radix))
		if err != nil {
			return nil, err
		}
		out = append(out, &pb.DataTableAnalyzerConfiguration{
			AnalyzerId: analyzerID,
			RadixType:  radix,
		})
	}
	return out, nil
}

func parseRadixType(s string) (pb.RadixType, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "hex":
		return pb.RadixType_RADIX_TYPE_HEXADECIMAL, nil
	case "dec":
		return pb.RadixType_RADIX_TYPE_DECIMAL, nil
	case "bin":
		return pb.RadixType_RADIX_TYPE_BINARY, nil
	case "ascii":
		return pb.RadixType_RADIX_TYPE_ASCII, nil
	default:
		return pb.RadixType_RADIX_TYPE_UNSPECIFIED, errors.Errorf("unknown radix %q (expected: hex|dec|bin|ascii)", s)
	}
}
