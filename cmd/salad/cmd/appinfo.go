package cmd

import (
	"context"
	"fmt"

	"github.com/go-go-golems/salad/internal/saleae"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var appinfoCmd = &cobra.Command{
	Use:   "appinfo",
	Short: "Print Logic 2 application + API version information",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}

		c, err := saleae.New(ctx, saleae.Config{
			Host:    host,
			Port:    port,
			Timeout: timeout,
		})
		if err != nil {
			return err
		}
		defer c.Close()

		info, err := c.GetAppInfo(ctx)
		if err != nil {
			return err
		}

		api := info.GetApiVersion()

		_, err = fmt.Fprintf(
			cmd.OutOrStdout(),
			"application_version=%s\napi_version=%d.%d.%d\nlaunch_pid=%d\n",
			info.GetApplicationVersion(),
			api.GetMajor(), api.GetMinor(), api.GetPatch(),
			info.GetLaunchPid(),
		)
		return errors.Wrap(err, "write output")
	},
}
