---
Title: 'Diary: Capture Start Implementation'
Ticket: 002-CAPTURE-START
Status: active
Topics:
    - go
    - saleae
    - logic-analyzer
    - client
    - implementation
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Step-by-step narrative of implementing capture start functionality, documenting decisions, learnings, and challenges."
LastUpdated: 2025-12-27T00:00:00Z
WhatFor: ""
WhenToUse: ""
---

# Diary: Capture Start Implementation

## Goal

Document the implementation journey for `salad capture start` command, including proto analysis, config parsing design, code structure decisions, and testing approach. This diary captures what changed, why it changed, what happened (including failures), and what we learned.

## Step 1: Comprehensive Proto and Codebase Analysis

This step involved deep-diving into the `StartCapture` RPC proto structures, understanding how existing commands work in the codebase, and designing the config parsing approach. The goal was to create a complete technical reference before starting implementation.

**Commit (code):** N/A — Analysis phase

### What I did

- Read and analyzed `proto/saleae/grpc/saleae.proto` focusing on `StartCaptureRequest` and all nested message types
- Examined generated Go code in `gen/saleae/automation/saleae.pb.go` to understand Go type structures
- Reviewed existing command implementations (`capture.go`, `export.go`, `devices.go`) to understand patterns
- Studied client wrapper pattern in `internal/saleae/client.go`
- Analyzed helper utilities in `cmd/salad/cmd/util.go` for parsing patterns
- Created comprehensive analysis document covering:
  - Complete proto structure breakdown
  - Existing codebase patterns
  - Config file parsing strategy
  - Implementation plan
  - Error handling approach

### Why

- Need to understand proto oneof handling before implementing (critical for correctness)
- Existing commands show established patterns we should follow for consistency
- Config parsing is new infrastructure — need to design it properly
- Validation logic is complex (mode-specific rules, trigger types, etc.) — need to document requirements clearly

### What worked

- Proto structure is well-documented in comments
- Generated Go code follows predictable patterns
- Existing commands provide clear examples of:
  - Context handling
  - Error wrapping with `pkg/errors`
  - Flag parsing and validation
  - Output formatting

### What didn't work

- Initially tried to understand oneof handling from proto file alone — needed to look at generated Go code to see wrapper types
- Config parsing infrastructure doesn't exist — need to build from scratch (no existing YAML/JSON libraries in go.mod)

### What I learned

**Proto Oneof Handling:**
- Oneof fields in Go are accessed via wrapper types (`*TypeName_FieldName`)
- Must explicitly set wrapper type, not just the inner message
- Example: `EnabledChannels: &pb.LogicDeviceConfiguration_LogicChannels{LogicChannels: ...}`
- Access via `GetFieldName()` methods or type assertions

**Existing Patterns:**
- All client methods take `context.Context` as first parameter
- Methods validate inputs before RPC calls
- Error wrapping uses `pkg/errors` with operation name context
- CLI commands follow consistent structure: context → timeout → client → defer close → call → output

**Config Parsing:**
- No existing infrastructure in salad codebase
- Need to add YAML library (`gopkg.in/yaml.v3` recommended)
- JSON parsing available via standard library
- Should support both YAML and JSON for flexibility

**Validation Complexity:**
- Mode-specific validation rules (e.g., pulse triggers require pulse width bounds)
- Channel validation (at least one channel, sample rates must match enabled channels)
- Trigger type validation (enum conversion from strings)
- Linked channel state validation (high/low strings to enum)

### What was tricky to build

**Understanding Oneof Structures:**
- Proto oneofs generate wrapper types in Go that aren't immediately obvious from proto file
- Had to examine generated code to understand exact structure
- Wrapper types follow pattern: `*MessageName_FieldName` containing the actual message

**Config Structure Design:**
- Balancing YAML readability vs. Go struct simplicity
- Deciding between flat vs. nested structures
- Mode-specific sections (manual/timed/digital-trigger) — how to represent in config
- String-to-enum conversion for trigger types and linked channel states

