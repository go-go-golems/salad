---
Title: Diary
Ticket: 011-MOCK-AGAINST-REAL
Status: active
Topics:
    - saleae
    - mock-server
    - testing
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2025-12-27T16:51:53.693526559-05:00
WhatFor: ""
WhenToUse: ""
---

# Diary

## Goal

Keep a step-by-step diary of work to compare `salad-mock` against the real Saleae Logic 2 automation server, so we can prove (and continuously re-prove) the mock’s correctness and identify intentional divergences.

## Step 1: Create ticket + identify the “switching” knobs (host/port)

This step set up the ticket workspace and validated that the system already has a clean switch between “real server” and “mock server”: the gRPC client dials `host:port`, and both CLIs expose `--host/--port` flags. That means we can run the same `salad` command suite twice (once per endpoint) without code changes, as long as we avoid port collisions.

### What I did
- Created ticket `011-MOCK-AGAINST-REAL` with an analysis document and this diary.
- Read the existing `010-MOCK-SERVER` docs to find the relevant code locations and common pitfalls (especially port collisions).
- Verified the endpoint selection knobs:
  - `salad` defaults to `127.0.0.1:10430` via persistent flags.
  - `salad-mock` defaults to `127.0.0.1:10431` via flags.
  - `salad/internal/saleae/client.go` dials `cfg.Addr()` (no special-casing for mock vs real).

### Why
- Without a clean way to switch endpoints, any “compare real vs mock” work devolves into ad-hoc editing and brittle scripts.
- `010-MOCK-SERVER` already highlighted that **port collisions** lead to false conclusions (“wrong server answering”), so we need to design the comparison workflow to be collision-resistant.

### What worked
- The repo already has a simple, explicit switch:
  - `salad --host ... --port ...`
  - `salad-mock --host ... --port ...`

### What didn't work
- N/A (this was a documentation + discovery step).

### What I learned
- The real-vs-mock switch is purely an address selection problem today. That’s good: we can start with a CLI-level comparison harness immediately.

### What was tricky to build
- Avoiding “wrong-server” confusion requires operational guardrails (port checks, explicit ports per workflow, logs).

### What warrants a second pair of eyes
- Confirm the chosen “comparison harness” doesn’t accidentally compare different scenarios (e.g., real server with real devices vs mock server fixtures) without labeling it as such.

### What should be done in the future
- Add a recording/replay harness at the RPC level (or gRPC interceptor-based transcript) so comparisons don’t rely solely on CLI stdout formatting.

### Code review instructions
- Start with:
  - `salad/cmd/salad/cmd/root.go` (real default port and flags)
  - `salad/cmd/salad-mock/cmd/root.go` (mock default port and flags)
  - `salad/internal/saleae/client.go` (dial semantics)
  - `salad/ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/playbook/01-debugging-the-mock-server.md` (port collision playbook)

### Technical details
- Defaults discovered:
  - Real: `127.0.0.1:10430`
  - Mock: `127.0.0.1:10431`

## Related

- `../analysis/01-mock-vs-real-comparison-strategy.md`

## Step 2: Write the comparison strategy (what to compare + where drift is likely)

This step captured the concrete plan for comparing “mock vs real” at multiple layers (CLI-level, RPC-level, transcript record/replay), and pulled out the most important drift risks from the mock implementation itself. The most valuable discovery is that the mock already has explicit knobs for the areas most likely to differ from the real server (especially timing / `WaitCapture` behavior), so the comparison should focus on *observed real behavior* → *explicit mock policy*.

### What I did
- Wrote `analysis/01-mock-vs-real-comparison-strategy.md` describing comparison layers and a recommended incremental workflow.
- Skimmed the mock scenario YAMLs and compiler defaults to understand what is configurable vs hard-coded.

### Why
- Without an explicit “comparison contract,” we’ll confuse intentional divergences (hardware reality, file I/O) with mock bugs.
- Capturing where drift is likely helps us focus the comparison suite on semantics that matter (status codes, validation, state transitions).

### What worked
- The mock’s YAML already exposes many of the likely-drift semantics:
  - fixtures (devices/captures/appinfo)
  - behavior knobs for validation, transitions, side effects
  - faults (deterministic error injection)

### What I learned
- `WaitCapture` in the mock is governed by:
  - `defaults.timing.wait_capture_policy`: `immediate`, `error_if_running`, `block_until_done`
  - `defaults.timing.max_block_ms`: cap for blocking behavior (when enabled)
  This should map cleanly to whatever the real server does once we measure it.
- The baseline mock scenario file we should use for comparisons lives at `salad/configs/mock/happy-path.yaml`.

## Step 3: Implement an RPC-level probe harness (real vs mock)

This step implemented a small, repeatable probe tool that talks to **both** endpoints at the gRPC level and emits a JSON report plus a focused diff. The key idea is to compare the things that should be comparable across environments (gRPC status codes for “unknown capture id”, validation errors, API version), while treating “device inventory” and nondeterministic values (PID/version string) as *report-only* or optional diffs.

### What I did
- Added a Go probe tool under the ticket:
  - `scripts/probe_real_vs_mock.go`
- Added a wrapper script that runs it from the `salad` module root and writes into `various/` (no new directories):
  - `scripts/01-probe-real-vs-mock.sh`

### Why
- CLI stdout diffs are noisy and can hide gRPC semantics; probing gRPC directly lets us diff status codes and request/response shapes.
- The output is JSON so it can be checked into `various/` and diffed in PRs.

### What warrants a second pair of eyes
- The selected “probe calls” are intentionally conservative (no real file writes), but we should confirm they are safe/non-invasive against a real Logic 2 instance.

## Step 4: Run the first probe against the real server (baseline semantics)

This step ran the probe against the running real Logic server and the mock (happy-path scenario) and captured the full JSON output under `various/`. The key outcome is a concrete list of behavior differences that we can now either (a) encode as mock config for “real-like” scenarios, (b) change mock defaults, or (c) document as intentional divergences.

### What I did
- Started `salad-mock` briefly with `salad/configs/mock/happy-path.yaml` on port `10431`.
- Ran the probe against:
  - real: `127.0.0.1:10430`
  - mock: `127.0.0.1:10431`
- Saved the report:
  - `various/probe-20251227-165937.json`

### What worked
- We successfully connected to the real server (so port `10430` is correct in this environment) and extracted baseline semantics.

### What I learned
- The real server returns `codes.Aborted` for several “invalid/unknown” cases where the mock currently returns `codes.InvalidArgument`:
  - `WaitCapture` unknown capture id
  - `StopCapture` unknown capture id
  - `CloseCapture` unknown capture id
  - `LoadCapture` empty filepath
  - `LoadCapture` missing filepath
- The real server’s `GetDevices(include_sim=true)` returns additional simulation devices, which is a useful behavioral baseline for how the real server treats simulation devices.
