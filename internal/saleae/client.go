package saleae

import (
	"context"
	"time"

	pb "github.com/go-go-golems/salad/gen/saleae/automation"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn    *grpc.ClientConn
	manager pb.ManagerClient
}

func New(ctx context.Context, cfg Config) (*Client, error) {
	if _, ok := ctx.Deadline(); !ok && cfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.Timeout)
		defer cancel()
	}

	conn, err := grpc.DialContext(
		ctx,
		cfg.Addr(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "dial saleae automation grpc at %s", cfg.Addr())
	}

	return &Client{
		conn:    conn,
		manager: pb.NewManagerClient(conn),
	}, nil
}

func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *Client) GetAppInfo(ctx context.Context) (*pb.AppInfo, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	reply, err := c.manager.GetAppInfo(ctx, &pb.GetAppInfoRequest{})
	if err != nil {
		return nil, errors.Wrap(err, "GetAppInfo RPC")
	}

	if reply.GetAppInfo() == nil {
		return nil, errors.New("GetAppInfo: reply.app_info is nil")
	}

	return reply.GetAppInfo(), nil
}

func (c *Client) GetDevices(ctx context.Context, includeSimulationDevices bool) ([]*pb.Device, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	reply, err := c.manager.GetDevices(ctx, &pb.GetDevicesRequest{
		IncludeSimulationDevices: includeSimulationDevices,
	})
	if err != nil {
		return nil, errors.Wrap(err, "GetDevices RPC")
	}

	return reply.GetDevices(), nil
}

func (c *Client) LoadCapture(ctx context.Context, filepath string) (uint64, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if filepath == "" {
		return 0, errors.New("LoadCapture: filepath is required")
	}

	reply, err := c.manager.LoadCapture(ctx, &pb.LoadCaptureRequest{Filepath: filepath})
	if err != nil {
		return 0, errors.Wrap(err, "LoadCapture RPC")
	}

	if reply.GetCaptureInfo() == nil {
		return 0, errors.New("LoadCapture: reply.capture_info is nil")
	}

	return reply.GetCaptureInfo().GetCaptureId(), nil
}

func (c *Client) SaveCapture(ctx context.Context, captureID uint64, filepath string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if captureID == 0 {
		return errors.New("SaveCapture: capture-id must be non-zero")
	}
	if filepath == "" {
		return errors.New("SaveCapture: filepath is required")
	}

	_, err := c.manager.SaveCapture(ctx, &pb.SaveCaptureRequest{
		CaptureId: captureID,
		Filepath:  filepath,
	})
	return errors.Wrap(err, "SaveCapture RPC")
}

func (c *Client) CloseCapture(ctx context.Context, captureID uint64) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if captureID == 0 {
		return errors.New("CloseCapture: capture-id must be non-zero")
	}

	_, err := c.manager.CloseCapture(ctx, &pb.CloseCaptureRequest{CaptureId: captureID})
	return errors.Wrap(err, "CloseCapture RPC")
}

func (c *Client) StopCapture(ctx context.Context, captureID uint64) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if captureID == 0 {
		return errors.New("StopCapture: capture-id must be non-zero")
	}

	_, err := c.manager.StopCapture(ctx, &pb.StopCaptureRequest{CaptureId: captureID})
	return errors.Wrap(err, "StopCapture RPC")
}

func (c *Client) WaitCapture(ctx context.Context, captureID uint64) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if captureID == 0 {
		return errors.New("WaitCapture: capture-id must be non-zero")
	}

	_, err := c.manager.WaitCapture(ctx, &pb.WaitCaptureRequest{CaptureId: captureID})
	return errors.Wrap(err, "WaitCapture RPC")
}

func (c *Client) ExportRawDataCsv(
	ctx context.Context,
	captureID uint64,
	directory string,
	channels *pb.LogicChannels,
	analogDownsampleRatio uint64,
	iso8601Timestamp bool,
) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if captureID == 0 {
		return errors.New("ExportRawDataCsv: capture-id must be non-zero")
	}
	if directory == "" {
		return errors.New("ExportRawDataCsv: directory is required")
	}
	if channels == nil {
		return errors.New("ExportRawDataCsv: channels are required")
	}
	if analogDownsampleRatio == 0 {
		analogDownsampleRatio = 1
	}

	req := &pb.ExportRawDataCsvRequest{
		CaptureId:             captureID,
		Directory:             directory,
		Channels:              &pb.ExportRawDataCsvRequest_LogicChannels{LogicChannels: channels},
		AnalogDownsampleRatio: analogDownsampleRatio,
		Iso8601Timestamp:      iso8601Timestamp,
	}
	_, err := c.manager.ExportRawDataCsv(ctx, req)
	return errors.Wrap(err, "ExportRawDataCsv RPC")
}

func (c *Client) ExportRawDataBinary(
	ctx context.Context,
	captureID uint64,
	directory string,
	channels *pb.LogicChannels,
	analogDownsampleRatio uint64,
) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if captureID == 0 {
		return errors.New("ExportRawDataBinary: capture-id must be non-zero")
	}
	if directory == "" {
		return errors.New("ExportRawDataBinary: directory is required")
	}
	if channels == nil {
		return errors.New("ExportRawDataBinary: channels are required")
	}
	if analogDownsampleRatio == 0 {
		analogDownsampleRatio = 1
	}

	req := &pb.ExportRawDataBinaryRequest{
		CaptureId:             captureID,
		Directory:             directory,
		Channels:              &pb.ExportRawDataBinaryRequest_LogicChannels{LogicChannels: channels},
		AnalogDownsampleRatio: analogDownsampleRatio,
	}
	_, err := c.manager.ExportRawDataBinary(ctx, req)
	return errors.Wrap(err, "ExportRawDataBinary RPC")
}

// DialTimeout is retained for future use (eg separate dial/RPC timeouts).
// For now, Config.Timeout is used for both.
var DialTimeout = 0 * time.Second
