---
Title: CLI Verbs to gRPC Methods Mapping
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
    - Path: cmd/salad/cmd/appinfo.go
      Note: CLI command that calls GetAppInfo
    - Path: cmd/salad/cmd/devices.go
      Note: CLI command that calls GetDevices
    - Path: cmd/salad/cmd/capture.go
      Note: CLI commands for capture lifecycle management
    - Path: cmd/salad/cmd/export.go
      Note: CLI commands for data export
    - Path: internal/saleae/client.go
      Note: Client wrapper methods that call gRPC methods
ExternalSources: []
Summary: "Analysis of current salad CLI commands and the gRPC methods they use, mapping each command to its underlying RPC calls and identifying requirements for mock server implementation."
LastUpdated: 2025-12-27T12:48:31.21303191-05:00
WhatFor: "Understand which gRPC methods need to be implemented in the mock server to support testing existing CLI commands"
WhenToUse: "When implementing mock server RPC handlers or adding new CLI command tests"
---

# CLI Verbs to gRPC Methods Mapping

## Executive Summary

This document maps each implemented salad CLI command to the gRPC methods it calls, analyzes how each command uses those methods, and identifies what the mock server must implement to support testing. This analysis ensures the mock server covers all current CLI functionality.

**Current Status:** Salad has 8 implemented CLI commands across 3 categories (discovery, capture management, export). All commands use the `Manager` gRPC service.

## Command Categories

### 1. Discovery Commands

#### `salad appinfo`

**CLI File:** `cmd/salad/cmd/appinfo.go`  
**Client Method:** `c.GetAppInfo(ctx)` (line 33)  
**gRPC Method:** `Manager.GetAppInfo`

**Request/Response:**
- Request: `GetAppInfoRequest{}` (empty)
- Response: `GetAppInfoReply{AppInfo: *AppInfo}`

**What the Command Does:**
1. Creates client connection
2. Calls `GetAppInfo` RPC
3. Extracts `api_version`, `application_version`, `launch_pid` from response
4. Prints formatted output: `application_version=X\napi_version=Y.Z.W\nlaunch_pid=P\n`

**Mock Server Requirements:**
- ✅ Must return `AppInfo` with:
  - `api_version`: `Version{Major: 1, Minor: 0, Patch: 0}` (or configurable)
  - `application_version`: string (e.g., "2.3.56" or configurable)
  - `launch_pid`: uint64 (can use `os.Getpid()` or configurable)

**Complexity:** ⭐ Simple — No state, no side effects, just return configured values.

---

#### `salad devices`

**CLI File:** `cmd/salad/cmd/devices.go`  
**Client Method:** `c.GetDevices(ctx, includeSimulationDevices)` (line 35)  
**gRPC Method:** `Manager.GetDevices`

**Request/Response:**
- Request: `GetDevicesRequest{IncludeSimulationDevices: bool}`
- Response: `GetDevicesReply{Devices: []*Device}`

**What the Command Does:**
1. Creates client connection
2. Calls `GetDevices` RPC with `--include-simulation-devices` flag value
3. Iterates over returned devices
4. Prints each device: `device_id=X device_type=Y is_simulation=Z\n`

**Mock Server Requirements:**
- ✅ Must maintain list of devices (can be empty initially)
- ✅ Must filter simulation devices based on `include_simulation_devices` flag
- ✅ Each device must have:
  - `device_id`: string (unique identifier)
  - `device_type`: `DeviceType` enum (e.g., `DEVICE_TYPE_LOGIC_PRO_8`)
  - `is_simulation`: bool

**Complexity:** ⭐ Simple — Just return configured device list, filter by flag.

**State Management:**
- Mock server should allow tests to configure devices via `server.AddDevice()` or similar
- Default: empty list (tests can add devices as needed)

---

### 2. Capture Management Commands

#### `salad capture load`

**CLI File:** `cmd/salad/cmd/capture.go` (lines 22-47)  
**Client Method:** `c.LoadCapture(ctx, filepath)` (line 39)  
**gRPC Method:** `Manager.LoadCapture`

