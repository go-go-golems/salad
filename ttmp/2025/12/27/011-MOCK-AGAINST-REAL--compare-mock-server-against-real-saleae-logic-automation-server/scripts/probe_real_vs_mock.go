package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	pb "github.com/go-go-golems/salad/gen/saleae/automation"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type DeviceSummary struct {
	DeviceID     string `json:"device_id"`
	DeviceType   string `json:"device_type"`
	IsSimulation bool   `json:"is_simulation"`
}

type AppInfoSummary struct {
	ApplicationVersion string `json:"application_version"`
	APIVersion         string `json:"api_version"`
	LaunchPID          uint64 `json:"launch_pid"`
}

type RPCErrorSummary struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type EndpointReport struct {
	Name       string          `json:"name"`
	Addr       string          `json:"addr"`
	AppInfo    *AppInfoSummary `json:"appinfo,omitempty"`
	DevicesNo  []DeviceSummary `json:"devices_no_sim,omitempty"`
	DevicesYes []DeviceSummary `json:"devices_with_sim,omitempty"`

	WaitCaptureUnknown     *RPCErrorSummary `json:"wait_capture_unknown,omitempty"`
	StopCaptureUnknown     *RPCErrorSummary `json:"stop_capture_unknown,omitempty"`
	CloseCaptureUnknown    *RPCErrorSummary `json:"close_capture_unknown,omitempty"`
	LoadCaptureEmptyPath   *RPCErrorSummary `json:"load_capture_empty_path,omitempty"`
	LoadCaptureMissingPath *RPCErrorSummary `json:"load_capture_missing_path,omitempty"`
}

type Diff struct {
	Path string `json:"path"`
	Real any    `json:"real"`
	Mock any    `json:"mock"`
}

func main() {
	var (
		realHost = flag.String("real-host", "127.0.0.1", "Real Logic 2 automation host")
		realPort = flag.Int("real-port", 10430, "Real Logic 2 automation port")
		mockHost = flag.String("mock-host", "127.0.0.1", "Mock server host")
		mockPort = flag.Int("mock-port", 10431, "Mock server port")
		timeout  = flag.Duration("timeout", 5*time.Second, "Dial/RPC timeout")

		outPath = flag.String("out", "", "Optional remember-to-diff output file path")

		ignoreLaunchPID  = flag.Bool("ignore-launch-pid", true, "Ignore appinfo.launch_pid differences when diffing")
		ignoreAppVersion = flag.Bool("ignore-app-version", true, "Ignore appinfo.application_version differences when diffing")
		diffDevices      = flag.Bool("diff-devices", false, "Diff device lists (often expected to differ real vs mock)")
		diffMessages     = flag.Bool("diff-messages", false, "Diff gRPC error messages (codes are always compared)")
		failOnDiff       = flag.Bool("fail-on-diff", false, "Exit non-zero if any diffs are found")

		missingFilepath = flag.String("missing-filepath", "/this/does/not/exist.sal", "Path used for LoadCapture missing-file probe (should not exist)")
	)
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	realReport, err := probeEndpoint(ctx, "real", *realHost, *realPort, *missingFilepath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "probe real endpoint: %v\n", err)
		os.Exit(2)
	}

	mock, err := probeEndpoint(ctx, "mock", *mockHost, *mockPort, *missingFilepath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "probe mock endpoint: %v\n", err)
		os.Exit(2)
	}

	diffs := diffReports(realReport, mock, diffOptions{
		ignoreLaunchPID:  *ignoreLaunchPID,
		ignoreAppVersion: *ignoreAppVersion,
		diffDevices:      *diffDevices,
		diffMessages:     *diffMessages,
	})

	report := map[string]any{
		"generated_at": time.Now().Format(time.RFC3339Nano),
		"real":         realReport,
		"mock":         mock,
		"diffs":        diffs,
	}

	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "marshal report: %v\n", err)
		os.Exit(2)
	}

	_, _ = os.Stdout.Write(append(b, '\n'))

	if *outPath != "" {
		if err := os.WriteFile(*outPath, append(b, '\n'), 0o644); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "write --out file: %v\n", err)
			os.Exit(2)
		}
	}

	if *failOnDiff && len(diffs) > 0 {
		os.Exit(1)
	}
}

