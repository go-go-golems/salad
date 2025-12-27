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
