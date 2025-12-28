package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	pb "github.com/go-go-golems/salad/gen/saleae/automation"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	var (
		host    = flag.String("host", "127.0.0.1", "Logic 2 automation host")
		port    = flag.Int("port", 10430, "Logic 2 automation port")
		timeout = flag.Duration("timeout", 5*time.Second, "Dial/RPC timeout")

		deviceID = flag.String("device-id", "", "Optional explicit device_id (default: first physical device)")

		bufferMB  = flag.Uint64("buffer-mb", 16, "Capture buffer size in megabytes")
		digitalHz = flag.Uint64("digital-hz", 1_000_000, "Digital sample rate (Hz)")
		digital   = flag.String("digital", "0,1,2,3", "Digital channels to enable (comma-separated, e.g. 0,1,2,3)")
	)
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	addr := fmt.Sprintf("%s:%d", *host, *port)
	conn, err := dialAndWaitReady(ctx, addr)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "dial: %v\n", err)
		os.Exit(2)
	}
	defer func() { _ = conn.Close() }()

	client := pb.NewManagerClient(conn)

	dev := *deviceID
	if dev == "" {
		reply, err := client.GetDevices(ctx, &pb.GetDevicesRequest{IncludeSimulationDevices: false})
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "GetDevices RPC: %v\n", err)
			os.Exit(2)
		}
		for _, d := range reply.GetDevices() {
			if d == nil {
				continue
			}
			if d.GetIsSimulation() {
				continue
			}
			dev = d.GetDeviceId()
			break
		}
		if dev == "" {
			_, _ = fmt.Fprintf(os.Stderr, "no physical devices found\n")
			os.Exit(2)
		}
	}

	req := &pb.StartCaptureRequest{
		DeviceId: dev,
		DeviceConfiguration: &pb.StartCaptureRequest_LogicDeviceConfiguration{
			LogicDeviceConfiguration: &pb.LogicDeviceConfiguration{
				EnabledChannels: &pb.LogicDeviceConfiguration_LogicChannels{
					LogicChannels: &pb.LogicChannels{
						DigitalChannels: nil,
					},
				},
				DigitalSampleRate: uint32(*digitalHz),
			},
		},
		CaptureConfiguration: &pb.CaptureConfiguration{
			BufferSizeMegabytes: uint32(*bufferMB),
			CaptureMode: &pb.CaptureConfiguration_ManualCaptureMode{
				ManualCaptureMode: &pb.ManualCaptureMode{
					TrimDataSeconds: 0,
				},
			},
		},
	}

	digitalChannels, err := parseUint32CSV(*digital)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "parse --digital: %v\n", err)
		os.Exit(2)
	}
	if len(digitalChannels) == 0 {
		_, _ = fmt.Fprintf(os.Stderr, "at least one digital channel must be specified via --digital\n")
		os.Exit(2)
	}
	req.GetLogicDeviceConfiguration().GetLogicChannels().DigitalChannels = digitalChannels

	reply, err := client.StartCapture(ctx, req)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "StartCapture RPC: %v\n", err)
		os.Exit(2)
	}
	if reply.GetCaptureInfo() == nil {
		_, _ = fmt.Fprintf(os.Stderr, "StartCapture: reply.capture_info is nil\n")
		os.Exit(2)
	}

	_, _ = fmt.Fprintf(os.Stdout, "capture_id=%d\n", reply.GetCaptureInfo().GetCaptureId())
}

func dialAndWaitReady(ctx context.Context, addr string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, errors.Wrapf(err, "grpc.NewClient(%s)", addr)
	}

	conn.Connect()
	for {
		state := conn.GetState()
		if state == connectivity.Ready {
			return conn, nil
		}
		if state == connectivity.Shutdown {
			_ = conn.Close()
			return nil, errors.Errorf("grpc connection shutdown while connecting to %s", addr)
		}
		if !conn.WaitForStateChange(ctx, state) {
			_ = conn.Close()
			return nil, errors.Wrapf(ctx.Err(), "connect to %s", addr)
		}
	}
}

func parseUint32CSV(s string) ([]uint32, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	parts := strings.Split(s, ",")
	out := make([]uint32, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		v, err := strconv.ParseUint(part, 10, 32)
		if err != nil {
			return nil, errors.Wrapf(err, "parse uint32 %q", part)
		}
		out = append(out, uint32(v))
	}
	return out, nil
}
