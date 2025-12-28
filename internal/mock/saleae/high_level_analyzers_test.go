package saleae

import (
	"context"
	"testing"
	"time"

	pb "github.com/go-go-golems/salad/gen/saleae/automation"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func TestMockServer_AddRemoveHighLevelAnalyzer(t *testing.T) {
	cfg := Config{
		Version:  1,
		Scenario: "hla-happy-path",
		Defaults: DefaultsConfig{
			IDs: IDsDefaultsConfig{
				Deterministic: ptrBool(true),
				CaptureIDStart: 1,
				AnalyzerIDStart: 10000,
			},
		},
		Fixtures: FixturesConfig{
			AppInfo: &AppInfoConfig{ApplicationVersion: "mock"},
			Devices: []DeviceConfig{
				{DeviceID: "DEV1", DeviceType: "DEVICE_TYPE_LOGIC_PRO_8", IsSimulation: false},
			},
		},
	}

	plan, err := Compile(cfg)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	server, grpcServer, listener, cleanup, err := StartMockServer(plan)
	_ = server // reserved for future assertions
	if err != nil {
		t.Fatalf("StartMockServer: %v", err)
	}
	defer cleanup()
	defer grpcServer.Stop()

	conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("grpc.NewClient: %v", err)
	}
	defer func() { _ = conn.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	manager := pb.NewManagerClient(conn)

	// Create a capture (LoadCapture is simplest in the mock).
	loadReply, err := manager.LoadCapture(ctx, &pb.LoadCaptureRequest{Filepath: "/tmp/mock.sal"})
	if err != nil {
		t.Fatalf("LoadCapture: %v", err)
	}
	captureID := loadReply.GetCaptureInfo().GetCaptureId()
	if captureID == 0 {
		t.Fatalf("expected non-zero capture_id")
	}

	// Add a base analyzer, which the HLA will use as input.
	addReply, err := manager.AddAnalyzer(ctx, &pb.AddAnalyzerRequest{
		CaptureId:    captureID,
		AnalyzerName: "SPI",
		AnalyzerLabel: "base",
		Settings: map[string]*pb.AnalyzerSettingValue{
			"Clock": {Value: &pb.AnalyzerSettingValue_Int64Value{Int64Value: 0}},
		},
	})
	if err != nil {
		t.Fatalf("AddAnalyzer: %v", err)
	}
	inputAnalyzerID := addReply.GetAnalyzerId()
	if inputAnalyzerID == 0 {
		t.Fatalf("expected non-zero analyzer_id")
	}

	// Add HLA.
	hlaReply, err := manager.AddHighLevelAnalyzer(ctx, &pb.AddHighLevelAnalyzerRequest{
		CaptureId:           captureID,
		ExtensionDirectory:  "/tmp/ext",
		HlaName:             "my_hla",
		HlaLabel:            "hla",
		InputAnalyzerId:     inputAnalyzerID,
		Settings:            map[string]*pb.HighLevelAnalyzerSettingValue{"foo": {Value: &pb.HighLevelAnalyzerSettingValue_StringValue{StringValue: "bar"}}},
	})
	if err != nil {
		t.Fatalf("AddHighLevelAnalyzer: %v", err)
	}
	hlaID := hlaReply.GetAnalyzerId()
	if hlaID == 0 {
		t.Fatalf("expected non-zero hla analyzer_id")
	}

	// Remove HLA.
	_, err = manager.RemoveHighLevelAnalyzer(ctx, &pb.RemoveHighLevelAnalyzerRequest{
		CaptureId:  captureID,
		AnalyzerId: hlaID,
	})
	if err != nil {
		t.Fatalf("RemoveHighLevelAnalyzer: %v", err)
	}

	// Removing again should error by default.
	_, err = manager.RemoveHighLevelAnalyzer(ctx, &pb.RemoveHighLevelAnalyzerRequest{
		CaptureId:  captureID,
		AnalyzerId: hlaID,
	})
	if err == nil {
		t.Fatalf("expected second RemoveHighLevelAnalyzer to fail")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error, got %T", err)
	}
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %s (%s)", st.Code(), st.Message())
	}
}

func TestMockServer_AddHighLevelAnalyzer_RequiresInputAnalyzerExists(t *testing.T) {
	cfg := Config{
		Version:  1,
		Scenario: "hla-requires-input",
		Fixtures: FixturesConfig{
			AppInfo: &AppInfoConfig{ApplicationVersion: "mock"},
			Devices: []DeviceConfig{
				{DeviceID: "DEV1", DeviceType: "DEVICE_TYPE_LOGIC_PRO_8", IsSimulation: false},
			},
		},
	}
	plan, err := Compile(cfg)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	_, grpcServer, listener, cleanup, err := StartMockServer(plan)
	if err != nil {
		t.Fatalf("StartMockServer: %v", err)
	}
	defer cleanup()
	defer grpcServer.Stop()

	conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("grpc.NewClient: %v", err)
	}
	defer func() { _ = conn.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	manager := pb.NewManagerClient(conn)

	loadReply, err := manager.LoadCapture(ctx, &pb.LoadCaptureRequest{Filepath: "/tmp/mock.sal"})
	if err != nil {
		t.Fatalf("LoadCapture: %v", err)
	}
	captureID := loadReply.GetCaptureInfo().GetCaptureId()

	_, err = manager.AddHighLevelAnalyzer(ctx, &pb.AddHighLevelAnalyzerRequest{
		CaptureId:          captureID,
		ExtensionDirectory: "/tmp/ext",
		HlaName:            "my_hla",
		HlaLabel:           "hla",
		InputAnalyzerId:    999999,
	})
	if err == nil {
		t.Fatalf("expected AddHighLevelAnalyzer to fail when input analyzer does not exist")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error, got %T", err)
	}
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %s (%s)", st.Code(), st.Message())
	}
}