**Validation Logic:**
- Mode-specific rules (pulse triggers need pulse width, edge triggers don't)
- Cross-field validation (channels → sample rates)
- Ensuring all required fields present for each mode

### What warrants a second pair of eyes

**Proto Oneof Handling:**
- Verify wrapper type usage is correct (especially `CaptureConfiguration.CaptureMode`)
- Check that we're setting oneof fields correctly (not just assigning inner message)

**Config Parsing:**
- Review struct tags for YAML/JSON mapping
- Validate that all proto fields are covered in config structs
- Check enum conversion logic (string → proto enum) for correctness

**Validation Logic:**
- Review all mode-specific validation rules
- Ensure error messages are clear and actionable
- Check edge cases (empty arrays, zero values, etc.)

**Error Handling:**
- Verify error wrapping provides enough context
- Check that validation errors happen before RPC calls (fail fast)

### What should be done in the future

**Config File Enhancements:**
- Consider supporting environment variable substitution in config files
- Add config file schema validation (JSON Schema or similar)
- Support config file includes/merging for reusable device configs

**Testing:**
- Unit tests for all validation rules
- Integration tests with real Logic 2 instance (requires test harness)
- Config file parsing tests (YAML and JSON, valid and invalid cases)

**Documentation:**
- Example config files for each capture mode
- Troubleshooting guide for common validation errors
- Migration guide if config format changes

**CLI Enhancements:**
- Add `--dry-run` flag to validate config without starting capture
- Add `--wait` flag to automatically call `WaitCapture` for timed/trigger modes
- Add `--json` output format (ticket 007)
- Consider flag-only mode for quick use cases (without config file)

### Code review instructions

**Start here:**
- `analysis/02-comprehensive-proto-codebase-analysis.md` — complete technical reference

**Key files to review:**
- `proto/saleae/grpc/saleae.proto:362-374` — StartCapture RPC and request/reply
- `proto/saleae/grpc/saleae.proto:219-345` — All nested message types
- `gen/saleae/automation/saleae.pb.go` — Generated Go types (especially oneof wrappers)
- `internal/saleae/client.go` — Existing client wrapper pattern
- `cmd/salad/cmd/capture.go` — Existing capture command structure

**How to validate:**
- Read analysis document sections on proto structure
- Compare oneof handling examples with generated code
- Review config parsing strategy against existing codebase patterns
- Check that implementation plan covers all proto fields

### Technical details

**Proto Oneof Examples:**

```go
// Setting LogicChannels in LogicDeviceConfiguration
deviceConfig := &pb.LogicDeviceConfiguration{
    EnabledChannels: &pb.LogicDeviceConfiguration_LogicChannels{
        LogicChannels: &pb.LogicChannels{
            DigitalChannels: []uint32{0, 1, 2},
            AnalogChannels:  []uint32{0},
        },
    },
}

// Setting capture mode (oneof)
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

**Config File Structure:**

```yaml
device:
  device_id: ""
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
  mode: timed
  timed:
    duration_seconds: 2.0
    trim_data_seconds: 0
```

**Enum Conversion:**

```go
// String to DigitalTriggerType enum
var triggerType pb.DigitalTriggerType
switch config.TriggerType {
case "rising":
    triggerType = pb.DigitalTriggerType_DIGITAL_TRIGGER_TYPE_RISING
case "falling":
    triggerType = pb.DigitalTriggerType_DIGITAL_TRIGGER_TYPE_FALLING
case "pulse-high":
    triggerType = pb.DigitalTriggerType_DIGITAL_TRIGGER_TYPE_PULSE_HIGH
case "pulse-low":
    triggerType = pb.DigitalTriggerType_DIGITAL_TRIGGER_TYPE_PULSE_LOW
default:
    return nil, errors.Errorf("invalid trigger_type: %q", config.TriggerType)
}
```

### What I'd do differently next time

- Start with generated Go code first, then refer back to proto file (Go code shows exact structure)
- Create example config files earlier in the process (helps validate design)
- Write validation tests before implementing validation logic (TDD approach)

