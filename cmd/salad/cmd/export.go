package cmd

import (
	"context"
	"fmt"

	pb "github.com/go-go-golems/salad/gen/saleae/automation"
	"github.com/go-go-golems/salad/internal/saleae"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export data from captures",
}

var (
	exportCaptureID          uint64
	exportDirectory          string
	exportDigitalChannelsCSV string
	exportAnalogChannelsCSV  string
	exportAnalogDownsample   uint64
	exportIso8601Timestamps  bool
)

func makeLogicChannelsFromFlags() (*pb.LogicChannels, error) {
	digital, err := parseUint32CSV(exportDigitalChannelsCSV)
	if err != nil {
		return nil, err
	}
	analog, err := parseUint32CSV(exportAnalogChannelsCSV)
	if err != nil {
		return nil, err
	}
	if len(digital) == 0 && len(analog) == 0 {
		return nil, errors.New("at least one channel must be specified via --digital or --analog")
	}
	return &pb.LogicChannels{
		DigitalChannels: digital,
		AnalogChannels:  analog,
	}, nil
}

var exportRawCsvCmd = &cobra.Command{
	Use:   "raw-csv",
	Short: "Export raw data to CSV files",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}

		ch, err := makeLogicChannelsFromFlags()
		if err != nil {
			return err
		}

		c, err := saleae.New(ctx, saleae.Config{Host: host, Port: port, Timeout: timeout})
		if err != nil {
			return err
		}
		defer func() { _ = c.Close() }()

		if err := c.ExportRawDataCsv(ctx, exportCaptureID, exportDirectory, ch, exportAnalogDownsample, exportIso8601Timestamps); err != nil {
			return err
		}

		_, err = fmt.Fprintln(cmd.OutOrStdout(), "ok")
		return errors.Wrap(err, "write output")
	},
}

var exportRawBinaryCmd = &cobra.Command{
	Use:   "raw-binary",
	Short: "Export raw data to binary files",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}

		ch, err := makeLogicChannelsFromFlags()
		if err != nil {
			return err
		}

		c, err := saleae.New(ctx, saleae.Config{Host: host, Port: port, Timeout: timeout})
		if err != nil {
			return err
		}
		defer func() { _ = c.Close() }()

		if err := c.ExportRawDataBinary(ctx, exportCaptureID, exportDirectory, ch, exportAnalogDownsample); err != nil {
			return err
		}

		_, err = fmt.Fprintln(cmd.OutOrStdout(), "ok")
		return errors.Wrap(err, "write output")
	},
}

func init() {
	exportRawCsvCmd.Flags().Uint64Var(&exportCaptureID, "capture-id", 0, "Capture ID")
	_ = exportRawCsvCmd.MarkFlagRequired("capture-id")
	exportRawCsvCmd.Flags().StringVar(&exportDirectory, "directory", "", "Directory to write exported files into")
	_ = exportRawCsvCmd.MarkFlagRequired("directory")
	exportRawCsvCmd.Flags().StringVar(&exportDigitalChannelsCSV, "digital", "", "Digital channels to export (comma-separated, e.g. \"0,1,2\")")
	exportRawCsvCmd.Flags().StringVar(&exportAnalogChannelsCSV, "analog", "", "Analog channels to export (comma-separated, e.g. \"0,1\")")
	exportRawCsvCmd.Flags().Uint64Var(&exportAnalogDownsample, "analog-downsample-ratio", 1, "Analog downsample ratio (1..1,000,000)")
	exportRawCsvCmd.Flags().BoolVar(&exportIso8601Timestamps, "iso8601-timestamp", false, "Use ISO8601 timestamps in CSV export")

	exportRawBinaryCmd.Flags().Uint64Var(&exportCaptureID, "capture-id", 0, "Capture ID")
	_ = exportRawBinaryCmd.MarkFlagRequired("capture-id")
	exportRawBinaryCmd.Flags().StringVar(&exportDirectory, "directory", "", "Directory to write exported files into")
	_ = exportRawBinaryCmd.MarkFlagRequired("directory")
	exportRawBinaryCmd.Flags().StringVar(&exportDigitalChannelsCSV, "digital", "", "Digital channels to export (comma-separated, e.g. \"0,1,2\")")
	exportRawBinaryCmd.Flags().StringVar(&exportAnalogChannelsCSV, "analog", "", "Analog channels to export (comma-separated, e.g. \"0,1\")")
	exportRawBinaryCmd.Flags().Uint64Var(&exportAnalogDownsample, "analog-downsample-ratio", 1, "Analog downsample ratio (1..1,000,000)")

	exportCmd.AddCommand(exportRawCsvCmd, exportRawBinaryCmd)
}
