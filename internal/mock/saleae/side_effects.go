package saleae

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	pb "github.com/go-go-golems/salad/gen/saleae/automation"
	"github.com/pkg/errors"
)

type SideEffects interface {
	SaveCapture(filepath string, captureID uint64, payload []byte) error
	ExportRawCSV(directory string, req *pb.ExportRawDataCsvRequest, opts ExportCSVOptions) error
	ExportRawBinary(directory string, req *pb.ExportRawDataBinaryRequest, opts ExportBinaryOptions) error
	ExportDataTableCSV(filepath string, req *pb.ExportDataTableCsvRequest, opts ExportDataTableCSVOptions) error
}

type ExportCSVOptions struct {
	WriteDigital             bool
	WriteAnalog              bool
	DigitalFilename          string
	AnalogFilename           string
	IncludeRequestedChannels bool
}

type ExportBinaryOptions struct {
	WriteDigital    bool
	WriteAnalog     bool
	DigitalFilename string
	AnalogFilename  string
}

type ExportDataTableCSVOptions struct {
	IncludeRequest bool
}

type NoopSideEffects struct{}

func (NoopSideEffects) SaveCapture(string, uint64, []byte) error {
	return nil
}

func (NoopSideEffects) ExportRawCSV(string, *pb.ExportRawDataCsvRequest, ExportCSVOptions) error {
	return nil
}

func (NoopSideEffects) ExportRawBinary(string, *pb.ExportRawDataBinaryRequest, ExportBinaryOptions) error {
	return nil
}

func (NoopSideEffects) ExportDataTableCSV(string, *pb.ExportDataTableCsvRequest, ExportDataTableCSVOptions) error {
	return nil
}

type FileSideEffects struct{}

func (FileSideEffects) SaveCapture(path string, captureID uint64, payload []byte) error {
	if path == "" {
		return errors.New("save capture path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return errors.Wrapf(err, "create save capture directory for %s", path)
	}
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return errors.Wrapf(err, "write save capture placeholder for %s", path)
	}
	return nil
}

func (FileSideEffects) ExportRawCSV(directory string, req *pb.ExportRawDataCsvRequest, opts ExportCSVOptions) error {
	if directory == "" {
		return errors.New("export directory is required")
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return errors.Wrapf(err, "create export directory %s", directory)
	}

	if opts.WriteDigital {
		path := filepath.Join(directory, opts.DigitalFilename)
		payload := buildCSVPlaceholder("digital", req, opts.IncludeRequestedChannels)
		if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
			return errors.Wrapf(err, "write digital csv placeholder %s", path)
		}
	}
	if opts.WriteAnalog {
		path := filepath.Join(directory, opts.AnalogFilename)
		payload := buildCSVPlaceholder("analog", req, opts.IncludeRequestedChannels)
		if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
			return errors.Wrapf(err, "write analog csv placeholder %s", path)
		}
	}
	return nil
}

func (FileSideEffects) ExportRawBinary(directory string, req *pb.ExportRawDataBinaryRequest, opts ExportBinaryOptions) error {
	if directory == "" {
		return errors.New("export directory is required")
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return errors.Wrapf(err, "create export directory %s", directory)
	}

	if opts.WriteDigital {
		path := filepath.Join(directory, opts.DigitalFilename)
		payload := fmt.Sprintf("SALAD_MOCK_DIGITAL_BIN capture_id=%d\n", req.GetCaptureId())
		if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
			return errors.Wrapf(err, "write digital bin placeholder %s", path)
		}
	}
	if opts.WriteAnalog {
		path := filepath.Join(directory, opts.AnalogFilename)
		payload := fmt.Sprintf("SALAD_MOCK_ANALOG_BIN capture_id=%d\n", req.GetCaptureId())
		if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
			return errors.Wrapf(err, "write analog bin placeholder %s", path)
		}
		return nil
	}
	return nil
}

func (FileSideEffects) ExportDataTableCSV(path string, req *pb.ExportDataTableCsvRequest, opts ExportDataTableCSVOptions) error {
	if path == "" {
		return errors.New("export filepath is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return errors.Wrapf(err, "create export directory for %s", path)
	}
	payload := buildDataTableCSVPlaceholder(req, opts.IncludeRequest)
	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		return errors.Wrapf(err, "write data table csv placeholder %s", path)
	}
	return nil
}

func buildCSVPlaceholder(kind string, req *pb.ExportRawDataCsvRequest, includeChannels bool) string {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("SALAD_MOCK_%s_CSV capture_id=%d\n", strings.ToUpper(kind), req.GetCaptureId()))
	if !includeChannels {
		return builder.String()
	}
	channels := req.GetLogicChannels()
	if channels == nil {
		return builder.String()
	}
	if len(channels.GetDigitalChannels()) > 0 {
		builder.WriteString(fmt.Sprintf("digital=%v\n", channels.GetDigitalChannels()))
	}
	if len(channels.GetAnalogChannels()) > 0 {
		builder.WriteString(fmt.Sprintf("analog=%v\n", channels.GetAnalogChannels()))
	}
	return builder.String()
}

func buildDataTableCSVPlaceholder(req *pb.ExportDataTableCsvRequest, includeRequest bool) string {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("SALAD_MOCK_DATA_TABLE_CSV capture_id=%d\n", req.GetCaptureId()))
	if !includeRequest {
		return builder.String()
	}

	builder.WriteString(fmt.Sprintf("iso8601_timestamp=%v\n", req.GetIso8601Timestamp()))
	if len(req.GetExportColumns()) > 0 {
		builder.WriteString(fmt.Sprintf("export_columns=%v\n", req.GetExportColumns()))
	}
	if len(req.GetAnalyzers()) > 0 {
		pairs := make([]string, 0, len(req.GetAnalyzers()))
		for _, a := range req.GetAnalyzers() {
			if a == nil {
				continue
			}
			pairs = append(pairs, fmt.Sprintf("%d:%s", a.GetAnalyzerId(), a.GetRadixType().String()))
		}
		builder.WriteString(fmt.Sprintf("analyzers=%v\n", pairs))
	}
	if f := req.GetFilter(); f != nil {
		if f.GetQuery() != "" {
			builder.WriteString(fmt.Sprintf("filter.query=%s\n", f.GetQuery()))
		}
		if len(f.GetColumns()) > 0 {
			builder.WriteString(fmt.Sprintf("filter.columns=%v\n", f.GetColumns()))
		}
	}
	return builder.String()
}
