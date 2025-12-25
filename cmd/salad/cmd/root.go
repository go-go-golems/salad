package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	host     string
	port     int
	timeout  time.Duration
	logLevel string
)

var rootCmd = &cobra.Command{
	Use:   "salad",
	Short: "Saleae Logic 2 Automation (gRPC) client",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		level, err := zerolog.ParseLevel(logLevel)
		if err != nil {
			return errors.Wrapf(err, "invalid --log-level %q", logLevel)
		}

		zerolog.TimeFieldFormat = time.RFC3339Nano
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339Nano}).Level(level)

		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&host, "host", "127.0.0.1", "Logic 2 automation host")
	rootCmd.PersistentFlags().IntVar(&port, "port", 10430, "Logic 2 automation port")
	rootCmd.PersistentFlags().DurationVar(&timeout, "timeout", 5*time.Second, "RPC timeout (also used for dialing if no context deadline is set)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Log level (trace,debug,info,warn,error,fatal,panic)")

	rootCmd.AddCommand(appinfoCmd)
	rootCmd.AddCommand(devicesCmd)
}
