package cmd

import (
	"context"
	"fmt"

	"github.com/go-go-golems/salad/internal/pipeline"
	"github.com/go-go-golems/salad/internal/saleae"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	runConfigPath string
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a pipeline from a YAML/JSON config file",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}

		cfg, err := pipeline.Load(runConfigPath)
		if err != nil {
			return err
		}

		r := &pipeline.Runner{
			SaleaeConfig: saleae.Config{Host: host, Port: port, Timeout: timeout},
		}
		res, err := r.Run(ctx, cfg)
		if err != nil {
			return err
		}

		// Keep output simple + greppable for now (ticket 007 will unify structured output).
		if res.CaptureID != 0 {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "capture_id=%d\n", res.CaptureID)
		}
		for label, id := range res.Analyzers {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "analyzer_id=%d label=%q\n", id, label)
		}
		for _, path := range res.Artifacts {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "artifact=%s\n", path)
		}

		_, err = fmt.Fprintln(cmd.OutOrStdout(), "ok")
		return errors.Wrap(err, "write output")
	},
}

func init() {
	runCmd.Flags().StringVar(&runConfigPath, "config", "", "Pipeline config file (.yaml/.yml/.json)")
	_ = runCmd.MarkFlagRequired("config")
}
