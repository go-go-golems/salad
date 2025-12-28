package cmd

import (
	"context"
	"fmt"

	"github.com/go-go-golems/salad/internal/saleae"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var captureCmd = &cobra.Command{
	Use:   "capture",
	Short: "Capture operations (load/save/stop/wait/close)",
}

var (
	captureID uint64
	filepath  string
)

var captureLoadCmd = &cobra.Command{
	Use:   "load",
	Short: "Load a .sal capture file",
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

		id, err := c.LoadCapture(ctx, filepath)
		if err != nil {
			return err
		}

		_, err = fmt.Fprintf(cmd.OutOrStdout(), "capture_id=%d\n", id)
		return errors.Wrap(err, "write output")
	},
}

var captureSaveCmd = &cobra.Command{
	Use:   "save",
	Short: "Save a capture to a .sal file",
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

		if err := c.SaveCapture(ctx, captureID, filepath); err != nil {
			return err
		}

		_, err = fmt.Fprintln(cmd.OutOrStdout(), "ok")
		return errors.Wrap(err, "write output")
	},
}

var captureStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop an active capture",
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

		if err := c.StopCapture(ctx, captureID); err != nil {
			return err
		}

		_, err = fmt.Fprintln(cmd.OutOrStdout(), "ok")
		return errors.Wrap(err, "write output")
	},
}

var captureWaitCmd = &cobra.Command{
	Use:   "wait",
	Short: "Wait for a capture to complete (not for manual capture mode)",
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

		if err := c.WaitCapture(ctx, captureID); err != nil {
			return err
		}

		_, err = fmt.Fprintln(cmd.OutOrStdout(), "ok")
		return errors.Wrap(err, "write output")
	},
}

var captureCloseCmd = &cobra.Command{
	Use:   "close",
	Short: "Close a capture to release resources",
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

		if err := c.CloseCapture(ctx, captureID); err != nil {
			return err
		}

		_, err = fmt.Fprintln(cmd.OutOrStdout(), "ok")
		return errors.Wrap(err, "write output")
	},
}

func init() {
	captureLoadCmd.Flags().StringVar(&filepath, "filepath", "", "Absolute filepath of Logic 2 .sal file to load")
	_ = captureLoadCmd.MarkFlagRequired("filepath")

	captureSaveCmd.Flags().Uint64Var(&captureID, "capture-id", 0, "Capture ID")
	_ = captureSaveCmd.MarkFlagRequired("capture-id")
	captureSaveCmd.Flags().StringVar(&filepath, "filepath", "", "Absolute filepath to save the .sal file to")
	_ = captureSaveCmd.MarkFlagRequired("filepath")

	captureStopCmd.Flags().Uint64Var(&captureID, "capture-id", 0, "Capture ID")
	_ = captureStopCmd.MarkFlagRequired("capture-id")

	captureWaitCmd.Flags().Uint64Var(&captureID, "capture-id", 0, "Capture ID")
	_ = captureWaitCmd.MarkFlagRequired("capture-id")

	captureCloseCmd.Flags().Uint64Var(&captureID, "capture-id", 0, "Capture ID")
	_ = captureCloseCmd.MarkFlagRequired("capture-id")

	captureCmd.AddCommand(captureLoadCmd, captureSaveCmd, captureStopCmd, captureWaitCmd, captureCloseCmd)
}
