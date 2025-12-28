---
Title: 'Mock Saleae Server Design: Testing Without Real Hardware'
Ticket: 010-MOCK-SERVER
Status: active
Topics:
    - go
    - saleae
    - testing
    - mock
    - grpc
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: gen/saleae/automation/saleae_grpc.pb.go
      Note: Generated server interface ManagerServer that we need to implement
    - Path: internal/saleae/client.go
      Note: Client code that will connect to our mock server
    - Path: proto/saleae/grpc/saleae.proto
      Note: Proto definition showing all RPCs and message types
ExternalSources: []
Summary: "Design for a mock Saleae Logic 2 gRPC server that allows testing salad commands without requiring a real Logic 2 instance or hardware device."
LastUpdated: 2025-12-27T00:00:00Z
WhatFor: "Enable unit and integration tests for salad commands without requiring Logic 2 installation or physical hardware"
WhenToUse: "When writing tests for capture start, export, analyzer operations, or any other salad functionality"
---

# Mock Saleae Server Design: Testing Without Real Hardware

## Executive Summary

This guide explains how to build a mock Saleae Logic 2 gRPC server that implements the `Manager` service interface. The mock server allows us to test salad commands—especially complex ones like `capture start`—without requiring a real Logic 2 instance running or physical hardware connected.

**Who this is for:** Engineers implementing tests for salad commands. We assume familiarity with Go, gRPC, and the salad codebase, but not prior experience building gRPC servers or mock services.

**What you'll learn:** How to implement the `ManagerServer` interface from `gen/saleae/automation/saleae_grpc.pb.go`, structure state management for captures and devices, handle RPC requests correctly, and integrate the mock server into tests. By the end, you'll understand not just what to implement, but why each design decision matters for testability.

**The challenge:** The Saleae API has stateful operations (captures have IDs, devices can be queried, analyzers are added to captures). A good mock must track this state correctly and handle edge cases (invalid capture IDs, missing devices) to make tests meaningful. We'll show you how to structure the mock to be both simple and realistic.

## Mental Model: What a Mock Server Does

Before diving into implementation, let's understand what we're building. A mock server is a **stand-in** for the real Logic 2 automation server. When your test code calls `saleae.New()` with `host=127.0.0.1` and `port=10431`, it connects to our mock instead of Logic 2.

The mock server:
1. **Listens** on a TCP port (like `:10431`)
2. **Implements** all RPC methods from the `Manager` service
3. **Tracks state** (devices, captures, analyzers) in memory
4. **Returns responses** that match the proto contract

**Why this matters:** Tests can run anywhere (CI, developer machines) without Logic 2 installed. They're faster (no real hardware), deterministic (no flaky hardware), and can test error cases (invalid capture IDs, missing devices) that are hard to reproduce with real hardware.

**The trade-off:** The mock won't perfectly simulate Logic 2's behavior (e.g., actual signal capture, file I/O). But for testing **our code** (config parsing, proto conversion, error handling), it's sufficient. We're testing the salad client, not Logic 2 itself.

## Understanding the Codebase Structure

### Where the Client Connects

The client code in `internal/saleae/client.go` shows how connections work:

```go
func New(ctx context.Context, cfg Config) (*Client, error) {
    conn, err := grpc.DialContext(
        ctx,
        cfg.Addr(),  // e.g., "127.0.0.1:10430"
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
```

**Key insight:** The client uses `grpc.DialContext` to connect to any address. For tests, we'll start a mock server on `127.0.0.1:10431` (or a random port) and point the client there. The client doesn't know it's talking to a mock—it's just a gRPC connection.

**The address format:** `Config.Addr()` returns `host:port` (see `internal/saleae/config.go:15`). Our mock server will listen on `:port` (all interfaces) or `127.0.0.1:port` (localhost only).

### The Server Interface We Must Implement

The generated code in `gen/saleae/automation/saleae_grpc.pb.go` defines the `ManagerServer` interface (lines 260-294):

