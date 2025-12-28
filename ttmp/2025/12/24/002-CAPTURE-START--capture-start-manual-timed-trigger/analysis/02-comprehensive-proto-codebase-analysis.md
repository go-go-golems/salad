---
Title: 'Comprehensive Analysis: StartCapture Proto and Codebase Integration'
Ticket: 002-CAPTURE-START
Status: active
Topics:
    - go
    - saleae
    - logic-analyzer
    - client
    - proto
    - config-parsing
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: proto/saleae/grpc/saleae.proto
      Note: Source proto defining StartCapture RPC and all message types
    - Path: gen/saleae/automation/saleae.pb.go
      Note: Generated Go types for StartCaptureRequest/Reply and nested messages
    - Path: gen/saleae/automation/saleae_grpc.pb.go
      Note: Generated gRPC client stubs including Manager.StartCapture
    - Path: internal/saleae/client.go
      Note: Existing client wrapper pattern to follow for StartCapture
    - Path: cmd/salad/cmd/capture.go
      Note: Existing capture commands showing CLI structure
    - Path: cmd/salad/cmd/export.go
      Note: Example of complex flag parsing (channels) and proto building
    - Path: cmd/salad/cmd/util.go
      Note: Helper functions for parsing CSV channel lists
ExternalSources: []
Summary: "Deep dive into StartCapture proto structures, existing codebase patterns, config parsing approach, and implementation strategy for capture start command."
LastUpdated: 2025-12-27T00:00:00Z
WhatFor: ""
WhenToUse: ""
---

# Comprehensive Analysis: StartCapture Proto and Codebase Integration

## Executive Summary

This guide explains how to implement `salad capture start`, the command that initiates logic analyzer captures through Saleae Logic 2's Automation API. Unlike the existing capture commands (`load`, `save`, `stop`) which work with already-running captures, `capture start` is where we **create** a new capture by configuring the device, selecting channels, and choosing a capture mode.

**Who this is for:** Senior engineers implementing the capture start feature. We assume familiarity with Go, protobuf, and gRPC, but not prior knowledge of Saleae's proto structures or the salad codebase patterns.

**What you'll learn:** The complete structure of `StartCaptureRequest` (including nested oneof fields), how existing commands in `cmd/salad/cmd/` handle similar complexity, and why we're choosing a config-file approach over flag explosion. By the end, you'll understand not just what to implement, but why each design decision matters.

**The challenge:** `StartCapture` has deep nesting (device config → channels → sample rates, capture config → mode → trigger conditions) and uses protobuf oneofs extensively. Getting the oneof wrapper types wrong will compile but fail at runtime. We'll show you exactly how to set them correctly by examining the generated code in `gen/saleae/automation/saleae.pb.go`.

## Mental Model: What StartCapture Actually Does

Before diving into proto structures, let's understand what `StartCapture` represents conceptually. When you call this RPC, you're telling Logic 2: "Start recording signals on these channels, at these sample rates, and stop when this condition is met."

The request has three main parts:

1. **Device selection**: Which physical device to use (or "first available")
2. **Device configuration**: Which channels to record, at what sample rates, with what thresholds and filters
3. **Capture configuration**: How much memory to use, and when to stop (manual, timed, or trigger-based)

**Why this matters:** The proto uses oneof fields to represent mutually exclusive choices. For example, you can't have both a timed capture mode and a trigger mode active simultaneously. Understanding this constraint helps explain why the generated Go code uses wrapper types—they enforce this exclusivity at compile time.

The response is simple: a `capture_id` (uint64) that you use with other commands like `capture stop`, `export raw-csv`, or `analyzer add`. This ID is your handle for the entire capture lifecycle.

## Understanding the Proto Structure

### The RPC Definition

The `StartCapture` RPC lives in the `Manager` service, defined in `proto/saleae/grpc/saleae.proto` at line 36:

```proto
rpc StartCapture(StartCaptureRequest) returns (StartCaptureReply) {}
```

In Go, this becomes a method on `ManagerClient` (generated in `gen/saleae/automation/saleae_grpc.pb.go:117`):