**Request/Response:**
- Request: `LoadCaptureRequest{Filepath: string}` (required flag)
- Response: `LoadCaptureReply{CaptureInfo: *CaptureInfo}`

**What the Command Does:**
1. Creates client connection
2. Calls `LoadCapture` with absolute filepath
3. Extracts `capture_id` from response
4. Prints: `capture_id=X\n`

**Mock Server Requirements:**
- ✅ Must accept filepath (can be validated or ignored for mock)
- ✅ Must return `CaptureInfo{CaptureId: uint64}` with new capture ID
- ✅ Must track loaded capture in state (for later operations)

**Complexity:** ⭐⭐ Medium — Requires state tracking (capture ID generation, capture storage).

**State Management:**
- Generate new capture ID (increment counter)
- Store capture in `captures` map with status "loaded"
- Filepath validation can be minimal (just check non-empty) or strict (file exists)

---

#### `salad capture save`

**CLI File:** `cmd/salad/cmd/capture.go` (lines 49-73)  
**Client Method:** `c.SaveCapture(ctx, captureID, filepath)` (line 66)  
**gRPC Method:** `Manager.SaveCapture`

**Request/Response:**
- Request: `SaveCaptureRequest{CaptureId: uint64, Filepath: string}` (both required)
- Response: `SaveCaptureReply{}` (empty)

**What the Command Does:**
1. Creates client connection
2. Calls `SaveCapture` with capture ID and filepath
3. Prints: `ok\n` on success

**Mock Server Requirements:**
- ✅ Must validate capture ID exists
- ✅ Must accept filepath (can write to disk or just track call)
- ✅ Return error if capture not found (`codes.InvalidArgument`)

**Complexity:** ⭐⭐ Medium — Requires state lookup, optional file I/O.

**State Management:**
- Lookup capture by ID in `captures` map
- If not found, return error
- Optionally: actually write file (for integration tests) or just track that save was called

---

#### `salad capture stop`

**CLI File:** `cmd/salad/cmd/capture.go` (lines 75-99)  
**Client Method:** `c.StopCapture(ctx, captureID)` (line 92)  
**gRPC Method:** `Manager.StopCapture`

**Request/Response:**
- Request: `StopCaptureRequest{CaptureId: uint64}` (required)
- Response: `StopCaptureReply{}` (empty)

**What the Command Does:**
1. Creates client connection
2. Calls `StopCapture` with capture ID
3. Prints: `ok\n` on success

**Mock Server Requirements:**
- ✅ Must validate capture ID exists
- ✅ Must update capture status to "stopped"
- ✅ Return error if capture not found

**Complexity:** ⭐⭐ Medium — Requires state lookup and update.

**State Management:**
- Lookup capture by ID
- Update `capture.Status` to `CaptureStatusStopped`
- If capture not found, return `codes.InvalidArgument`

---

#### `salad capture wait`

**CLI File:** `cmd/salad/cmd/capture.go` (lines 101-125)  
**Client Method:** `c.WaitCapture(ctx, captureID)` (line 118)  
**gRPC Method:** `Manager.WaitCapture`

**Request/Response:**
- Request: `WaitCaptureRequest{CaptureId: uint64}` (required)
- Response: `WaitCaptureReply{}` (empty)

**What the Command Does:**
1. Creates client connection
2. Calls `WaitCapture` with capture ID
3. Prints: `ok\n` on success (after capture completes)

**Mock Server Requirements:**
- ✅ Must validate capture ID exists
- ✅ Must check capture mode (manual vs timed vs trigger)
- ✅ For timed captures: check if duration elapsed, update status to "completed"
- ✅ For manual captures: return error (per proto comment: "not for manual capture mode")
- ✅ Return error if capture not found

**Complexity:** ⭐⭐⭐ Complex — Requires mode-specific logic, timing simulation.

