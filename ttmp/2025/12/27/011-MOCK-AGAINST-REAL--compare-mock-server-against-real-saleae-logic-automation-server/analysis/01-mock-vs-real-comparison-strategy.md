---
Title: 'Mock vs Real: comparison strategy'
Ticket: 011-MOCK-AGAINST-REAL
Status: active
Topics:
    - saleae
    - mock-server
    - testing
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2025-12-27T16:51:53.603353341-05:00
WhatFor: ""
WhenToUse: ""
---

# Mock vs Real: comparison strategy

## Goal

Validate that `salad-mock` matches the real Saleae Logic 2 Automation (gRPC) server closely enough that tests against the mock are meaningful, and identify (and document) any intentional divergences.

## Key constraints (practical)

### The port collision problem

Both the mock and the real server are gRPC servers listening on a TCP `host:port`. **Two servers can’t bind the same port on the same host.**

So any comparison workflow must support switching endpoints explicitly, or running both simultaneously on different ports.

### The good news: switching already exists

The repo already uses an explicit address selection model:

- `salad` (client CLI) takes `--host/--port` and defaults to **`127.0.0.1:10430`**.
- `salad-mock` (mock server CLI) takes `--host/--port` and defaults to **`127.0.0.1:10431`**.
- `salad/internal/saleae/client.go` dials whatever `host:port` you pass via `saleae.Config{Host, Port}`.

This means we can compare real vs mock **without code changes** by running the same command suite twice, changing only the endpoint.

## What does “compare mock vs real” mean?

There are multiple valid comparison layers; each answers a slightly different question.

### Layer 1: CLI black-box comparison (fastest start)

Run the same `salad` commands against:

- **real** server (`--port 10430` unless you changed it)
- **mock** server (`--port 10431` or another chosen port)

Compare:
- exit status
- stdout/stderr (or normalized subsets)
- error codes/messages surfaced by `salad`

**Pros:** zero code changes; uses your actual client paths; catches “integration feels wrong.”  
**Cons:** differences can be caused by CLI formatting changes, and the CLI may hide server-level details (gRPC status code vs wrapped error string).

### Layer 2: RPC-level contract comparison (most precise)

Directly compare the gRPC API surface:
- method semantics (what happens for valid/invalid requests)
- gRPC status codes (`InvalidArgument`, `NotFound`, `Unavailable`, …)
- required/optional fields in responses

How:
- write a small Go harness that calls `pb.ManagerClient` with known requests; or
- use `grpcurl` with prepared JSON payloads.

**Pros:** precise; isolates “server contract” from “CLI behavior.”  
**Cons:** requires authoring requests; some requests require hardware realities (real devices/captures).

### Layer 3: Record/replay transcript (best long-term regression tool)

Run a suite against the **real** server while recording a transcript:
- method name
- request proto (with redactions/normalization)
- response proto or gRPC status (code/message)

Then replay the same requests against the **mock** and diff.

Implementation options:
- a gRPC client interceptor on the `salad` client connection
- or a dedicated harness that already owns the client and can log everything

**Pros:** stable regression tool; finds drift early.  
**Cons:** needs careful normalization for nondeterministic values.

## Most likely sources of real-vs-mock drift (call these out explicitly)

### 1) Device reality vs fixtures

The real server reflects connected hardware; the mock reflects `fixtures.devices` from the YAML scenario.

So comparisons must be explicit about *which* “world” is being compared:
- “real hardware present” vs “single fixture device”
- simulation devices included/excluded

### 2) File I/O semantics (`LoadCapture`, exports)

The mock is configurable:
- `LoadCapture.validate.require_file_exists` defaults to `false` in the plan compiler (tests can opt-in).
- exports/save can write placeholder files (mock-side side effects).

The real server almost certainly has real file existence constraints and writes real data.

**Comparison approach:** treat file contents as out-of-scope initially; focus on:
- status codes for obvious invalid inputs
- whether calls succeed given a valid setup

### 3) Timing and `WaitCapture` semantics

The mock has explicit policy knobs:
- `defaults.timing.wait_capture_policy`: `immediate`, `error_if_running`, `block_until_done`
- `defaults.timing.max_block_ms`: cap for blocking behavior (when enabled)

Default behavior in the mock is intentionally “test-friendly” (often **non-blocking**), which can differ from the real server’s “block until done” behavior.

**Comparison approach:**
- Compare `WaitCapture` behavior per policy, not “one true behavior.”
- For contract matching, aim to match the real server’s observed behavior with an explicit mock policy.

### 4) Nondeterministic values

Some fields are inherently unstable across runs:
- `AppInfo.launch_pid`
- version strings (real Logic 2 version)
- capture IDs (if not deterministic)

**Comparison approach:** normalize these fields (ignore them or compare only shape/ranges).

## Recommended comparison workflow (practical, incremental)

### Phase A: Prove endpoint switching + avoid false conclusions

- Always run real and mock on **different ports**.
- If something “looks wrong,” assume port collision / wrong server answering until proven otherwise (this already burned us once in ticket 010).

Concrete “switch endpoints” examples (adjust ports if you changed them):

```bash
# Real server baseline (default salad port is 10430)
go run ./cmd/salad --host 127.0.0.1 --port 10430 appinfo
go run ./cmd/salad --host 127.0.0.1 --port 10430 devices

# Start mock server (default salad-mock port is 10431)
go run ./cmd/salad-mock --config salad/configs/mock/happy-path.yaml --host 127.0.0.1 --port 10431

# Compare against mock
go run ./cmd/salad --host 127.0.0.1 --port 10431 appinfo
go run ./cmd/salad --host 127.0.0.1 --port 10431 devices
```

### Phase B: Start with a small CLI “contract suite”

Pick a handful of `salad` flows that are meaningful and stable:
- `appinfo`
- `devices` (with/without simulation devices if there’s a CLI flag)
- capture lifecycle (against the mock: `LoadCapture` → `SaveCapture` → `CloseCapture`)
- exports (only assert “files created” for the mock placeholder case)

Run each flow against:
- the real server (note: real may require actual devices/captures)
- the mock server with `salad/configs/mock/happy-path.yaml`

Record outputs and call out differences as:
- **expected divergence** (hardware reality, file I/O, nondeterminism)
- **mock bug** (status codes, validation rules, missing state transitions)

### Phase C: Tighten with RPC-level assertions where it matters

For RPCs that are central to salad correctness (and should be matchable regardless of hardware), add RPC-level checks:
- unknown capture ID: what status code?
- empty filepath validation
- `WaitCapture` manual mode behavior
- fault injection semantics (mock-only, but ensure it doesn’t break “normal” behavior)

## Suggested “diff format” (so this stays actionable)

When we discover a difference, record it as:
- **Method**: `SaveCapture`
- **Setup**: capture exists? filepath? nth call?
- **Real**: status code + message + response shape
- **Mock**: status code + message + response shape
- **Conclusion**: bug vs divergence
- **Fix plan**: YAML knob change vs code change (`internal/mock/saleae/server.go`, `plan.go`, etc.)

## Where to store comparison results

Use this ticket’s workspace for evidence:
- `various/` for raw runs (captured stdout/stderr, transcripts)
- `reference/` for stable “known differences” tables once validated