```go
StartCapture(ctx context.Context, in *StartCaptureRequest, opts ...grpc.CallOption) (*StartCaptureReply, error)
```

**Why we care:** This is the entry point. Everything else builds up to constructing a valid `StartCaptureRequest` and calling this method. The reply contains a `CaptureInfo` with the `capture_id` we need.

### StartCaptureRequest: The Top Level

The request structure (`proto/saleae/grpc/saleae.proto:362-373`, Go type at `gen/saleae/automation/saleae.pb.go:1364`) has three fields:

```go
type StartCaptureRequest struct {
    DeviceId string  // Optional; empty = first physical device
    
    DeviceConfiguration isStartCaptureRequest_DeviceConfiguration  // Oneof wrapper
    
    CaptureConfiguration *CaptureConfiguration  // Required
}
```

**Key insight:** The `device_id` is optional. If you pass an empty string, Logic 2 picks the first physical (non-simulation) device. This is convenient for scripts, but you lose explicit control. The proto comment at line 364-365 explains: "If a device id is not specified, the first physical device will be used. If no physical device is connected, an error will be returned."

**The oneof trap:** `DeviceConfiguration` is a oneof interface type (`isStartCaptureRequest_DeviceConfiguration`). You can't just assign a `LogicDeviceConfiguration` directly. You must wrap it in `*StartCaptureRequest_LogicDeviceConfiguration`. We'll see this pattern repeatedly—it's the most common mistake when working with these protos.

### LogicDeviceConfiguration: Channels, Rates, and Filters

The device configuration (`proto/saleae/grpc/saleae.proto:219-238`, Go type at `gen/saleae/automation/saleae.pb.go:737`) tells Logic 2 which channels to record and how:

```go
type LogicDeviceConfiguration struct {
    EnabledChannels isLogicDeviceConfiguration_EnabledChannels  // Oneof: LogicChannels
    
    DigitalSampleRate uint32  // Samples per second
    AnalogSampleRate  uint32  // Samples per second
    
    DigitalThresholdVolts float64  // 1.2, 1.8, or 3.3 (Pro 8/16 only)
    
    GlitchFilters []*GlitchFilterEntry
}
```

**The channels oneof:** `EnabledChannels` is another oneof, but currently only has one variant: `LogicChannels`. You set it like this:

```go
deviceConfig := &pb.LogicDeviceConfiguration{
    EnabledChannels: &pb.LogicDeviceConfiguration_LogicChannels{
        LogicChannels: &pb.LogicChannels{
            DigitalChannels: []uint32{0, 1, 2},
            AnalogChannels:  []uint32{0},
        },
    },
    DigitalSampleRate: 10000000,
    AnalogSampleRate:  1000000,
}
```

**Why this nesting:** The oneof wrapper (`LogicDeviceConfiguration_LogicChannels`) allows the proto to evolve—if Saleae adds other channel types later, they can add new variants without breaking existing code. For now, we always use `LogicChannels`.

**Sample rate validation:** You must specify a non-zero sample rate for any channel type you enable. If you enable digital channels but set `DigitalSampleRate` to 0, Logic 2 will reject the request. This is a common validation error we'll need to catch early.

**Threshold volts gotcha:** `DigitalThresholdVolts` only matters for Logic Pro 8 and Logic Pro 16 devices. For other devices (Logic 8, etc.), this field is ignored. Valid values are 1.2, 1.8, or 3.3. If you're writing generic code, you might want to query the device type first (via `GetDevices`) to validate this field.

**Glitch filters:** These filter out pulses shorter than a threshold. Each entry specifies a channel index and a minimum pulse width in seconds. For example, filtering 50ns glitches on channel 0:

```go
glitchFilters := []*pb.GlitchFilterEntry{
    {
        ChannelIndex:     0,
        PulseWidthSeconds: 0.00000005,  // 50 nanoseconds
    },
}
```

### CaptureConfiguration: Buffer Size and Mode

The capture configuration (`proto/saleae/grpc/saleae.proto:328-345`, Go type at `gen/saleae/automation/saleae.pb.go:1159`) controls memory usage and stop conditions:

