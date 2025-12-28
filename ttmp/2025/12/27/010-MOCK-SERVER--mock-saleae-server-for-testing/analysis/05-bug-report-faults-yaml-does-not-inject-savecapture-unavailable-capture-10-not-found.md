---
Title: 'Bug report: faults.yaml does not inject SaveCapture UNAVAILABLE (capture 10 not found)'
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
    - Path: configs/mock/faults.yaml
      Note: Scenario intended to seed capture_id=10 and inject UNAVAILABLE on first SaveCapture call
    - Path: ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/scripts/03-smoke-faults.sh
      Note: Repro script currently failing (likely missing diagnostic output / possibly picking wrong config)
    - Path: cmd/salad-mock/cmd/root.go
      Note: Loads YAML config and starts mock server with compiled plan
    - Path: internal/mock/saleae/config.go
      Note: YAML decoding (KnownFields) for Config / Fixtures / Faults
    - Path: internal/mock/saleae/plan.go
      Note: Compile(cfg) -> Plan, including compileFixtures and compileFaults
    - Path: internal/mock/saleae/exec.go
      Note: Server.exec and maybeFault (fault injection happens before RPC handler logic)
    - Path: internal/mock/saleae/server.go
      Note: SaveCapture handler calls captureFor and side-effects; symptom arises here if fault not applied or fixture missing
ExternalSources: []
Summary: "When running the faults scenario, SaveCapture does not return the configured UNAVAILABLE on the first call; instead the server returns InvalidArgument \"capture 10 not found\", suggesting the scenario config is not applied or fixtures/fault rules are not being loaded."
LastUpdated: 2025-12-27T15:52:13.911298572-05:00
WhatFor: "Track and debug the mismatch between the faults YAML scenario and observed SaveCapture behavior"
WhenToUse: "When the faults scenario doesn’t behave as expected, or when debugging YAML scenario loading / fault injection"
---

# Bug report: `configs/mock/faults.yaml` does not inject SaveCapture UNAVAILABLE (capture 10 not found)

## Summary

Running the “faults” scenario is expected to:
- pre-seed capture `capture_id=10`, and
- inject a transient `UNAVAILABLE` error on the **first** `SaveCapture` call (`nth_call: 1`),
then succeed on the second call.

Observed behavior instead:
- `SaveCapture` returns `InvalidArgument: capture 10 not found` (both first and second call),
which implies **either**:
- the mock server did not load the intended scenario config (wrong config file), **or**
- `fixtures.captures` did not end up in runtime state, **or**
- fault rules were not compiled/applied (or method call counting differs from expectations).

## Environment

- **Module**: `salad/` (note: repo root `go.work` does not include `salad`; use `GOWORK=off` for tests)
- **Go**: `go1.25.3` (from local output)

## Expected behavior (from YAML)

From `configs/mock/faults.yaml`:
- `fixtures.captures` contains capture `10` with `status: completed`, `origin: loaded`
- `faults[0]`:
  - `when.method: SaveCapture`
  - `when.nth_call: 1`
  - `respond.status: UNAVAILABLE`
  - `respond.message: "temporary mock failure"`

Therefore, the first SaveCapture should return: `rpc error: code = Unavailable desc = temporary mock failure`

## Actual behavior

When running the ticket smoke script:

- Script: `ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/scripts/03-smoke-faults.sh`
- Observed output (abridged):
  - `SaveCapture RPC: rpc error: code = InvalidArgument desc = capture 10 not found`
  - Second call also fails with the same error

## Triage notes (via plz-confirm, 2025-12-27)

We asked the user for confirmation and context via `plz-confirm form` and got the following key signals:

- **Intended config**: `configs/mock/faults.yaml`
- **How it was run**: “i did run it.”
- **Theory**: “maybe wrong yaml loaded?”
- **OK to add debug logs**: yes
- **Teaching checks (freeform answers)**:
  - fault injection should happen “1st” (interpreted as “before handler”, i.e. `exec/maybeFault`)
  - `nth_call` should count “1” (interpreted as “per-method since server start”)

Missing but still needed to confirm the top hypothesis (“wrong config loaded”):
- output of `env | egrep '^(CFG|HOST|PORT)=' || true` for the script run
- `mock.log` from the failing faults run

## How to reproduce (current)

1. Run the mock server with the faults scenario, then attempt `SaveCapture` twice.

The script intended for this is:
- `.../scripts/03-smoke-faults.sh`

Important note: the script currently does **not** print the config path being used, and it exits before dumping mock logs if the second call fails. That makes it hard to confirm whether the correct config loaded.

## Most likely “where the bug lives” (files + symbols)

### 1) Wrong config file / scenario not being loaded (very plausible)

If the mock server starts with `happy-path.yaml` instead of `faults.yaml`, then capture 10 won’t exist and no SaveCapture fault will be injected.

