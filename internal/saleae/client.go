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

// DialTimeout is retained for future use (eg separate dial/RPC timeouts).
// For now, Config.Timeout is used for both.
var DialTimeout = 0 * time.Second