```go
type ManagerServer interface {
    GetAppInfo(context.Context, *GetAppInfoRequest) (*GetAppInfoReply, error)
    GetDevices(context.Context, *GetDevicesRequest) (*GetDevicesReply, error)
    StartCapture(context.Context, *StartCaptureRequest) (*StartCaptureReply, error)
    StopCapture(context.Context, *StopCaptureRequest) (*StopCaptureReply, error)
    WaitCapture(context.Context, *WaitCaptureRequest) (*WaitCaptureReply, error)
    LoadCapture(context.Context, *LoadCaptureRequest) (*LoadCaptureReply, error)
    SaveCapture(context.Context, *SaveCaptureRequest) (*SaveCaptureReply, error)
    CloseCapture(context.Context, *CloseCaptureRequest) (*CloseCaptureReply, error)
    AddAnalyzer(context.Context, *AddAnalyzerRequest) (*AddAnalyzerReply, error)
    RemoveAnalyzer(context.Context, *RemoveAnalyzerRequest) (*RemoveAnalyzerReply, error)
    AddHighLevelAnalyzer(context.Context, *AddHighLevelAnalyzerRequest) (*AddHighLevelAnalyzerReply, error)
    RemoveHighLevelAnalyzer(context.Context, *RemoveHighLevelAnalyzerRequest) (*RemoveHighLevelAnalyzerReply, error)
    ExportRawDataCsv(context.Context, *ExportRawDataCsvRequest) (*ExportRawDataCsvReply, error)
    ExportRawDataBinary(context.Context, *ExportRawDataBinaryRequest) (*ExportRawDataBinaryReply, error)
    ExportDataTableCsv(context.Context, *ExportDataTableCsvRequest) (*ExportDataTableCsvReply, error)
    LegacyExportAnalyzer(context.Context, *LegacyExportAnalyzerRequest) (*LegacyExportAnalyzerReply, error)
    
    mustEmbedUnimplementedManagerServer()
}
```

**Why embed UnimplementedManagerServer:** The generated code provides `UnimplementedManagerServer` (lines 302-352) that returns "not implemented" errors for all methods. We embed this by value (not pointer) so we only implement the methods we need. This is forward-compatible—if Saleae adds new RPCs, our mock won't break.

**The registration function:** `RegisterManagerServer` (line 362) registers our implementation with a gRPC server. We'll call this after creating the server.

### All RPCs We Need to Implement

From `proto/saleae/grpc/saleae.proto`, the `Manager` service has 14 RPCs:

**Discovery:**
- `GetAppInfo` - Returns app version, API version, PID
- `GetDevices` - Returns list of connected devices

**Capture lifecycle:**
- `StartCapture` - Creates a new capture, returns capture_id
- `StopCapture` - Stops an active capture
- `WaitCapture` - Waits for capture completion (for timed/trigger modes)
- `LoadCapture` - Loads a capture from file, returns capture_id
- `SaveCapture` - Saves a capture to file
- `CloseCapture` - Releases capture resources

**Analyzers:**
- `AddAnalyzer` - Adds a protocol analyzer to a capture, returns analyzer_id
- `RemoveAnalyzer` - Removes an analyzer
- `AddHighLevelAnalyzer` - Adds an HLA to a capture, returns analyzer_id
- `RemoveHighLevelAnalyzer` - Removes an HLA

**Exports:**
- `ExportRawDataCsv` - Exports raw channel data to CSV files
- `ExportRawDataBinary` - Exports raw channel data to binary files
- `ExportDataTableCsv` - Exports decoded analyzer data to CSV
- `LegacyExportAnalyzer` - Legacy export format

**Priority for initial implementation:** Start with the RPCs used by existing commands:
1. `GetAppInfo` (used by `salad appinfo`)
2. `GetDevices` (used by `salad devices`)
3. `LoadCapture`, `SaveCapture`, `StopCapture`, `WaitCapture`, `CloseCapture` (used by `salad capture`)
4. `ExportRawDataCsv`, `ExportRawDataBinary` (used by `salad export`)
5. `StartCapture` (needed for ticket 002)

