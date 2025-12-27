---
Title: 'Mock Server YAML DSL: Configurable Behavior Scenarios'
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
    - Path: proto/saleae/grpc/saleae.proto
      Note: Proto contract (requests/replies) that the DSL must be able to simulate
    - Path: cmd/salad/cmd/appinfo.go
      Note: CLI verb that exercises GetAppInfo
    - Path: cmd/salad/cmd/devices.go
      Note: CLI verb that exercises GetDevices and simulation filtering
    - Path: cmd/salad/cmd/capture.go
      Note: CLI verbs that exercise capture lifecycle RPCs
    - Path: cmd/salad/cmd/export.go
      Note: CLI verbs that exercise export RPCs and filesystem outputs
    - Path: ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/tasks.md
      Note: Simplified capability-based task list that this DSL design feeds into
ExternalSources: []
Summary: "Design for a YAML DSL that configures the mock Saleae gRPC server behavior (fixtures, timing, exports, error injection) so tests can cover many scenarios without code changes."
LastUpdated: 2025-12-27T12:51:17.514825723-05:00
WhatFor: "Enable configuring mock server behavior via YAML to test salad CLI verbs across happy paths and failure modes deterministically"
WhenToUse: "When writing tests or local repros that need the mock server to behave like specific Saleae/Logic2 situations (devices present/missing, capture states, export behaviors, injected errors)"
---

# Mock Server YAML DSL: Configurable Behavior Scenarios

## Executive Summary

We want the mock Saleae gRPC server to be **configured**, not hard-coded: tests (and humans) should be able to spin up a mock server with specific devices, capture states, timing behavior, export side effects, and error injections using a **YAML DSL**.

This document proposes a DSL that is:
- **Deterministic**: stable IDs, predictable outcomes.
- **Scenario-driven**: a single YAML file describes a cohesive “world”.
- **Composable**: defaults + per-method overrides.
- **Minimal but extensible**: enough to support current salad CLI verbs, with room for future RPCs.

## Non-goals (for now)

- Simulating real sample data content (we can write placeholder export outputs).
- Full fidelity timing/blocking semantics identical to Logic2 (we’ll model enough to test salad).
- Implementing analyzer/HLA RPC behavior beyond what tests currently need.

## Key idea: configuration controls *server behavior*, not client behavior

The salad CLI is a black box from the mock’s perspective. The YAML config needs to express:
- **Initial fixtures**: appinfo, devices, existing captures (optional).
- **State transitions**: what happens when `LoadCapture`, `StopCapture`, `WaitCapture`, etc. are called.
- **Side effects**: writing placeholder files for export/save.
- **Failures**: inject gRPC status errors deterministically (by method, by call count, by request match).

## Where the YAML is used

Two main uses:
- **Tests**: `StartMockServerFromYAML(t, "scenario.yaml")`.
- **Manual local testing**: run a `salad-mock` binary with `--config scenario.yaml`.

## DSL structure (high-level)

At a high level:

```yaml
version: 1
scenario: happy-path-minimal

defaults:
  grpc:
    status_on_unknown_capture_id: INVALID_ARGUMENT
  ids:
    capture_id_start: 1
    deterministic: true

fixtures:
  appinfo:
    application_version: "2.3.56-mock"
    api_version: { major: 1, minor: 0, patch: 0 }
    launch_pid: 4242
  devices:
    - device_id: "DEV1"
      device_type: DEVICE_TYPE_LOGIC_PRO_8
      is_simulation: false

behavior:
  GetDevices:
    filter_simulation_devices: true
  LoadCapture:
    on_call:
      create_capture:
        status: completed
  ExportRawDataCsv:
    side_effect:
      write_placeholders:
        digital_csv: true
        analog_csv: true

faults:
  - when:
      method: SaveCapture
      nth_call: 1
    respond:
      status: INVALID_ARGUMENT
      message: "SaveCapture: capture not found"
```

## Core schema (proposed)

### Top-level fields

- `version` (int, required): DSL version. Start at `1`.
- `scenario` (string, optional): human-readable scenario name.
- `defaults` (object, optional): global defaults for ids/timing/errors.
- `fixtures` (object, optional): initial state (appinfo, devices, captures).
- `behavior` (object, optional): per-RPC behavior knobs.
- `faults` (list, optional): error injection rules (ordered; first match wins).

### `fixtures.appinfo`

Represents `GetAppInfoReply.app_info`.

```yaml
fixtures:
  appinfo:
    application_version: "2.3.56-mock"
    api_version: { major: 1, minor: 0, patch: 0 }
    launch_pid: 4242
```

### `fixtures.devices`

List returned by `GetDevices` (subject to filter semantics).

```yaml
fixtures:
  devices:
    - device_id: "DEV1"
      device_type: DEVICE_TYPE_LOGIC_PRO_8
      is_simulation: false
    - device_id: "SIM1"
      device_type: DEVICE_TYPE_LOGIC_8
      is_simulation: true
```