```go
type CaptureConfiguration struct {
    BufferSizeMegabytes uint32  // Max MB for data storage
    
    CaptureMode isCaptureConfiguration_CaptureMode  // Oneof: Manual, Timed, or DigitalTrigger
}
```

**Buffer behavior depends on mode:** This is critical to understand. When the buffer fills up:
- **Manual mode**: Oldest data is deleted (ring buffer behavior)
- **Timed/Trigger modes**: Capture terminates immediately

This difference exists because manual captures are meant to run indefinitely until you stop them, while timed/trigger captures have a natural end point.

**The capture mode oneof:** This is where the complexity lives. You can choose one of three modes, each with different fields:

#### Manual Capture Mode

```go
type ManualCaptureMode struct {
    TrimDataSeconds float64  // Seconds to keep after stop (0 = no trim)
}
```

**When to use:** For interactive debugging where you want to capture until you manually stop it. The ring buffer behavior means you can capture for hours without running out of memory—old data gets discarded.

**Trim gotcha:** `TrimDataSeconds` only applies *after* you call `StopCapture`. If set to 0, you keep everything. If set to 5.0, you keep only the last 5 seconds before the stop command. This is useful for capturing a long session but only analyzing the end.

#### Timed Capture Mode

```go
type TimedCaptureMode struct {
    DurationSeconds float64  // How long to capture
    TrimDataSeconds float64  // Seconds to keep after completion (0 = no trim)
}
```

**When to use:** For reproducible captures of fixed duration. Common for automated testing or when you know exactly how long your signal lasts.

**Completion:** The capture stops automatically after `DurationSeconds`. You can call `WaitCapture` to block until it finishes. This is synchronous—the RPC returns when the capture ends.

#### Digital Trigger Capture Mode

This is the most complex mode (`proto/saleae/grpc/saleae.proto:300-326`, Go type at `gen/saleae/automation/saleae.pb.go:1057`):

```go
type DigitalTriggerCaptureMode struct {
    TriggerType         DigitalTriggerType              // RISING, FALLING, PULSE_HIGH, PULSE_LOW
    AfterTriggerSeconds float64                         // Continue capturing after trigger
    TrimDataSeconds     float64                         // Seconds to keep after completion
    TriggerChannelIndex uint32                          // Which channel to watch
    MinPulseWidthSeconds float64                        // Only for pulse triggers
    MaxPulseWidthSeconds float64                        // Only for pulse triggers
    LinkedChannels      []*DigitalTriggerLinkedChannel  // Optional conditions on other channels
}
```

**Trigger types:** The enum (`gen/saleae/automation/saleae.pb.go:340`) has four values:
- `RISING`: Low-to-high transition
- `FALLING`: High-to-low transition
- `PULSE_HIGH`: Rising edge followed by falling edge
- `PULSE_LOW`: Falling edge followed by rising edge

**Pulse width validation:** `MinPulseWidthSeconds` and `MaxPulseWidthSeconds` only apply to pulse triggers (`PULSE_HIGH` or `PULSE_LOW`). For edge triggers, these fields are ignored. This is a validation rule we must enforce—setting pulse width bounds on an edge trigger won't cause a proto error, but it's semantically wrong.

**Linked channels:** These add conditions on other digital channels. For edge triggers, the linked channel must be in the specified state (HIGH or LOW) when the trigger edge occurs. For pulse triggers, the linked channel must maintain that state for the duration of the pulse. This allows complex trigger conditions like "rising edge on channel 0, but only if channel 1 is HIGH."

**After-trigger capture:** `AfterTriggerSeconds` controls how long to keep recording after the trigger fires. This is useful for capturing context around an event. For example, if you're debugging a boot sequence, you might trigger on a reset pulse but capture 100ms after it to see initialization.

### Setting the Capture Mode Oneof

Here's how you set each mode in Go:

**Manual mode:**
```go
captureConfig := &pb.CaptureConfiguration{
    BufferSizeMegabytes: 128,
    CaptureMode: &pb.CaptureConfiguration_ManualCaptureMode{
        ManualCaptureMode: &pb.ManualCaptureMode{
            TrimDataSeconds: 0,
        },
    },
}
```

