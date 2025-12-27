package saleae

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	client "github.com/go-go-golems/salad/internal/saleae"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

func StartMockServer(plan *Plan, opts ...Option) (*Server, *grpc.Server, net.Listener, func(), error) {
	server := NewServer(plan, opts...)
	grpcServer := grpc.NewServer()
	server.Register(grpcServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, nil, nil, nil, errors.Wrap(err, "listen on random port")
	}

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	cleanup := func() {
		grpcServer.Stop()
		_ = listener.Close()
	}

	return server, grpcServer, listener, cleanup, nil
}

func StartMockServerFromYAML(t *testing.T, configPath string) (*Server, *client.Client, func()) {
	if t != nil {
		t.Helper()
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		if t != nil {
			t.Fatalf("load mock config: %v", err)
		}
		return nil, nil, func() {}
	}

	plan, err := Compile(cfg)
	if err != nil {
		if t != nil {
			t.Fatalf("compile mock config: %v", err)
		}
		return nil, nil, func() {}
	}

	server, grpcServer, listener, cleanup, err := StartMockServer(plan)
	if err != nil {
		if t != nil {
			t.Fatalf("start mock server: %v", err)
		}
		return nil, nil, func() {}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, portStr, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		cleanup()
		if t != nil {
			t.Fatalf("parse mock server port: %v", err)
		}
		return nil, nil, func() {}
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		cleanup()
		if t != nil {
			t.Fatalf("parse mock server port: %v", err)
		}
		return nil, nil, func() {}
	}

	clientConn, err := client.New(ctx, client.Config{Host: "127.0.0.1", Port: port, Timeout: 5 * time.Second})
	if err != nil {
		cleanup()
		if t != nil {
			t.Fatalf("connect to mock server: %v", err)
		}
		return nil, nil, func() {}
	}

	fullCleanup := func() {
		clientConn.Close()
		grpcServer.Stop()
		_ = listener.Close()
	}

	return server, clientConn, fullCleanup
}