### `fixtures.captures` (optional)

Pre-seed capture IDs (useful for testing `save/stop/wait/close/export` without calling `LoadCapture`).

```yaml
fixtures:
  captures:
    - capture_id: 10
      status: completed      # running|stopped|completed|closed
      origin: loaded         # loaded|started (for future)
      started_at: "2025-12-27T00:00:00Z"
      mode:
        kind: timed
        duration_seconds: 1.0
```

## Behavior knobs per RPC (current verbs)

### `GetAppInfo`

Mostly fixture-driven. Behavior knob: optionally override per-call to simulate changing versions.

```yaml
behavior:
  GetAppInfo:
    reply_override:
      application_version: "2.3.57-mock"
```

### `GetDevices`

We need to model how `include_simulation_devices` affects output.

```yaml
behavior:
  GetDevices:
    # If true, and request.include_simulation_devices == false,
    # filter fixtures.devices where is_simulation==true.
    filter_simulation_devices: true
```

### `LoadCapture`

For current CLI, `LoadCapture` just needs to create a capture and return an ID. The DSL should control:
- capture status after load
- optional “file existence” check behavior
- deterministic vs generated capture IDs

```yaml
behavior:
  LoadCapture:
    validate:
      require_non_empty_filepath: true
      require_file_exists: false
    on_call:
      create_capture:
        status: completed
        mode: { kind: timed, duration_seconds: 0.0 }
```

### `SaveCapture`

For CLI tests, we often want either:
- success no-op
- actually write a placeholder `.sal` file (0 bytes or small marker)
- fail if capture doesn’t exist

```yaml
behavior:
  SaveCapture:
    validate:
      require_capture_exists: true
    side_effect:
      write_placeholder_file: true
      placeholder_bytes: "SALAD_MOCK_SAL_V1\n"
```

### `StopCapture`

Controls how state transitions happen.

```yaml
behavior:
  StopCapture:
    validate:
      require_capture_exists: true
    transition:
      from: running
      to: stopped
```

### `WaitCapture`

This is the trickiest. We want configurability around:
- whether `WaitCapture` blocks vs returns immediately
- what “still running” means
- how long it should “pretend” a timed capture takes

Proposed knob: **policy**.

```yaml
defaults:
  timing:
    wait_capture_policy: immediate   # immediate|block_until_done|error_if_running
    max_block_ms: 50

behavior:
  WaitCapture:
    validate:
      require_capture_exists: true
      error_on_manual_mode: true
    completion:
      # for timed captures: complete after duration_seconds (relative to started_at)
      timed_captures_complete_after_duration: true
```

Recommended for tests: `immediate` or `error_if_running`. Avoid long sleeps.

### `CloseCapture`

Decide whether close deletes the capture or marks closed (affects double-close behavior).

```yaml
behavior:
  CloseCapture:
    mode: delete          # delete|mark_closed
```

### `ExportRawDataCsv` / `ExportRawDataBinary`

Side effects are key: we want to optionally write placeholder outputs so tests can assert file presence.

```yaml
behavior:
  ExportRawDataCsv:
    validate:
      require_capture_exists: true
    side_effect:
      write_placeholders:
        digital_csv: true
        analog_csv: true
        filenames:
          digital: "digital.csv"
          analog: "analog.csv"
      include_requested_channels_in_file: true

  ExportRawDataBinary:
    validate:
      require_capture_exists: true
    side_effect:
      write_placeholders:
        digital_bin: true
        analog_bin: true
        filenames:
          digital: "digital.bin"
          analog: "analog.bin"
```

## Fault injection (error rules)

We need deterministic “make this method fail” controls without editing Go code. Proposed model:
- `faults` is an ordered list
- each fault has `when` + `respond`
- first match wins

### `when` fields (matching)

- `method` (string, required): e.g. `GetDevices`, `SaveCapture`
- `nth_call` (int, optional): apply only on Nth call for that method
- `match` (object, optional): match on request fields (subset)

Example: fail only for capture_id 123:

```yaml
faults:
  - when:
      method: StopCapture
      match:
        capture_id: 123
    respond:
      status: INVALID_ARGUMENT
      message: "capture not found"
```

Example: fail first call, succeed afterwards:

```yaml
faults:
  - when:
      method: GetDevices
      nth_call: 1
    respond:
      status: UNAVAILABLE
      message: "transient mock failure"
```

### `respond` fields

- `status` (string, required): gRPC code name (e.g. `INVALID_ARGUMENT`)
- `message` (string, required)
- `details` (optional, future): richer error details; likely not needed now

## Scenario library (variety of configs)

Below are intentionally varied scenarios we should support.

### Scenario A: happy path (no simulation devices)