**Timed mode:**
```go
captureConfig := &pb.CaptureConfiguration{
    BufferSizeMegabytes: 128,
    CaptureMode: &pb.CaptureConfiguration_TimedCaptureMode{
        TimedCaptureMode: &pb.TimedCaptureMode{
            DurationSeconds: 2.0,
            TrimDataSeconds: 0,
        },
    },
}
```

**Trigger mode:**
```go
captureConfig := &pb.CaptureConfiguration{
    BufferSizeMegabytes: 256,
    CaptureMode: &pb.CaptureConfiguration_DigitalCaptureMode{
        DigitalCaptureMode: &pb.DigitalTriggerCaptureMode{
            TriggerType:         pb.DigitalTriggerType_DIGITAL_TRIGGER_TYPE_RISING,
            AfterTriggerSeconds: 0.25,
            TrimDataSeconds:     0,
            TriggerChannelIndex: 0,
            // Min/Max pulse width only for pulse triggers
            LinkedChannels: []*pb.DigitalTriggerLinkedChannel{
                {
                    ChannelIndex: 1,
                    State:        pb.DigitalTriggerLinkedChannelState_DIGITAL_TRIGGER_LINKED_CHANNEL_STATE_HIGH,
                },
            },
        },
    },
}
```

**The pattern:** Notice how each mode uses a different wrapper type (`CaptureConfiguration_ManualCaptureMode`, `CaptureConfiguration_TimedCaptureMode`, `CaptureConfiguration_DigitalCaptureMode`). The wrapper type name follows the pattern `*ParentMessageName_FieldName`, and it contains the actual message type.

### StartCaptureReply: Getting the Capture ID

The reply is simple (`proto/saleae/grpc/saleae.proto:374`, Go type at `gen/saleae/automation/saleae.pb.go:1449`):

```go
type StartCaptureReply struct {
    CaptureInfo *CaptureInfo
}

type CaptureInfo struct {
    CaptureId uint64
}
```

**Usage:** Extract the `capture_id` and use it with other commands. For example:
- `salad capture stop --capture-id <id>`
- `salad capture wait --capture-id <id>` (for timed/trigger modes)
- `salad export raw-csv --capture-id <id> ...`

**Nil check required:** Always check that `reply.GetCaptureInfo()` is not nil before accessing `CaptureId`. The proto doesn't guarantee non-nil, and a nil pointer dereference here would be a runtime panic.

## Existing Codebase Patterns

### How Client Wrappers Work

The existing client wrapper in `internal/saleae/client.go` follows a consistent pattern that we should replicate. Let's examine `LoadCapture` as an example:

```go
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
```

**Why this pattern:** Each method validates inputs before making the RPC call (fail fast), wraps errors with the operation name (easier debugging), checks reply fields for nil (defensive programming), and returns Go-native types (uint64 instead of proto types). This makes the client API cleaner than working with proto types directly.

**For StartCapture:** We'll follow the same pattern, but our validation will be more complex because we need to validate the entire nested structure. We'll do this in a separate validation function that we can unit test.

### How CLI Commands Are Structured

Existing capture commands in `cmd/salad/cmd/capture.go` show the CLI pattern. Here's `captureLoadCmd`:

```go
var captureLoadCmd = &cobra.Command{
    Use:   "load",
    Short: "Load a .sal capture file",
    RunE: func(cmd *cobra.Command, args []string) error {
        ctx := cmd.Context()
        if timeout > 0 {
            var cancel context.CancelFunc
            ctx, cancel = context.WithTimeout(ctx, timeout)
            defer cancel()
        }

        c, err := saleae.New(ctx, saleae.Config{Host: host, Port: port, Timeout: timeout})
        if err != nil {
            return err
        }
        defer c.Close()

        id, err := c.LoadCapture(ctx, filepath)
        if err != nil {
            return err
        }

        _, err = fmt.Fprintf(cmd.OutOrStdout(), "capture_id=%d\n", id)
        return errors.Wrap(err, "write output")
    },
}
```

**The flow:** Extract context → apply timeout → create client → defer close → call client method → format output. This is boilerplate, but it's consistent across all commands. For `capture start`, we'll add config parsing before the client call.

