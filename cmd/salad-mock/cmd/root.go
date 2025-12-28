package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	mock "github.com/go-go-golems/salad/internal/mock/saleae"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	configPath string
	host       string
	port       int
	logLevel   string
)

var rootCmd = &cobra.Command{
	Use:   "salad-mock",
	Short: "Mock Saleae Logic 2 Automation (gRPC) server",
	RunE: func(cmd *cobra.Command, args []string) error {
		level, err := zerolog.ParseLevel(logLevel)
		if err != nil {
			return errors.Wrapf(err, "invalid --log-level %q", logLevel)
		}

		zerolog.TimeFieldFormat = time.RFC3339Nano
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339Nano}).Level(level)

		cfg, err := mock.LoadConfig(configPath)
		if err != nil {
			return err
		}
		plan, err := mock.Compile(cfg)
		if err != nil {
			return err
		}

		server := mock.NewServer(plan)
		addr := fmt.Sprintf("%s:%d", host, port)
		grpcServer, listener, err := server.Start(addr)
		if err != nil {
			return err
		}
		log.Info().Str("addr", addr).Msg("mock server started")

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		<-ctx.Done()

		log.Info().Msg("shutting down mock server")
		grpcServer.Stop()
		_ = listener.Close()
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
	rootCmd.Flags().StringVar(&configPath, "config", "", "Path to mock server YAML config")
	rootCmd.Flags().StringVar(&host, "host", "127.0.0.1", "Bind host for mock server")
	rootCmd.Flags().IntVar(&port, "port", 10431, "Bind port for mock server")
	rootCmd.Flags().StringVar(&logLevel, "log-level", "info", "Log level (trace,debug,info,warn,error,fatal,panic)")

	_ = rootCmd.MarkFlagRequired("config")
}