func probeEndpoint(ctx context.Context, name, host string, port int, missingFilepath string) (*EndpointReport, error) {
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := dialAndWaitReady(ctx, addr)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	client := pb.NewManagerClient(conn)

	r := &EndpointReport{
		Name: name,
		Addr: addr,
	}

	// GetAppInfo
	{
		reply, err := client.GetAppInfo(ctx, &pb.GetAppInfoRequest{})
		if err != nil {
			return nil, errors.Wrap(err, "GetAppInfo RPC")
		}
		app := reply.GetAppInfo()
		if app == nil {
			return nil, errors.New("GetAppInfo: reply.app_info is nil")
		}
		r.AppInfo = &AppInfoSummary{
			ApplicationVersion: app.GetApplicationVersion(),
			APIVersion:         fmt.Sprintf("%d.%d.%d", app.GetApiVersion().GetMajor(), app.GetApiVersion().GetMinor(), app.GetApiVersion().GetPatch()),
			LaunchPID:          app.GetLaunchPid(),
		}
	}

	// GetDevices (no simulation)
	{
		reply, err := client.GetDevices(ctx, &pb.GetDevicesRequest{IncludeSimulationDevices: false})
		if err != nil {
			return nil, errors.Wrap(err, "GetDevices(include_sim=false) RPC")
		}
		r.DevicesNo = summarizeDevices(reply.GetDevices())
	}

	// GetDevices (include simulation)
	{
		reply, err := client.GetDevices(ctx, &pb.GetDevicesRequest{IncludeSimulationDevices: true})
		if err != nil {
			return nil, errors.Wrap(err, "GetDevices(include_sim=true) RPC")
		}
		r.DevicesYes = summarizeDevices(reply.GetDevices())
	}

	// WaitCapture unknown capture id
	r.WaitCaptureUnknown = callExpectError(ctx, func() error {
		_, err := client.WaitCapture(ctx, &pb.WaitCaptureRequest{CaptureId: 999999999})
		return err
	})

	// StopCapture unknown capture id
	r.StopCaptureUnknown = callExpectError(ctx, func() error {
		_, err := client.StopCapture(ctx, &pb.StopCaptureRequest{CaptureId: 999999999})
		return err
	})

	// CloseCapture unknown capture id
	r.CloseCaptureUnknown = callExpectError(ctx, func() error {
		_, err := client.CloseCapture(ctx, &pb.CloseCaptureRequest{CaptureId: 999999999})
		return err
	})

	// LoadCapture empty filepath
	r.LoadCaptureEmptyPath = callExpectError(ctx, func() error {
		_, err := client.LoadCapture(ctx, &pb.LoadCaptureRequest{Filepath: ""})
		return err
	})

	// LoadCapture missing filepath
	r.LoadCaptureMissingPath = callExpectError(ctx, func() error {
		_, err := client.LoadCapture(ctx, &pb.LoadCaptureRequest{Filepath: missingFilepath})
		return err
	})

	return r, nil
}

func summarizeDevices(devs []*pb.Device) []DeviceSummary {
	out := make([]DeviceSummary, 0, len(devs))
	for _, d := range devs {
		if d == nil {
			continue
		}
		out = append(out, DeviceSummary{
			DeviceID:     d.GetDeviceId(),
			DeviceType:   deviceTypeString(d.GetDeviceType()),
			IsSimulation: d.GetIsSimulation(),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].DeviceID < out[j].DeviceID
	})
	return out
}

func deviceTypeString(dt pb.DeviceType) string {
	name, ok := pb.DeviceType_name[int32(dt)]
	if ok {
		return name
	}
	return fmt.Sprintf("DEVICE_TYPE_%d", int32(dt))
}

func callExpectError(ctx context.Context, fn func() error) *RPCErrorSummary {
	err := fn()
	if err == nil {
		return &RPCErrorSummary{Code: codes.OK.String(), Message: ""}
	}
	st, ok := status.FromError(err)
	if ok {
		return &RPCErrorSummary{Code: st.Code().String(), Message: st.Message()}
	}
	// Not a gRPC status error (still useful to record)
	return &RPCErrorSummary{Code: "NON_GRPC_ERROR", Message: err.Error()}
}