Relevant code:
- `cmd/salad-mock/cmd/root.go`: loads config path and calls `mock.LoadConfig(configPath)` then `mock.Compile(cfg)`
- `internal/mock/saleae/config.go`: YAML decode (KnownFields)

Isolation steps:
- Ensure the process truly loaded `configs/mock/faults.yaml` (print config path + scenario at startup).

### 2) Fixtures not being compiled into the plan or not being loaded into runtime state

The seeded capture should go:

`faults.yaml -> Config.Fixtures.Captures -> Compile(cfg)->compileFixtures -> Plan.Fixtures.Captures -> newState(plan, clock) -> State.Captures[10]`

Relevant code:
- `internal/mock/saleae/plan.go`:
  - `Compile(cfg Config) (*Plan, error)`
  - `compileFixtures(cfg FixturesConfig) (FixturesPlan, error)`
  - `parseCaptureStatus`, `parseCaptureOrigin` (should accept “completed”/“loaded”)
- `internal/mock/saleae/exec.go`:
  - `newState(plan, clock) State` populates `State.Captures` from `plan.Fixtures.Captures`

Isolation steps:
- Add temporary logging in `NewServer(...)` or `newState(...)` of the capture IDs loaded into state.

### 3) Fault rules not being compiled or not being applied

Fault injection is designed to happen before RPC method logic:

- `Server.exec(...)` increments `s.calls[method]`, then calls `maybeFault(method, req, callN)`
- `maybeFault` iterates `plan.Faults` and returns `statusError(code,message)` if matched

Relevant code:
- `internal/mock/saleae/plan.go`:
  - `compileFaults(cfg []FaultRuleConfig) ([]FaultRule, error)`
  - `parseMethod(method string) (Method, error)` (exact match against `AllMethods`)
- `internal/mock/saleae/exec.go`:
  - `func (s *Server) maybeFault(method Method, req any, callN int) error`
- `internal/mock/saleae/server.go`:
  - `SaveCapture` calls `s.exec(... MethodSaveCapture ...)` (so fault matching should see method == `SaveCapture`)

Isolation steps:
- Log `len(plan.Faults)` at startup (or log the compiled fault list).
- Log in `maybeFault`: `method`, `callN`, and whether a rule matched.

## A “minimum isolation” repro that removes CLI complexity

If the CLI is suspected to be the issue, isolate with:

- Start `salad-mock` with `--log-level debug` (after adding a couple logs below)
- Use a minimal Go snippet or `grpcurl` to call `SaveCapture` twice against the mock

This narrows the problem to “server plan/state/faults” vs “CLI invocation / wrong port”.

## Logging that would make this easy (suggested additions)

### In `cmd/salad-mock/cmd/root.go` (startup)

Log at `INFO`:
- config path
- scenario name (`cfg.Scenario`)
- plan counts:
  - `len(plan.Fixtures.Devices)`
  - `len(plan.Fixtures.Captures)`
  - `len(plan.Faults)`

### In `internal/mock/saleae/exec.go` (server internals)

Log at `DEBUG`:
- in `newState`: list of capture IDs inserted into `State.Captures`
- in `maybeFault`: method + callN + matched rule (if any)

### In `internal/mock/saleae/server.go` (SaveCapture)

Log at `DEBUG`:
- capture_id requested
- whether capture exists before side-effects

These logs would directly answer: “Did the right config load? Did fixtures load? Did faults compile? Did fault match fire?”

## Notes / hypotheses (ranked)

1. **Highest likelihood**: the script or environment is causing the mock server to run with the wrong config (or the script doesn’t prove what config was used).
2. If config is correct: either fixture captures aren’t getting into runtime state, or faults aren’t compiled/applied (despite the architecture suggesting they should).

## New finding: port collisions can mimic “wrong scenario”

We later observed that the system can have *multiple* `salad-mock` processes running simultaneously (e.g., one started by an earlier smoke script). If the smoke script tries to start a new mock server on a port that is already occupied, one of two confusing outcomes can happen:

- The new server fails to bind (`bind: address already in use`), but the CLI still succeeds because it’s talking to the *old* server on that port (with a different config/scenario).
- This makes it look like “faults.yaml doesn’t work”, when the reality is “we never actually ran faults.yaml on that port”.

Mitigations added in the ticket workspace:
- Use tmux scripts to explicitly manage server lifecycle:
  - `scripts/10-tmux-mock-start.sh`, `scripts/11-tmux-mock-stop.sh`, `scripts/12-tmux-mock-restart.sh`
- Write server output to persistent log files under `scripts/logs/` (ignored in git).
- Use `scripts/09-kill-mock-on-port.sh` to kill stuck listeners by port.
- Default the faults smoke script to use a separate port (`PORT=10432`) and always dump its `mock.log` on exit.