```yaml
version: 1
scenario: happy-path

fixtures:
  appinfo:
    application_version: "2.3.56-mock"
    api_version: { major: 1, minor: 0, patch: 0 }
    launch_pid: 1111
  devices:
    - device_id: "DEV1"
      device_type: DEVICE_TYPE_LOGIC_PRO_8
      is_simulation: false

defaults:
  ids: { deterministic: true, capture_id_start: 1 }
  timing: { wait_capture_policy: immediate }

behavior:
  LoadCapture:
    on_call: { create_capture: { status: completed } }
  SaveCapture:
    side_effect: { write_placeholder_file: true, placeholder_bytes: "SALAD_MOCK_SAL_V1\n" }
  ExportRawDataCsv:
    side_effect: { write_placeholders: { digital_csv: true, analog_csv: true } }
  ExportRawDataBinary:
    side_effect: { write_placeholders: { digital_bin: true, analog_bin: true } }
```

### Scenario B: devices include simulation; filter behavior exercised

```yaml
version: 1
scenario: devices-filtering

fixtures:
  devices:
    - { device_id: "DEV1", device_type: DEVICE_TYPE_LOGIC_PRO_16, is_simulation: false }
    - { device_id: "SIM1", device_type: DEVICE_TYPE_LOGIC_8, is_simulation: true }

behavior:
  GetDevices:
    filter_simulation_devices: true
```

### Scenario C: no devices connected (GetDevices empty)

```yaml
version: 1
scenario: no-devices
fixtures:
  devices: []
```

### Scenario D: LoadCapture fails for specific filepath

```yaml
version: 1
scenario: loadcapture-filepath-fails

faults:
  - when:
      method: LoadCapture
      match:
        filepath: "/nope/file.sal"
    respond:
      status: INVALID_ARGUMENT
      message: "LoadCapture failed: file not found"
```

### Scenario E: SaveCapture fails first time (transient)

```yaml
version: 1
scenario: savecapture-transient-fail

fixtures:
  captures:
    - { capture_id: 1, status: completed, origin: loaded }

faults:
  - when: { method: SaveCapture, nth_call: 1 }
    respond: { status: UNAVAILABLE, message: "temporary failure" }
```

### Scenario F: WaitCapture behavior matrix (still running vs completed)

This scenario models a running timed capture and forces `WaitCapture` to error until we advance time (or until duration elapses, depending on policy).

```yaml
version: 1
scenario: waitcapture-running

defaults:
  timing:
    wait_capture_policy: error_if_running

fixtures:
  captures:
    - capture_id: 1
      status: running
      origin: started
      started_at: "2025-12-27T00:00:00Z"
      mode: { kind: timed, duration_seconds: 10.0 }
```

Variation (block a tiny bit):

```yaml
defaults:
  timing:
    wait_capture_policy: block_until_done
    max_block_ms: 25
```

### Scenario G: Export is no-op (only validates capture exists)

```yaml
version: 1
scenario: export-noop

fixtures:
  captures:
    - { capture_id: 1, status: completed, origin: loaded }

behavior:
  ExportRawDataCsv:
    side_effect: { write_placeholders: { digital_csv: false, analog_csv: false } }
  ExportRawDataBinary:
    side_effect: { write_placeholders: { digital_bin: false, analog_bin: false } }
```

### Scenario H: Export writes deterministic filenames and content markers

```yaml
version: 1
scenario: export-files-assertable

fixtures:
  captures:
    - { capture_id: 1, status: completed, origin: loaded }

behavior:
  ExportRawDataCsv:
    side_effect:
      write_placeholders:
        digital_csv: true
        analog_csv: true
        filenames: { digital: "d.csv", analog: "a.csv" }
      include_requested_channels_in_file: true
```

### Scenario I: Unknown capture IDs return configurable status code

Some tests may want `NOT_FOUND`; others want `INVALID_ARGUMENT`. Make it configurable:

```yaml
defaults:
  grpc:
    status_on_unknown_capture_id: NOT_FOUND
```

## Recommended implementation mapping (how YAML drives Go)

This is intentionally simple:

- Parse YAML into a `Config` struct.
- Convert string status codes to `codes.Code`.
- Maintain:
  - `fixtures` as initial state in server struct
  - `behavior` as policy knobs (enums/bools/strings)
  - `faults` as a list of matchers evaluated at start of every RPC
  - `call_counters[method]++` for `nth_call`

RPC handler pseudo-flow:

1. `if fault := matchFault(method, req, counters); fault != nil { return status.Error(fault.code, fault.msg) }`
2. Validate using `behavior.<Method>.validate` + defaults
3. Apply state transitions (if any)
4. Perform side effects (optional file writes)
5. Return reply

## Open questions (intentionally few)

- Do we want request matching to support nested fields (e.g. channels)? Probably not initially.
- Do we want “time travel” hooks for `WaitCapture`? For now: `wait_capture_policy` + seeded `started_at`.
- Do we want multiple scenarios per file? Maybe later (a runner could load “scenario sets”); start with one scenario per YAML file.