**Output format:** Commands output `capture_id=<number>` to stdout. This is machine-parseable and can be piped to other commands. Future work (ticket 007) will add `--json` output, but for now we keep it simple.

### Parsing Complex Inputs: The Channel Example

The export commands in `cmd/salad/cmd/export.go` show how to handle complex flag parsing. They use a helper function `parseUint32CSV` from `cmd/salad/cmd/util.go`:

```go
func parseUint32CSV(s string) ([]uint32, error) {
    s = strings.TrimSpace(s)
    if s == "" {
        return nil, nil
    }
    parts := strings.Split(s, ",")
    out := make([]uint32, 0, len(parts))
    for _, p := range parts {
        p = strings.TrimSpace(p)
        if p == "" {
            continue
        }
        v, err := strconv.ParseUint(p, 10, 32)
        if err != nil {
            return nil, errors.Wrapf(err, "parse channel %q", p)
        }
        out = append(out, uint32(v))
    }
    return out, nil
}
```

**Why this approach:** The helper handles edge cases (empty strings, whitespace, invalid numbers) and returns clear errors. For `capture start`, we won't use CSV flags—we'll use a config file instead—but the validation principles are the same: parse, validate, provide clear errors.

## Config File Strategy: Why Not Flags?

**The problem:** `StartCaptureRequest` has ~15 fields across nested structures. If we used flags, the command would look like:

```bash
salad capture start \
  --device-id "" \
  --digital-channels 0,1,2 \
  --analog-channels "" \
  --digital-sample-rate 10000000 \
  --analog-sample-rate 0 \
  --digital-threshold-volts 1.8 \
  --glitch-filter "0:0.00000005" \
  --buffer-size-mb 128 \
  --mode timed \
  --duration-seconds 2.0 \
  --trim-data-seconds 0
```

This is unreadable, error-prone, and hard to reuse. For trigger mode, it gets worse with pulse width bounds and linked channels.

**The solution:** A config file (YAML or JSON) that groups related fields and can be version-controlled, shared, and reused. The command becomes:

```bash
salad capture start --config capture.yaml
```

**Trade-off:** We lose the convenience of quick one-liners, but we gain reproducibility and maintainability. If users need quick captures, they can create simple config templates.

### Flag Overrides: Future Enhancement

**Note:** This is a design for future work. The initial implementation will use config files only. Flag overrides can be added later based on user feedback.

While config files are the primary interface, there's value in allowing flags to override specific config values. This enables quick tweaks without editing files: "use my standard config, but change the duration to 5 seconds."

**Flag naming convention:** Flags mirror the YAML hierarchy using `--LAYER-FIELD(-SUBFIELD...)` pattern. The two top-level layers are `device` and `capture`, matching the config structure.

**Precedence:** Config file values are loaded first, then flags override them. This means you can have a base config with sensible defaults, then override only what you need for a specific run.

**Example flags:**

```bash
# Device layer flags
--device-device-id "DEVICE123"
--device-channels-digital "0,1,2,3"      # CSV format, like existing export commands
--device-channels-analog "0,1"          # CSV format
--device-digital-sample-rate 10000000
--device-analog-sample-rate 1000000
--device-digital-threshold-volts 1.8

# Capture layer flags
--capture-buffer-size-megabytes 256
--capture-mode timed                    # Overrides mode, which affects which sub-flags are valid

# Mode-specific flags (only valid when mode matches)
--capture-timed-duration-seconds 5.0
--capture-timed-trim-data-seconds 1.0

# For trigger mode
--capture-digital-trigger-trigger-type rising
--capture-digital-trigger-trigger-channel-index 0
--capture-digital-trigger-after-trigger-seconds 0.25
--capture-digital-trigger-min-pulse-width-seconds 0.00000001  # Only for pulse triggers
--capture-digital-trigger-max-pulse-width-seconds 0.000001    # Only for pulse triggers
```

**Why this naming:** The `--LAYER-FIELD` pattern makes it clear which config section each flag affects. Nested fields use additional dashes (`--capture-timed-duration-seconds`), preserving the YAML hierarchy. This is verbose, but explicit—you always know where a flag maps in the config structure.

