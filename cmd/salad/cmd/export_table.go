package cmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	pb "github.com/go-go-golems/salad/gen/saleae/automation"
	"github.com/go-go-golems/salad/internal/saleae"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	exportTableCaptureID        uint64
	exportTableFilepath         string
	exportTableAnalyzers        []string
	exportTableIso8601Timestamp bool

	exportTableColumnsCSV       string
	exportTableFilterQuery      string
	exportTableFilterColumnsCSV string
)

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

func parseDataTableAnalyzerSelectors(selectors []string) ([]*pb.DataTableAnalyzerConfiguration, error) {
	out := make([]*pb.DataTableAnalyzerConfiguration, 0, len(selectors))
	for _, sel := range selectors {
		sel = strings.TrimSpace(sel)
		if sel == "" {
			continue
		}
		idStr, radixStr, ok := strings.Cut(sel, ":")
		if !ok {
			return nil, errors.Errorf("invalid --analyzer %q (expected <id>:<radix>, e.g. 10025:hex)", sel)
		}
		idStr = strings.TrimSpace(idStr)
		radixStr = strings.TrimSpace(radixStr)
		if idStr == "" || radixStr == "" {
			return nil, errors.Errorf("invalid --analyzer %q (expected <id>:<radix>, e.g. 10025:hex)", sel)
		}
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "parse analyzer id %q", idStr)
		}
		if id == 0 {
			return nil, errors.Errorf("analyzer id must be non-zero (got %q)", idStr)
		}
		radixType, err := parseRadixType(radixStr)
		if err != nil {
			return nil, err
		}
		out = append(out, &pb.DataTableAnalyzerConfiguration{
			AnalyzerId: id,
			RadixType:  radixType,
		})
	}
	return out, nil
}

var exportTableCmd = &cobra.Command{
	Use:   "table",
	Short: "Export decoded analyzer data tables to a CSV file",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}

		if len(exportTableAnalyzers) == 0 {
			return errors.New("at least one analyzer must be specified via --analyzer <id>:<radix>")
		}
		analyzers, err := parseDataTableAnalyzerSelectors(exportTableAnalyzers)
		if err != nil {
			return err
		}
		if len(analyzers) == 0 {
			return errors.New("at least one analyzer must be specified via --analyzer <id>:<radix>")
		}

		exportColumns := parseStringCSV(exportTableColumnsCSV)
		filterColumns := parseStringCSV(exportTableFilterColumnsCSV)
		if exportTableFilterQuery == "" && len(filterColumns) > 0 {
			return errors.New("--filter-columns requires --filter-query")
		}

		var filter *pb.DataTableFilter
		if exportTableFilterQuery != "" {
			filter = &pb.DataTableFilter{
				Query:   exportTableFilterQuery,
				Columns: filterColumns,
			}
		}

		c, err := saleae.New(ctx, saleae.Config{Host: host, Port: port, Timeout: timeout})
		if err != nil {
			return err
		}
		defer func() { _ = c.Close() }()

		if err := c.ExportDataTableCsv(
			ctx,
			exportTableCaptureID,
			exportTableFilepath,
			analyzers,
			exportTableIso8601Timestamp,
			exportColumns,
			filter,
		); err != nil {
			return err
		}

		_, err = fmt.Fprintln(cmd.OutOrStdout(), "ok")
		return errors.Wrap(err, "write output")
	},
}

func init() {
	exportTableCmd.Flags().Uint64Var(&exportTableCaptureID, "capture-id", 0, "Capture ID")
	_ = exportTableCmd.MarkFlagRequired("capture-id")
	exportTableCmd.Flags().StringVar(&exportTableFilepath, "filepath", "", "Path to write exported CSV file to")
	_ = exportTableCmd.MarkFlagRequired("filepath")
	exportTableCmd.Flags().StringArrayVar(&exportTableAnalyzers, "analyzer", nil, "Analyzer selector (<id>:<radix>, radix: hex|dec|bin|ascii). Can be repeated.")
	_ = exportTableCmd.MarkFlagRequired("analyzer")

	exportTableCmd.Flags().BoolVar(&exportTableIso8601Timestamp, "iso8601-timestamp", false, "Use ISO8601 timestamps in CSV export")
	exportTableCmd.Flags().StringVar(&exportTableColumnsCSV, "columns", "", "Columns to export (comma-separated). If empty, all columns are exported.")
	exportTableCmd.Flags().StringVar(&exportTableFilterQuery, "filter-query", "", "Query to filter data table rows")
	exportTableCmd.Flags().StringVar(&exportTableFilterColumnsCSV, "filter-columns", "", "Columns to apply the filter query to (comma-separated). If empty, all columns are searched.")
}