**State Management:**
- Lookup capture by ID
- Check `capture.Config.CaptureMode`:
  - `ManualCaptureMode`: return error (shouldn't use WaitCapture)
  - `TimedCaptureMode`: check elapsed time vs duration, update status if complete
  - `DigitalTriggerCaptureMode`: check if triggered, update status if complete
- **Design Decision:** Real `WaitCapture` blocks. Mock can:
  - Option A: Return immediately if complete, error if still running (simplest)
  - Option B: Sleep until duration (slow tests)
  - Option C: Use channels/context for realistic blocking (complex)

**Recommendation:** Start with Option A, document limitation.

---

#### `salad capture close`

**CLI File:** `cmd/salad/cmd/capture.go` (lines 127-151)  
**Client Method:** `c.CloseCapture(ctx, captureID)` (line 144)  
**gRPC Method:** `Manager.CloseCapture`

**Request/Response:**
- Request: `CloseCaptureRequest{CaptureId: uint64}` (required)
- Response: `CloseCaptureReply{}` (empty)

**What the Command Does:**
1. Creates client connection
2. Calls `CloseCapture` with capture ID
3. Prints: `ok\n` on success

**Mock Server Requirements:**
- ✅ Must validate capture ID exists
- ✅ Must remove capture from state (or mark as closed)
- ✅ Return error if capture not found

**Complexity:** ⭐⭐ Medium — Requires state lookup and cleanup.

**State Management:**
- Lookup capture by ID
- Remove from `captures` map (or mark as closed if we want to track closed captures)
- If not found, return `codes.InvalidArgument`

**Design Decision:** Remove from map (simpler) vs keep with "closed" status (allows testing double-close). Start with remove, add status if needed.

---

### 3. Export Commands

#### `salad export raw-csv`

**CLI File:** `cmd/salad/cmd/export.go` (lines 45-74)  
**Client Method:** `c.ExportRawDataCsv(ctx, captureID, directory, channels, analogDownsample, iso8601Timestamp)` (line 67)  
**gRPC Method:** `Manager.ExportRawDataCsv`

**Request/Response:**
- Request: `ExportRawDataCsvRequest{
    CaptureId: uint64,
    Directory: string,
    Channels: *LogicChannels,
    AnalogDownsampleRatio: uint64,
    Iso8601Timestamp: bool
  }`
- Response: `ExportRawDataCsvReply{}` (empty)

**What the Command Does:**
1. Parses `--digital` and `--analog` flags into `LogicChannels`
2. Creates client connection
3. Calls `ExportRawDataCsv` with all parameters
4. Prints: `ok\n` on success

**Mock Server Requirements:**
- ✅ Must validate capture ID exists
- ✅ Must accept directory path
- ✅ Must accept channel selection (`LogicChannels`)
- ✅ Must accept analog downsample ratio (1-1,000,000)
- ✅ Must accept ISO8601 timestamp flag
- ✅ Optionally: actually write CSV files to directory (for test verification)

**Complexity:** ⭐⭐⭐ Complex — Requires state lookup, channel validation, optional file I/O.

**State Management:**
- Lookup capture by ID
- Validate channels exist on capture's device configuration
- Optionally: write CSV files (digital_channels.csv, analog_channels.csv) to directory
- If capture not found, return `codes.InvalidArgument`

**Design Decision:** Start with just validation (no file I/O), add file writing if tests need it.

---

#### `salad export raw-binary`

**CLI File:** `cmd/salad/cmd/export.go` (lines 76-105)  
**Client Method:** `c.ExportRawDataBinary(ctx, captureID, directory, channels, analogDownsample)` (line 98)  
**gRPC Method:** `Manager.ExportRawDataBinary`

**Request/Response:**
- Request: `ExportRawDataBinaryRequest{
    CaptureId: uint64,
    Directory: string,
    Channels: *LogicChannels,
    AnalogDownsampleRatio: uint64
  }`
- Response: `ExportRawDataBinaryReply{}` (empty)

**What the Command Does:**
1. Parses `--digital` and `--analog` flags into `LogicChannels`
2. Creates client connection
3. Calls `ExportRawDataBinary` with all parameters
4. Prints: `ok\n` on success

**Mock Server Requirements:**
- ✅ Must validate capture ID exists
- ✅ Must accept directory path
- ✅ Must accept channel selection (`LogicChannels`)
- ✅ Must accept analog downsample ratio (1-1,000,000)
- ✅ Optionally: actually write binary files to directory

**Complexity:** ⭐⭐⭐ Complex — Similar to raw-csv, but binary format.

**State Management:**
- Same as `ExportRawDataCsv`, but binary file format
- Optionally: write binary files (digital.bin, analog.bin) to directory

---

## Summary: Mock Server Implementation Priority

### Phase 1: Discovery (Required for Basic Testing)
1. ✅ `GetAppInfo` — Simple, no state
2. ✅ `GetDevices` — Simple, just return configured list

### Phase 2: Capture Lifecycle (Required for Capture Commands)
3. ✅ `LoadCapture` — Medium, requires capture ID generation
4. ✅ `SaveCapture` — Medium, requires capture lookup
5. ✅ `StopCapture` — Medium, requires capture lookup + status update
6. ✅ `WaitCapture` — Complex, requires mode-specific logic
7. ✅ `CloseCapture` — Medium, requires capture lookup + cleanup

### Phase 3: Export (Required for Export Commands)
8. ✅ `ExportRawDataCsv` — Complex, requires capture lookup + optional file I/O
9. ✅ `ExportRawDataBinary` — Complex, similar to CSV

### Phase 4: Future Commands (Not Yet Implemented)
- `StartCapture` — Will be needed for ticket 002
- `AddAnalyzer` / `RemoveAnalyzer` — Will be needed for ticket 003
- `AddHighLevelAnalyzer` / `RemoveHighLevelAnalyzer` — Will be needed for ticket 004
- `ExportDataTableCsv` — Will be needed for ticket 005
- `LegacyExportAnalyzer` — Legacy, low priority

## Mock Server State Requirements

### Devices
- **Storage:** `[]*pb.Device` (slice)
- **Operations:** Add device, list devices (filter by simulation flag)
- **Initialization:** Empty list (tests add devices)

### Captures
- **Storage:** `map[uint64]*CaptureState`
- **Operations:** Create (load/start), lookup, update status, delete (close)
- **CaptureState fields:**
  - `ID`: uint64
  - `DeviceID`: string
  - `Config`: `*pb.CaptureConfiguration`
  - `DeviceConfig`: `*pb.LogicDeviceConfiguration`
  - `Status`: enum (running, stopped, completed)
  - `StartTime`: time.Time (for timed captures)
- **ID Generation:** Atomic counter starting at 1

### Analyzers (Future)
- **Storage:** `map[string]*AnalyzerState` (key: "capture_id:analyzer_id")
- **Operations:** Add, remove, lookup
- **Initialization:** Empty map

## Error Handling Requirements

All RPCs must return appropriate gRPC status codes:

- `codes.InvalidArgument`: Invalid request (missing required fields, invalid capture ID)
- `codes.NotFound`: Resource not found (device not found, capture not found)
- `codes.DeadlineExceeded`: Timeout (for WaitCapture if still running)
- `codes.Internal`: Unexpected server error (shouldn't happen in mock, but good to have)

## Testing Considerations

### What Tests Need
1. **State Inspection:** Tests need to query mock server state (e.g., "did capture get created?")
2. **State Configuration:** Tests need to set up initial state (e.g., "add a device")
3. **Error Injection:** Tests might want to simulate errors (e.g., "return error on next GetDevices call")
4. **File Verification:** Export tests might want to verify files were created

### Mock Server API for Tests
```go
// State inspection
func (s *Server) GetCapture(id uint64) (*CaptureState, error)
func (s *Server) GetDevices() []*pb.Device

// State configuration
func (s *Server) AddDevice(device *pb.Device)
func (s *Server) SetAppInfo(info *pb.AppInfo)

// Error injection (future)
func (s *Server) SetErrorForNextCall(method string, err error)
```

## Next Steps

1. Create tasks for implementing each RPC handler
2. Prioritize Phase 1 and Phase 2 (discovery + capture lifecycle)
3. Implement test helper functions for common test scenarios
4. Document mock server limitations (e.g., WaitCapture doesn't block)
