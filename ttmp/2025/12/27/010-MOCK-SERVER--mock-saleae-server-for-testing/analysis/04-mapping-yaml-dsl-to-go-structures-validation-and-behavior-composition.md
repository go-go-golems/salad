---
Title: 'Mapping YAML DSL to Go: Structures, Validation, and Behavior Composition'
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
    - Path: ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/analysis/03-mock-server-yaml-dsl-configurable-behavior-scenarios.md
      Note: The YAML DSL being mapped into Go structs and runtime behavior
    - Path: proto/saleae/grpc/saleae.proto
      Note: RPC contract that handlers must satisfy
    - Path: gen/saleae/automation/saleae_grpc.pb.go
      Note: ManagerServer interface that mock implements
    - Path: internal/saleae/client.go
      Note: Client call patterns the mock must support
ExternalSources: []
Summary: "General method for translating YAML DSL configs into Go data structures and composable mock-server behavior (defaults, validation, fault injection, state transitions), with pseudocode patterns."
LastUpdated: 2025-12-27T12:55:19.253840035-05:00
WhatFor: "Guide implementation of a clean, composable, extendable mock server driven by YAML scenario configs"
WhenToUse: "When implementing or extending the mock server YAML DSL compiler, runtime plan, or gRPC handler pipeline"
---

# Mapping YAML DSL to Go: Structures, Validation, and Behavior Composition

## Executive Summary

This document describes a **general method** for mapping YAML DSL scenario files into:
- Go **config structs** (pure data),
- a compiled **runtime plan** (defaults applied, enums normalized, matchers compiled),
- and composable **server behavior** (fault injection, validation, transitions, side effects).

The goal is a mock server that stays **clean and extendable**:
- **Composable**: behavior = stackable layers (faults → validate → mutate state → side effects → reply).
- **Extendable**: adding a new RPC or knob doesn’t require editing every handler.
- **Deterministic**: IDs, faults, and timing are controlled and repeatable.

## General Method (recommended architecture)

### Step 0: Separate “Config” from “Runtime”

Have two representations:

1) **Config (YAML-facing)**: mirrors YAML keys, friendly types (strings for enums, optional fields).
2) **Runtime (Go-facing)**: normalized types (enums as `codes.Code`, durations as `time.Duration`), defaults applied, matchers compiled.

This prevents RPC handlers from having to interpret YAML-ish types constantly.

### Step 1: Define a versioned top-level schema

Top-level Go config should look like:
- `Version int`
- `Scenario string`
- `Defaults DefaultsConfig`
- `Fixtures FixturesConfig`
- `Behavior BehaviorConfig` (per-RPC)
- `Faults []FaultRuleConfig`

Prefer pointers for optional nested blocks (`*FooConfig`) to distinguish “unset” vs “set empty”.

### Step 2: Unmarshal YAML into Config structs (no logic)

Use `yaml.v3` to decode into `Config`.

Rules:
- Avoid doing validation inside custom `UnmarshalYAML` unless absolutely needed.
- Keep decoding strict-ish: unknown fields should be rejected (helps catch typos).

### Step 3: Validate + normalize + apply defaults into a Runtime Plan

Create a compiler:

```go
func Compile(cfg Config) (*Plan, error)
```

`Compile` does:
- schema validation (required fields, supported version)
- apply defaults (global → per-RPC)
- normalize enums/strings (`"INVALID_ARGUMENT"` → `codes.InvalidArgument`)
- compile matchers (fault rules, request match conditions)

The returned `Plan` should be immutable after compilation.

### Step 4: Build a server from a Plan

```go
type Server struct {
  pb.UnimplementedManagerServer
  plan *Plan

  mu    sync.Mutex
  state State
  calls map[Method]int
  clock Clock // deterministic time
}
```

Instantiate state from `plan.Fixtures`.

### Step 5: Implement each RPC via a shared “pipeline”

Each handler should be a thin wrapper that calls a generic executor:

```go
func (s *Server) GetDevices(ctx context.Context, req *pb.GetDevicesRequest) (*pb.GetDevicesReply, error) {
  out, err := s.exec(ctx, MethodGetDevices, req, func(rt *RuntimeCtx) (any, error) {
    // method-specific: compute reply from rt.State + rt.Plan
    return &pb.GetDevicesReply{Devices: rt.State.DevicesFiltered(req)}, nil
  })
  if err != nil { return nil, err }
  return out.(*pb.GetDevicesReply), nil
}
```

`exec` centralizes:
- call counters
- fault injection
- shared validations / policy defaults
- structured logging (optional)

## Go Data Structures (suggested patterns)

### Config layer (YAML-facing)

Keep config “boring” and easy to evolve:

```go
type Config struct {
  Version  int               `yaml:"version"`
  Scenario string            `yaml:"scenario,omitempty"`
  Defaults DefaultsConfig    `yaml:"defaults,omitempty"`
  Fixtures FixturesConfig    `yaml:"fixtures,omitempty"`
  Behavior BehaviorConfig    `yaml:"behavior,omitempty"`
  Faults   []FaultRuleConfig `yaml:"faults,omitempty"`
}
```

Use string forms for enums in YAML:
- `device_type: DEVICE_TYPE_LOGIC_PRO_8`
- `status: INVALID_ARGUMENT`

Use pointers for optional knobs:

```go
type GetDevicesBehaviorConfig struct {
  FilterSimulationDevices *bool `yaml:"filter_simulation_devices,omitempty"`
}
```

