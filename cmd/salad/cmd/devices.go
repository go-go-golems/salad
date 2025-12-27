package cmd

import (
	"context"
	"fmt"

	"github.com/go-go-golems/salad/internal/saleae"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var includeSimulationDevices bool

var devicesCmd = &cobra.Command{
	Use:   "devices",
	Short: "List connected devices",
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
		defer func() { _ = c.Close() }()

		devices, err := c.GetDevices(ctx, includeSimulationDevices)
		if err != nil {
			return err
		}

		for _, d := range devices {
			_, err := fmt.Fprintf(
				cmd.OutOrStdout(),
				"device_id=%s device_type=%s is_simulation=%t\n",
				d.GetDeviceId(),
				d.GetDeviceType().String(),
				d.GetIsSimulation(),
			)
			if err != nil {
				return errors.Wrap(err, "write output")
			}
		}

		return nil
	},
}

func init() {
	devicesCmd.Flags().BoolVar(&includeSimulationDevices, "include-simulation-devices", false, "Include simulation devices in the response")
}
