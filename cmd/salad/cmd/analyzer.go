package cmd

import (
	"context"
	"fmt"

	pb "github.com/go-go-golems/salad/gen/saleae/automation"
	saladconfig "github.com/go-go-golems/salad/internal/config"
	"github.com/go-go-golems/salad/internal/saleae"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var analyzerCmd = &cobra.Command{
	Use:   "analyzer",
	Short: "Analyzer operations (add/remove)",
}

var (
	analyzerCaptureID uint64
	analyzerID        uint64
	analyzerName      string
	analyzerLabel     string

	analyzerSettingsJSON string
	analyzerSettingsYAML string

	analyzerSet      []string
	analyzerSetBool  []string
	analyzerSetInt   []string
	analyzerSetFloat []string
)

var analyzerAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add an analyzer to a capture",
	RunE: func(cmd *cobra.Command, args []string) error {
		if analyzerSettingsJSON != "" && analyzerSettingsYAML != "" {
			return errors.New("only one of --settings-json or --settings-yaml may be specified")
		}

		ctx := cmd.Context()
		if timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}

		var settings map[string]*pb.AnalyzerSettingValue
		var err error
		switch {
		case analyzerSettingsJSON != "":
			settings, err = saladconfig.LoadAnalyzerSettingsJSON(analyzerSettingsJSON)
		case analyzerSettingsYAML != "":
			settings, err = saladconfig.LoadAnalyzerSettingsYAML(analyzerSettingsYAML)
		default:
			settings = map[string]*pb.AnalyzerSettingValue{}
		}
		if err != nil {
			return err
		}

		settings, err = saladconfig.ApplyAnalyzerSettingOverrides(settings, analyzerSet, analyzerSetBool, analyzerSetInt, analyzerSetFloat)
		if err != nil {
			return err
		}

		c, err := saleae.New(ctx, saleae.Config{Host: host, Port: port, Timeout: timeout})
		if err != nil {
			return err
		}
		defer func() { _ = c.Close() }()

		id, err := c.AddAnalyzer(ctx, analyzerCaptureID, analyzerName, analyzerLabel, settings)
		if err != nil {
			return errors.Wrapf(err, "add analyzer %q to capture %d", analyzerName, analyzerCaptureID)
		}

		_, err = fmt.Fprintf(cmd.OutOrStdout(), "analyzer_id=%d\n", id)
		return errors.Wrap(err, "write output")
	},
}

var analyzerRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove an analyzer from a capture",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}

		c, err := saleae.New(ctx, saleae.Config{Host: host, Port: port, Timeout: timeout})
		if err != nil {
			return err
		}
		defer func() { _ = c.Close() }()

		if err := c.RemoveAnalyzer(ctx, analyzerCaptureID, analyzerID); err != nil {
			return err
		}

		_, err = fmt.Fprintln(cmd.OutOrStdout(), "ok")
		return errors.Wrap(err, "write output")
	},
}

func init() {
	analyzerAddCmd.Flags().Uint64Var(&analyzerCaptureID, "capture-id", 0, "Capture ID")
	_ = analyzerAddCmd.MarkFlagRequired("capture-id")
	analyzerAddCmd.Flags().StringVar(&analyzerName, "name", "", "Analyzer name (exact UI name, e.g. \"SPI\", \"I2C\", \"Async Serial\")")
	_ = analyzerAddCmd.MarkFlagRequired("name")
	analyzerAddCmd.Flags().StringVar(&analyzerLabel, "label", "", "Analyzer label (user-facing name)")

	analyzerAddCmd.Flags().StringVar(&analyzerSettingsJSON, "settings-json", "", "Path to analyzer settings JSON file")
	analyzerAddCmd.Flags().StringVar(&analyzerSettingsYAML, "settings-yaml", "", "Path to analyzer settings YAML file")

	analyzerAddCmd.Flags().StringArrayVar(&analyzerSet, "set", nil, "Set string setting (key=value). Can be repeated.")
	analyzerAddCmd.Flags().StringArrayVar(&analyzerSetBool, "set-bool", nil, "Set bool setting (key=true/false). Can be repeated.")
	analyzerAddCmd.Flags().StringArrayVar(&analyzerSetInt, "set-int", nil, "Set int setting (key=123). Can be repeated.")
	analyzerAddCmd.Flags().StringArrayVar(&analyzerSetFloat, "set-float", nil, "Set float setting (key=12.34). Can be repeated.")

	analyzerRemoveCmd.Flags().Uint64Var(&analyzerCaptureID, "capture-id", 0, "Capture ID")
	_ = analyzerRemoveCmd.MarkFlagRequired("capture-id")
	analyzerRemoveCmd.Flags().Uint64Var(&analyzerID, "analyzer-id", 0, "Analyzer ID")
	_ = analyzerRemoveCmd.MarkFlagRequired("analyzer-id")

	analyzerCmd.AddCommand(analyzerAddCmd, analyzerRemoveCmd)
}
