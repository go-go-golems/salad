---
Title: 'Implementation: capture start (manual/timed/trigger)'
Ticket: 002-CAPTURE-START
Status: active
Topics:
    - go
    - saleae
    - logic-analyzer
    - client
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Implementation approach for `salad capture start` (manual/timed/digital-trigger) using Logic2 Automation API StartCaptureRequest and a config-file driven UX."
LastUpdated: 2025-12-24T22:42:11.995985586-05:00
WhatFor: ""
WhenToUse: ""
---

# Implementation: capture start (manual/timed/trigger)

## Goal

Implement a robust, scriptable `salad capture start` command that can start captures in:
- **manual** mode (ring buffer semantics, stop manually)
- **timed** mode (capture ends after duration)
- **digital-trigger** mode (end after trigger condition)

## Proto grounding

Relevant RPC and message types in `proto/saleae/grpc/saleae.proto`:
- `Manager.StartCapture(StartCaptureRequest) returns (StartCaptureReply)`
- `StartCaptureRequest`:
  - `device_id` (optional; default to first physical device)
  - `device_configuration` → `LogicDeviceConfiguration`
  - `capture_configuration` → `CaptureConfiguration`
- `LogicDeviceConfiguration`:
  - `enabled_channels` → `LogicChannels` (digital + analog index lists)
  - `digital_sample_rate`, `analog_sample_rate`
  - `digital_threshold_volts`
  - `glitch_filters` (`GlitchFilterEntry`)
- `CaptureConfiguration`:
  - `buffer_size_megabytes`
  - `capture_mode` oneof:
    - `ManualCaptureMode` (trim)
    - `TimedCaptureMode` (duration + trim)
    - `DigitalTriggerCaptureMode` (trigger type/channel/pulse bounds + linked channel conditions + after-trigger + trim)

## CLI surface

### Primary UX (config-first)

To avoid flag explosion and keep it reproducible, implement:
- `salad capture start --config /abs/path/capture.yaml`

Optionally allow light overrides (later):
- `--device-id ...`
- `--out-capture-id-file ...`

### Example config (YAML)

```yaml
device:
  device_id: ""            # optional, default: first physical
  channels:
    digital: [0,1,2,3]
    analog:  []
  digital_sample_rate: 10000000
  analog_sample_rate:  0
  digital_threshold_volts: 1.8
  glitch_filters:
    - channel_index: 0
      pulse_width_seconds: 0.00000005

capture:
  buffer_size_megabytes: 128
  mode: timed              # manual|timed|digital-trigger
  timed:
    duration_seconds: 2.0
    trim_data_seconds: 0
```

Trigger example:

```yaml
capture:
  buffer_size_megabytes: 256
  mode: digital-trigger
  digital_trigger:
    trigger_type: rising           # rising|falling|pulse-high|pulse-low
    trigger_channel_index: 0
    after_trigger_seconds: 0.25
    trim_data_seconds: 0
    linked_channels:
      - channel_index: 1
        state: high                # high|low
```

## Implementation approach

### Parsing and validation

- Parse YAML (and JSON) into an internal struct:
  - `internal/config/capture_start.go`
- Validate early with helpful error messages:
  - at least one channel in digital/analog
  - sample rates sensible (non-zero where required)
  - trigger config matches mode (pulse bounds only valid for pulse triggers)

### Building proto requests

- Convert config → `pb.StartCaptureRequest` (generated `pb` package)
- Explicitly set oneofs:
  - `LogicDeviceConfiguration.EnabledChannels = &pb.LogicDeviceConfiguration_LogicChannels{...}`
  - `CaptureConfiguration.CaptureMode = &pb.CaptureConfiguration_TimedCaptureMode{...}` etc.

### Control flow

- Dial via existing `internal/saleae.New(...)`.
- Call `StartCapture`.
- Print `capture_id=...`.
- If `--wait` is present (optional future), call `WaitCapture` automatically for timed/trigger modes.

## Files likely to change/add

- `cmd/salad/cmd/capture_start.go` (new Cobra command)
- `internal/saleae/client.go` (add `StartCapture` wrapper; may reuse existing helpers)
- `internal/config/` (new config parsing + validation)
- `playbook/` doc for capture-start examples

## Testing strategy

- Unit tests for config parsing and mapping to proto:
  - invalid modes, missing channels, bad trigger values
- Manual smoke test (requires Logic2 running):
  - start timed capture for 1s and wait
  - verify capture id returned

## Open questions / decisions

- Default sample rates: do we require explicit config or pick a safe default per device type?
- Should `--config` be required, or do we also provide a “flag-only” mode for quick use?