The analyzer and HLA RPCs can be stubbed initially (return success but don't track state).

## Architecture: How to Structure the Mock Server

### Package Location

Create `internal/mock/saleae/` for the mock server implementation. This keeps it separate from the real client code and makes it clear it's test infrastructure.

**Structure:**
```
internal/mock/saleae/
  server.go          # Main server struct and gRPC server setup
  state.go           # State management (captures, devices, analyzers)
  handlers.go        # RPC method implementations (or split by category)
  server_test.go     # Tests for the mock server itself
```

**Why internal:** The mock is test infrastructure, not part of the public API. Tests in other packages can import it, but external users shouldn't.

### State Management: What to Track

The mock server needs to track state that persists across RPC calls:

**Devices:**
- List of devices (device_id, device_type, is_simulation)
- Default device for `StartCapture` when device_id is empty

**Captures:**
- Map of `capture_id → CaptureState`
- CaptureState includes:
  - Device ID used
  - Capture configuration (mode, channels, sample rates)
  - Status (running, stopped, completed)
  - For timed captures: start time, duration
  - For trigger captures: trigger condition, triggered flag

**Analyzers:**
- Map of `(capture_id, analyzer_id) → AnalyzerState`
- AnalyzerState includes:
  - Analyzer name (SPI, I2C, etc.)
  - Settings map
  - For HLAs: extension directory, HLA name, input analyzer ID

**Why this structure:** Real Logic 2 tracks this state internally. Our mock must do the same to handle sequences like "start capture → add analyzer → export table" correctly.

### Server Struct Design

```go
package saleae

import (
    "context"
    "sync"
    
    pb "github.com/go-go-golems/salad/gen/saleae/automation"
    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

type Server struct {
    pb.UnimplementedManagerServer  // Embed for forward compatibility
    
    mu sync.RWMutex  // Protects all state
    
    // State
    devices   []*pb.Device
    captures  map[uint64]*CaptureState
    analyzers map[string]*AnalyzerState  // Key: "capture_id:analyzer_id"
    
    // Counters for ID generation
    nextCaptureID  uint64
    nextAnalyzerID uint64
    
    // Configuration
    defaultAppInfo *pb.AppInfo
}

type CaptureState struct {
    ID              uint64
    DeviceID        string
    Config          *pb.CaptureConfiguration
    DeviceConfig    *pb.LogicDeviceConfiguration
    Status          CaptureStatus  // running, stopped, completed
    StartTime       time.Time
    // ... mode-specific fields
}

type CaptureStatus int

const (
    CaptureStatusRunning CaptureStatus = iota
    CaptureStatusStopped
    CaptureStatusCompleted
)
```

**Why mutex:** Multiple goroutines might call RPCs concurrently (though tests usually don't). The mutex ensures thread-safe access to state.

**Why separate state structs:** Keeps the server struct clean and makes it easy to add fields later. We can also serialize state for debugging.

### Starting the Server

```go
func NewServer() *Server {
    return &Server{
        devices:   []*pb.Device{},
        captures:  make(map[uint64]*CaptureState),
        analyzers: make(map[string]*AnalyzerState),
        nextCaptureID:  1,
        nextAnalyzerID: 1,
        defaultAppInfo: &pb.AppInfo{
            ApiVersion: &pb.Version{Major: 1, Minor: 0, Patch: 0},
            ApplicationVersion: "2.3.56",
            LaunchPid: uint64(os.Getpid()),
        },
    }
}

func (s *Server) Start(addr string) (*grpc.Server, net.Listener, error) {
    lis, err := net.Listen("tcp", addr)
    if err != nil {
        return nil, nil, err
    }
    
    grpcServer := grpc.NewServer()
    pb.RegisterManagerServer(grpcServer, s)
    
    go func() {
        if err := grpcServer.Serve(lis); err != nil {
            log.Printf("Mock server error: %v", err)
        }
    }()
    
    return grpcServer, lis, nil
}
```

**Why return server and listener:** The test code needs to call `grpcServer.Stop()` and `lis.Close()` in cleanup. Returning both gives the caller control.

**Why goroutine:** `grpcServer.Serve()` blocks. We run it in a goroutine so the test can continue. The server runs until `Stop()` is called.

### Test Helper Pattern

Create a helper function that starts the server and returns a client connected to it:

```go
// internal/mock/saleae/test_helper.go

func StartMockServer(t *testing.T) (*Server, *saleae.Client, func()) {
    s := NewServer()
    
    // Use random port to avoid conflicts
    lis, err := net.Listen("tcp", ":0")
    require.NoError(t, err)
    
    addr := lis.Addr().String()
    grpcServer := grpc.NewServer()
    pb.RegisterManagerServer(grpcServer, s)
    
    go grpcServer.Serve(lis)
    
    // Connect client
    ctx := context.Background()
    client, err := saleae.New(ctx, saleae.Config{
        Host:    "127.0.0.1",
        Port:    extractPort(addr),
        Timeout: 5 * time.Second,
    })
    require.NoError(t, err)
    
    cleanup := func() {
        grpcServer.Stop()
        lis.Close()
        client.Close()
    }
    
    return s, client, cleanup
}
```

**Why this pattern:** Tests become simple: `server, client, cleanup := StartMockServer(t); defer cleanup()`. The helper handles all the boilerplate.

## RPC Implementation Patterns

### Simple RPCs: GetAppInfo

```go
func (s *Server) GetAppInfo(ctx context.Context, req *pb.GetAppInfoRequest) (*pb.GetAppInfoReply, error) {
    return &pb.GetAppInfoReply{
        AppInfo: s.defaultAppInfo,
    }, nil
}
```

**Why simple:** `GetAppInfo` has no side effects. We just return the configured app info.

### Stateful RPCs: StartCapture

```go
func (s *Server) StartCapture(ctx context.Context, req *pb.StartCaptureRequest) (*pb.StartCaptureReply, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    // Validate device
    deviceID := req.DeviceId
    if deviceID == "" {
        // Find first physical device
        for _, d := range s.devices {
            if !d.IsSimulation {
                deviceID = d.DeviceId
                break
            }
        }
        if deviceID == "" {
            return nil, status.Error(codes.NotFound, "no physical device available")
        }
    }
    
    // Validate device exists
    deviceExists := false
    for _, d := range s.devices {
        if d.DeviceId == deviceID {
            deviceExists = true
            break
        }
    }
    if !deviceExists {
        return nil, status.Error(codes.NotFound, "device not found")
    }
    
    // Create capture state
    captureID := s.nextCaptureID
    s.nextCaptureID++
    
    capture := &CaptureState{
        ID:           captureID,
        DeviceID:     deviceID,
        Config:       req.CaptureConfiguration,
        DeviceConfig: req.GetLogicDeviceConfiguration(),
        Status:       CaptureStatusRunning,
        StartTime:    time.Now(),
    }
    
    s.captures[captureID] = capture
    
    return &pb.StartCaptureReply{
        CaptureInfo: &pb.CaptureInfo{
            CaptureId: captureID,
        },
    }, nil
}
```

**Why this validation:** We mirror Logic 2's behavior: empty device_id uses first physical device, invalid device_id returns error. This makes tests realistic.

**Why track state:** Later RPCs (`StopCapture`, `WaitCapture`) need to find the capture by ID and check its status.

### Error Handling: Invalid Capture ID

```go
func (s *Server) StopCapture(ctx context.Context, req *pb.StopCaptureRequest) (*pb.StopCaptureReply, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    capture, exists := s.captures[req.CaptureId]
    if !exists {
        return nil, status.Error(codes.InvalidArgument, "capture not found")
    }
    
    capture.Status = CaptureStatusStopped
    
    return &pb.StopCaptureReply{}, nil
}
```

**Why status codes:** Use gRPC status codes (`codes.InvalidArgument`, `codes.NotFound`) to match real Logic 2 behavior. Tests can check error codes, not just error strings.

### Complex RPCs: WaitCapture (Timed Mode)

```go
func (s *Server) WaitCapture(ctx context.Context, req *pb.WaitCaptureRequest) (*pb.WaitCaptureReply, error) {
    s.mu.Lock()
    capture, exists := s.captures[req.CaptureId]
    if !exists {
        s.mu.Unlock()
        return nil, status.Error(codes.InvalidArgument, "capture not found")
    }
    
    // For timed captures, check if duration has elapsed
    if timedMode := capture.Config.GetTimedCaptureMode(); timedMode != nil {
        elapsed := time.Since(capture.StartTime)
        duration := time.Duration(timedMode.DurationSeconds * float64(time.Second))
        
        if elapsed < duration {
            // Still running
            s.mu.Unlock()
            // In real Logic 2, this would block. For mock, we can either:
            // 1. Return immediately (test assumes capture completes instantly)
            // 2. Sleep until duration (makes tests slower)
            // 3. Use a channel/context to simulate waiting (complex)
            // For now, return success if elapsed >= duration, error otherwise
            return nil, status.Error(codes.DeadlineExceeded, "capture still running")
        }
        
        capture.Status = CaptureStatusCompleted
    }
    
    s.mu.Unlock()
    return &pb.WaitCaptureReply{}, nil
}
```

**Why this complexity:** `WaitCapture` behavior depends on capture mode. For timed captures, we check elapsed time. For manual captures, it should error (per proto comment). This makes the mock realistic.

**The gotcha:** Real `WaitCapture` blocks until completion. Our mock can't block tests indefinitely. Options: return immediately (assume completion), or use a timeout. We'll document this limitation.

### Export RPCs: File I/O Simulation

```go
func (s *Server) ExportRawDataCsv(ctx context.Context, req *pb.ExportRawDataCsvRequest) (*pb.ExportRawDataCsvReply, error) {
    s.mu.RLock()
    capture, exists := s.captures[req.CaptureId]
    s.mu.RUnlock()
    
    if !exists {
        return nil, status.Error(codes.InvalidArgument, "capture not found")
    }
    
    // In real Logic 2, this writes CSV files. For mock:
    // Option 1: Actually write files (test can verify)
    // Option 2: Track export calls (test verifies RPC was called)
    // Option 3: Return success but do nothing (simplest)
    
    // For now, Option 3: just verify capture exists
    return &pb.ExportRawDataCsvReply{}, nil
}
```

**Why options:** Export RPCs write files. For tests, we might want to verify files were created (Option 1), or just verify the RPC was called correctly (Option 2). Start simple (Option 3), add file I/O if tests need it.

## Testing Strategy

### Unit Tests for the Mock Server

Test the mock server itself to ensure it behaves correctly:

```go
func TestMockServer_StartCapture(t *testing.T) {
    server := NewServer()
    server.devices = []*pb.Device{
        {DeviceId: "DEV1", DeviceType: pb.DeviceType_DEVICE_TYPE_LOGIC_PRO_8, IsSimulation: false},
    }
    
    req := &pb.StartCaptureRequest{
        DeviceId: "DEV1",
        DeviceConfiguration: &pb.StartCaptureRequest_LogicDeviceConfiguration{
            LogicDeviceConfiguration: &pb.LogicDeviceConfiguration{
                EnabledChannels: &pb.LogicDeviceConfiguration_LogicChannels{
                    LogicChannels: &pb.LogicChannels{
                        DigitalChannels: []uint32{0, 1},
                    },
                },
                DigitalSampleRate: 10000000,
            },
        },
        CaptureConfiguration: &pb.CaptureConfiguration{
            BufferSizeMegabytes: 128,
            CaptureMode: &pb.CaptureConfiguration_TimedCaptureMode{
                TimedCaptureMode: &pb.TimedCaptureMode{
                    DurationSeconds: 2.0,
                },
            },
        },
    }
    
    reply, err := server.StartCapture(context.Background(), req)
    require.NoError(t, err)
    require.NotNil(t, reply.CaptureInfo)
    require.Equal(t, uint64(1), reply.CaptureInfo.CaptureId)
    
    // Verify state
    require.Len(t, server.captures, 1)
    capture := server.captures[1]
    require.Equal(t, "DEV1", capture.DeviceID)
    require.Equal(t, CaptureStatusRunning, capture.Status)
}
```

**Why test the mock:** If the mock is wrong, all tests using it are wrong. Unit tests catch bugs in the mock itself.

### Integration Tests Using the Mock

Test actual client code against the mock:

```go
func TestClient_StartCapture(t *testing.T) {
    server, client, cleanup := StartMockServer(t)
    defer cleanup()
    
    // Setup: add a device
    server.devices = []*pb.Device{
        {DeviceId: "DEV1", DeviceType: pb.DeviceType_DEVICE_TYPE_LOGIC_PRO_8},
    }
    
    // Test: start capture via client
    req := &pb.StartCaptureRequest{
        // ... build request
    }
    
    captureID, err := client.StartCapture(context.Background(), req)
    require.NoError(t, err)
    require.Equal(t, uint64(1), captureID)
    
    // Verify via server state
    require.Len(t, server.captures, 1)
}
```

**Why integration tests:** These test the **client code**, not the mock. The mock is just infrastructure. These tests verify client error handling, proto conversion, etc.

## Implementation Priorities

### Phase 1: Basic Infrastructure

1. Create `internal/mock/saleae/` package
2. Implement `Server` struct with state management
3. Implement `StartMockServer` test helper
4. Implement `GetAppInfo` and `GetDevices` (simplest RPCs)

**Success criteria:** Tests can start mock server, connect client, call `GetAppInfo` and `GetDevices`.

### Phase 2: Capture Lifecycle

1. Implement `StartCapture` with device validation
2. Implement `StopCapture`, `WaitCapture`, `CloseCapture`
3. Implement `LoadCapture` and `SaveCapture` (can be stubs initially)

**Success criteria:** Tests for `capture start` command work end-to-end.

### Phase 3: Export RPCs

1. Implement `ExportRawDataCsv` and `ExportRawDataBinary`
2. Optionally: actually write files for verification

**Success criteria:** Tests for `export` commands work.

### Phase 4: Analyzers (Optional)

1. Implement `AddAnalyzer` and `RemoveAnalyzer`
2. Track analyzer state per capture
3. Implement `AddHighLevelAnalyzer` and `RemoveHighLevelAnalyzer`

**Success criteria:** Tests for analyzer commands work.

## Common Pitfalls

**Forgetting to lock:** All state access must be protected by `s.mu`. Forgetting locks causes race conditions in concurrent tests.

**Wrong status codes:** Use gRPC status codes (`codes.NotFound`, `codes.InvalidArgument`) not Go errors. Tests check status codes.

**State not persisting:** Each RPC call gets a new context. State must be in the `Server` struct, not function locals.

**Capture ID generation:** Use atomic increment or mutex-protected counter. Don't use `len(captures)`—deleted captures break this.

**WaitCapture blocking:** Real `WaitCapture` blocks. Mock shouldn't block tests. Return immediately or use short timeout.

## References

- **Server interface:** `gen/saleae/automation/saleae_grpc.pb.go` (lines 260-294)
- **Unimplemented server:** `gen/saleae/automation/saleae_grpc.pb.go` (lines 302-352)
- **Registration function:** `gen/saleae/automation/saleae_grpc.pb.go` (line 362)
- **Client code:** `internal/saleae/client.go`
- **Proto definition:** `proto/saleae/grpc/saleae.proto`
- **gRPC Go docs:** https://pkg.go.dev/google.golang.org/grpc