**Implementation approach (when built):**

1. Load config file (if `--config` provided)
2. Parse flag overrides into a partial config structure
3. Merge flags into config (flags win on conflicts)
4. Validate merged config
5. Convert to proto

**Complexity considerations:**

- **Array fields:** Channels and glitch filters need special handling. For channels, CSV parsing (like `parseUint32CSV` in `cmd/salad/cmd/util.go`) works. For glitch filters, we might need JSON/YAML parsing or a structured format like `"channel:width"` pairs.

- **Mode switching:** Changing `--capture-mode` should clear mode-specific fields from the config. For example, if config has `timed.duration_seconds` but you pass `--capture-mode manual`, the timed fields should be ignored.

- **Validation timing:** We must validate after merging, not before. A config might be valid, flags might be valid, but the merge might create an invalid combination (e.g., enabling digital channels but overriding sample rate to 0).

**When to build this:** Wait for user feedback. If users frequently need to override configs, flags become valuable. If they always use full configs or create variants, flags add complexity without benefit. Start with config-only, add flags if needed.

### Config File Structure

Here's a YAML example for a timed capture:

```yaml
device:
  device_id: ""  # Empty = first physical device
  channels:
    digital: [0, 1, 2, 3]
    analog: []
  digital_sample_rate: 10000000
  analog_sample_rate: 0
  digital_threshold_volts: 1.8
  glitch_filters:
    - channel_index: 0
      pulse_width_seconds: 0.00000005

capture:
  buffer_size_megabytes: 128
  mode: timed  # manual|timed|digital-trigger
  timed:
    duration_seconds: 2.0
    trim_data_seconds: 0
```

**Why this structure:** It mirrors the proto structure, making conversion straightforward. The `mode` field determines which subsection (`timed`, `manual`, `digital_trigger`) is used. This is explicit—you can't accidentally set conflicting modes.

**JSON support:** We'll support both YAML and JSON. YAML is more readable for humans, JSON is easier to generate programmatically. The parser detects the format from the file extension.

### Implementation: Config Parsing Package

We'll create `internal/config/capture_start.go` with:

1. **Config structs** mirroring the YAML/JSON structure
2. **Load function** that reads and parses the file
3. **Validate function** that checks all rules (channels, sample rates, mode-specific fields)
4. **ToProto function** that converts to `*pb.StartCaptureRequest`