func dialAndWaitReady(ctx context.Context, addr string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, errors.Wrapf(err, "grpc.NewClient(%s)", addr)
	}

	// grpc.NewClient is lazy; force a connection attempt and wait until READY.
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

type diffOptions struct {
	ignoreLaunchPID  bool
	ignoreAppVersion bool
	diffDevices      bool
	diffMessages     bool
}

func diffReports(realReport, mockReport *EndpointReport, opts diffOptions) []Diff {
	var diffs []Diff

	// AppInfo
	if realReport.AppInfo != nil && mockReport.AppInfo != nil {
		if strings.TrimSpace(realReport.AppInfo.APIVersion) != strings.TrimSpace(mockReport.AppInfo.APIVersion) {
			diffs = append(diffs, Diff{Path: "appinfo.api_version", Real: realReport.AppInfo.APIVersion, Mock: mockReport.AppInfo.APIVersion})
		}
		if !opts.ignoreAppVersion && realReport.AppInfo.ApplicationVersion != mockReport.AppInfo.ApplicationVersion {
			diffs = append(diffs, Diff{Path: "appinfo.application_version", Real: realReport.AppInfo.ApplicationVersion, Mock: mockReport.AppInfo.ApplicationVersion})
		}
		if !opts.ignoreLaunchPID && realReport.AppInfo.LaunchPID != mockReport.AppInfo.LaunchPID {
			diffs = append(diffs, Diff{Path: "appinfo.launch_pid", Real: realReport.AppInfo.LaunchPID, Mock: mockReport.AppInfo.LaunchPID})
		}
	} else if realReport.AppInfo != nil || mockReport.AppInfo != nil {
		diffs = append(diffs, Diff{Path: "appinfo", Real: realReport.AppInfo, Mock: mockReport.AppInfo})
	}

	// Error-code comparisons (codes always compared; messages optionally)
	diffs = append(diffs, diffErr("wait_capture_unknown", realReport.WaitCaptureUnknown, mockReport.WaitCaptureUnknown, opts.diffMessages)...)
	diffs = append(diffs, diffErr("stop_capture_unknown", realReport.StopCaptureUnknown, mockReport.StopCaptureUnknown, opts.diffMessages)...)
	diffs = append(diffs, diffErr("close_capture_unknown", realReport.CloseCaptureUnknown, mockReport.CloseCaptureUnknown, opts.diffMessages)...)
	diffs = append(diffs, diffErr("load_capture_empty_path", realReport.LoadCaptureEmptyPath, mockReport.LoadCaptureEmptyPath, opts.diffMessages)...)
	diffs = append(diffs, diffErr("load_capture_missing_path", realReport.LoadCaptureMissingPath, mockReport.LoadCaptureMissingPath, opts.diffMessages)...)

	// Devices (often expected to differ)
	if opts.diffDevices {
		if !equalDevices(realReport.DevicesNo, mockReport.DevicesNo) {
			diffs = append(diffs, Diff{Path: "devices_no_sim", Real: realReport.DevicesNo, Mock: mockReport.DevicesNo})
		}
		if !equalDevices(realReport.DevicesYes, mockReport.DevicesYes) {
			diffs = append(diffs, Diff{Path: "devices_with_sim", Real: realReport.DevicesYes, Mock: mockReport.DevicesYes})
		}
	}

	sort.Slice(diffs, func(i, j int) bool { return diffs[i].Path < diffs[j].Path })
	return diffs
}

func diffErr(path string, realErr, mockErr *RPCErrorSummary, diffMessages bool) []Diff {
	var diffs []Diff
	if realErr == nil && mockErr == nil {
		return nil
	}
	if realErr == nil || mockErr == nil {
		diffs = append(diffs, Diff{Path: path, Real: realErr, Mock: mockErr})
		return diffs
	}
	if realErr.Code != mockErr.Code {
		diffs = append(diffs, Diff{Path: path + ".code", Real: realErr.Code, Mock: mockErr.Code})
	}
	if diffMessages && realErr.Message != mockErr.Message {
		diffs = append(diffs, Diff{Path: path + ".message", Real: realErr.Message, Mock: mockErr.Message})
	}
	return diffs
}

func equalDevices(a, b []DeviceSummary) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
