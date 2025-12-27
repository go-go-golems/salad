# Tasks

The goal here is **small number of tasks**, each representing a meaningful capability. Subtasks are intentionally avoided; details live in analysis docs and code.

## Core deliverable: configurable mock server (YAML-driven, compiled plan)

- [x] Implement mock `Manager` gRPC server as a **pipeline** over a compiled runtime plan:
  - YAML `Config` → `Compile(Config) -> Plan` → server created from `Plan`
- [x] Add YAML config loader (strict decode) + `Compile` step:
  - validate schema + version
  - apply defaults layering (global → per-RPC)
  - normalize enums (e.g. `"INVALID_ARGUMENT"` → `codes.InvalidArgument`)
  - compile fault matchers (first-match-wins, per-method call counters)
- [x] Implement a shared `exec` wrapper used by all RPC handlers:
  - increments call counters
  - applies fault injection rules
  - constructs runtime context (plan + state + clock + side-effects)
  - runs method-specific handler closure
- [x] Add deterministic `Clock` injection to avoid flaky `WaitCapture` tests
- [x] Add pluggable `SideEffects` strategy:
  - noop
  - placeholder-files (exports/save write deterministic marker outputs)
- [x] Add test helper that starts the mock server from a YAML config (random port) and returns cleanup

## Support current CLI verbs (minimum set)

- [x] Implement RPCs required by existing CLI:
  - `GetAppInfo`, `GetDevices`
  - `LoadCapture`, `SaveCapture`, `StopCapture`, `WaitCapture`, `CloseCapture`
  - `ExportRawDataCsv`, `ExportRawDataBinary`
- [x] Implement `StartCapture` RPC for 002-CAPTURE-START testing:
  - [x] Add `MethodStartCapture` constant to `exec.go`
  - [x] Add `StartCaptureBehaviorConfig` to YAML config schema (`config.go`)
  - [x] Add `StartCapturePlan` to compiled plan structure (`plan.go`)
  - [x] Implement `StartCapture` handler in `server.go`:
    - [x] Device validation (empty device_id → first physical device, error if none)
    - [x] Device existence check (error if device_id not found)
    - [x] Capture state creation with mode from `CaptureConfiguration` (manual/timed/trigger)
    - [x] Capture ID generation (use `NextCaptureID`)
    - [x] Return `StartCaptureReply` with `CaptureInfo`
  - [x] Add YAML config examples for StartCapture scenarios (`configs/mock/start-capture.yaml`, updated `happy-path.yaml`)
  - [ ] Add tests for StartCapture (happy path + error cases: missing device, invalid device_id)

## Behavior knobs required by the YAML DSL

- [x] Deterministic IDs + seeding (capture IDs, optional analyzer IDs later)
- [x] Configurable device inventory + simulation filtering behavior
- [x] Capture lifecycle model (states + `WaitCapture` policies via clock/policy knob)
- [x] Configurable export behavior:
  - no-op success
  - write placeholder files (for tests that assert filesystem outputs)
- [x] Error injection model:
  - always error for a method
  - error Nth call for a method
  - error when request matches (e.g., capture_id not found)

## Thin test suite (confidence, not exhaustive)

- [ ] Table-driven tests that run the salad CLI against the mock for:
  - happy path (appinfo/devices/capture load+save+close/export)
  - missing capture id error behavior
  - `WaitCapture` still-running vs completed cases
  - export placeholder files present when configured

## Documentation + examples

- [x] Add `configs/mock/` example YAML scenarios (happy path + failure modes) used by tests and humans
- [x] Document the DSL and the supported behaviors (keep in sync with tests)
- [x] Create docs in `salad/pkg/doc/`:
  - **Developer guide**: “How to extend the mock server + DSL” (follow style from `glazed/pkg/doc/topics/how-to-write-good-documentation-pages.md`)
  - **User guide**: “How to run and use the mock server” (scenario YAML usage, how to point `salad` at it, common workflows)