**Why separate package:** Config parsing is reusable (future commands might use configs), testable (unit tests don't need gRPC), and keeps the CLI command simple.

**Validation complexity:** We need to validate:
- At least one channel enabled
- Sample rates match enabled channels
- Mode-specific fields present
- Pulse width bounds only for pulse triggers
- Enum conversions (trigger type strings → proto enum)

This is where most bugs will hide, so we'll write comprehensive unit tests.

### Converting Config to Proto: The Oneof Challenge

The `ToProto` function must correctly set all oneof fields. Here's the pattern for device configuration:

```go
deviceConfig := &pb.LogicDeviceConfiguration{
    EnabledChannels: &pb.LogicDeviceConfiguration_LogicChannels{
        LogicChannels: &pb.LogicChannels{
            DigitalChannels: c.Device.Channels.Digital,
            AnalogChannels:  c.Device.Channels.Analog,
        },
    },
    DigitalSampleRate:     c.Device.DigitalSampleRate,
    AnalogSampleRate:      c.Device.AnalogSampleRate,
    DigitalThresholdVolts: c.Device.DigitalThresholdVolts,
    GlitchFilters:         glitchFilters,
}
```

And for capture mode (timed example):

```go
captureConfig := &pb.CaptureConfiguration{
    BufferSizeMegabytes: c.Capture.BufferSizeMegabytes,
    CaptureMode: &pb.CaptureConfiguration_TimedCaptureMode{
        TimedCaptureMode: &pb.TimedCaptureMode{
            DurationSeconds: c.Capture.Timed.DurationSeconds,
            TrimDataSeconds: c.Capture.Timed.TrimDataSeconds,
        },
    },
}
```

**The gotcha:** If you forget the wrapper type and try to assign directly, Go will compile it, but the oneof field won't be set correctly. The proto library uses the wrapper type to determine which variant is active. Always use the wrapper pattern.

## Implementation Plan

### Phase 1: Dependencies and Config Infrastructure

**What:** Add YAML parsing library and create the config package.

**Why first:** Config parsing is independent of gRPC and can be tested in isolation. Getting this right prevents bugs in later phases.

**Steps:**
1. Add `gopkg.in/yaml.v3` to `go.mod`
2. Create `internal/config/capture_start.go` with structs, parsing, validation, and conversion
3. Write unit tests for parsing (YAML and JSON), validation (all error cases), and proto conversion

**Success criteria:** Unit tests pass, config files parse correctly, validation catches all invalid inputs.

### Phase 2: Client Wrapper

**What:** Add `StartCapture` method to `internal/saleae/client.go`.

**Why second:** The client wrapper is the bridge between config and gRPC. It's simpler than the CLI (no flag parsing), but more complex than existing methods (nested validation).

**Steps:**
1. Add method signature: `StartCapture(ctx context.Context, req *pb.StartCaptureRequest) (uint64, error)`
2. Follow existing pattern: context handling, error wrapping, nil checks
3. Return `capture_id` (uint64) for consistency with other methods

**Success criteria:** Method compiles, follows existing patterns, handles errors correctly.

### Phase 3: CLI Command

**What:** Create `cmd/salad/cmd/capture_start.go` with `--config` flag.

**Why third:** The CLI ties everything together. By this point, config parsing and client wrapper are tested, so CLI bugs are easier to isolate.

**Steps:**
1. Create Cobra command following `captureLoadCmd` pattern
2. Add `--config` flag (required)
3. Parse config → validate → convert to proto → call client → output `capture_id`
4. **Defer flag overrides:** Don't implement `--device-*` or `--capture-*` flags yet. Start with config-only to validate the approach.

**Success criteria:** Command works end-to-end with real Logic 2 instance using config files.

### Phase 4: Testing and Documentation

**What:** Manual smoke tests and example config files.

**Why last:** Integration testing requires a running Logic 2 instance, which is harder to automate. Manual tests verify the happy path, edge cases, and error handling.

**Steps:**
1. Test all three capture modes (manual, timed, trigger)
2. Test error cases (invalid config, missing device, etc.)
3. Create example config files for each mode
4. Document common pitfalls

## Common Pitfalls and Gotchas

**Oneof wrapper types:** Always use wrapper types (`*TypeName_FieldName`) when setting oneof fields. Direct assignment compiles but doesn't work.

**Sample rate validation:** You must set a non-zero sample rate for any channel type you enable. This validation must happen before the RPC call.

**Pulse width bounds:** `MinPulseWidthSeconds` and `MaxPulseWidthSeconds` only apply to pulse triggers. Don't set them for edge triggers, even though the proto allows it.

**Buffer behavior:** Manual mode uses ring buffer (old data deleted), timed/trigger modes terminate when full. This affects how you size the buffer.

**Device ID empty string:** An empty `device_id` means "first physical device." This is convenient but implicit—consider making it explicit in configs.

**Nil reply fields:** Always check `reply.GetCaptureInfo()` for nil before accessing `CaptureId`. The proto doesn't guarantee non-nil.

**Context handling:** Use `cmd.Context()` in CLI commands, but handle nil contexts in client methods (default to `context.Background()`).

## References

- **Proto source:** `proto/saleae/grpc/saleae.proto` (lines 36, 219-238, 300-326, 328-345, 362-374)
- **Generated Go types:** `gen/saleae/automation/saleae.pb.go` (search for `StartCaptureRequest`, `LogicDeviceConfiguration`, `CaptureConfiguration`)
- **Generated gRPC stubs:** `gen/saleae/automation/saleae_grpc.pb.go` (line 117 for `StartCapture` method)
- **Client wrapper pattern:** `internal/saleae/client.go` (see `LoadCapture` method)
- **CLI command pattern:** `cmd/salad/cmd/capture.go` (see `captureLoadCmd`)
- **Helper utilities:** `cmd/salad/cmd/util.go` (see `parseUint32CSV`)