### Runtime plan (compiled)

```go
type Plan struct {
  Version  int
  Scenario string

  Defaults Defaults
  Fixtures Fixtures
  Behavior Behavior
  Faults   []FaultRule // compiled matchers
}
```

Normalize:
- status codes: `codes.Code`
- wait policy: `WaitPolicy` enum
- durations: `time.Duration`

## Defaults layering (clean composability)

Use a consistent precedence rule:

1. Global defaults (`defaults.*`)
2. Per-method behavior (`behavior.<Method>.*`)
3. Request-driven variations (e.g. `include_simulation_devices`)

In `Compile`, produce “ready-to-use” runtime behavior for each method.

Helper pattern:

```go
func pickBool(v *bool, def bool) bool {
  if v == nil { return def }
  return *v
}
```

## Fault injection (general method)

### Goal

One uniform mechanism to inject errors in any RPC without hardcoding branches per handler.

### Data model

Config:
- `method`
- optional `nth_call`
- optional `match` (subset of request fields)
- response `status`, `message`

Runtime:

```go
type FaultRule struct {
  Method   Method
  NthCall  *int
  Match    func(req any) bool
  Code     codes.Code
  Message  string
}
```

### Matching algorithm

In `exec`:

```go
s.calls[m]++
callN := s.calls[m]

for _, rule := range s.plan.Faults {
  if rule.Method != m { continue }
  if rule.NthCall != nil && *rule.NthCall != callN { continue }
  if rule.Match != nil && !rule.Match(req) { continue }
  return nil, status.Error(rule.Code, rule.Message)
}
```

**First match wins** is easiest to reason about and keeps YAML deterministic.

### Request matching strategy (start small)

Avoid a generic “match any nested protobuf field” system at first.

Instead:
- support a **small set** of match keys per method (e.g. `capture_id`, `filepath`)
- compile strongly-typed matchers per method

Example pattern (pseudocode):

```go
func compileLoadCaptureMatch(m MatchConfig) func(any) bool {
  if m.Filepath == nil { return nil }
  want := *m.Filepath
  return func(req any) bool {
    r := req.(*pb.LoadCaptureRequest)
    return r.GetFilepath() == want
  }
}
```

This keeps the system maintainable and explicit.

## Deterministic time + `WaitCapture`

### Core idea: clock injection

```go
type Clock interface { Now() time.Time }
```

Use:
- `RealClock` for manual runs
- `FakeClock` for tests (optional)

### Policy knob (from YAML)

Model `WaitCapture` with a simple policy:
- `immediate`
- `error_if_running`
- `block_until_done` (bounded by `max_block_ms`)

General pseudocode:

```go
switch policy {
case Immediate:
  if capture.Completed() { return ok }
  return status.Error(codes.DeadlineExceeded, "still running")
case ErrorIfRunning:
  if capture.Running() { return status.Error(codes.DeadlineExceeded, "still running") }
  return ok
case BlockUntilDone:
  until := clock.Now().Add(maxBlock)
  for clock.Now().Before(until) {
    if capture.Completed() { return ok }
    sleep(2 * time.Millisecond)
  }
  return status.Error(codes.DeadlineExceeded, "still running")
}
```

Recommendation: default to `immediate` or `error_if_running` in CI tests.

## State transitions (capture lifecycle)

Keep the state model small and explicit:
- captures map: `capture_id → CaptureState`
- capture status: `running|stopped|completed|closed`

Then in each RPC:
- validate existence (policy-configurable status code)
- apply transition (idempotently)

Example transition rule:

```go
// StopCapture: running → stopped; stopped/completed are no-ops
if c.Status == Running { c.Status = Stopped }
```

## Side effects (save/export) as pluggable strategy

Don’t bake filesystem writes into handlers. Use a small interface:

```go
type SideEffects interface {
  SaveCapture(filepath string, captureID uint64) error
  ExportRawCSV(dir string, req *pb.ExportRawDataCsvRequest) error
  ExportRawBinary(dir string, req *pb.ExportRawDataBinaryRequest) error
}
```

Then provide implementations:
- `NoopSideEffects`
- `PlaceholderFilesSideEffects` (writes deterministic marker files)

YAML selects which strategy to use per method (or globally).

## The “exec” template (pseudocode)

```go
func (s *Server) exec(ctx context.Context, m Method, req any, fn func(rt *RuntimeCtx) (any, error)) (any, error) {
  s.mu.Lock()
  defer s.mu.Unlock()

  // 1) call counters
  s.calls[m]++

  // 2) fault injection
  if err := s.maybeFault(m, req, s.calls[m]); err != nil {
    return nil, err
  }

  // 3) runtime context
  rt := &RuntimeCtx{
    Ctx:   ctx,
    Plan:  s.plan,
    State: &s.state,
    Clock: s.clock,
    CallN: s.calls[m],
  }

  // 4) method-specific work
  return fn(rt)
}
```

The payoff: every RPC handler looks the same structurally, and most “behavior” is data-driven via the plan.

## Extension points (how to keep it clean as DSL grows)

- Add a new RPC:
  - add method enum
  - add (optional) behavior config struct
  - add handler using `exec`
  - optionally add request matchers for faults

- Add a new knob:
  - add it to config struct
  - validate + apply defaults in `Compile`
  - consume it from `Plan` in the relevant layer

This prevents YAML concerns from leaking across the codebase.