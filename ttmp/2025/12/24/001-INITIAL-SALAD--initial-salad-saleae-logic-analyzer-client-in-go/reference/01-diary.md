---
Title: Diary
Ticket: 001-INITIAL-SALAD
Status: active
Topics:
    - go
    - saleae
    - logic-analyzer
    - client
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Step-by-step implementation diary for building the Saleae Logic2 gRPC Go client."
LastUpdated: 2025-12-24T21:13:18.87518109-05:00
WhatFor: ""
WhenToUse: ""
---

# Diary

## Goal

Capture the implementation journey for `001-INITIAL-SALAD`: what we changed, why we changed it, what worked/didn’t, and the key gotchas for continuing the work later.

## Step 1: Create initial docs + break down the work

This step established the documentation scaffolding so implementation can proceed without losing context. I pulled the existing research note into the ticket workspace, created a build-plan analysis doc, and seeded a concrete task list to drive incremental delivery.

The immediate outcome is that we have a single place to track decisions (analysis), narrative progress (this diary), and execution (tasks.md), before touching Go code.

### What I did
- Created `analysis/01-build-plan-saleae-logic2-grpc-client-go.md` and filled it with an initial architecture + CLI + proto strategy.
- Created this diary document for frequent step logging.
- Seeded `tasks.md` with concrete implementation milestones (proto pinning, Go module, bindings, CLI commands).

### Why
- The Saleae automation API is beta and proto-driven; pinning + generation decisions need to be explicit and easy to revisit.
- A diary reduces “lost time” when debugging gRPC/proto/tooling issues later.

### What worked
- `docmgr` is correctly configured to use `salad/ttmp` as the docs root, and the moved ticket is recognized.

### What didn't work
- N/A

### What I learned
- The fastest path is “proto first”: once `saleae.proto` + generated bindings exist, `appinfo` becomes a straightforward Cobra + gRPC dial + RPC call.

### What was tricky to build
- N/A (documentation-only step)

### What warrants a second pair of eyes
- Proto pinning approach: confirm we’re happy committing generated code in-repo to avoid requiring `protoc` for normal builds.

### What should be done in the future
- N/A

### Code review instructions
- Start with the analysis doc: `analysis/01-build-plan-saleae-logic2-grpc-client-go.md`
- Then review `tasks.md` for the execution plan.

### Technical details
- Next manual smoke test target (once CLI exists):
  - `Logic-2.AppImage --automation --automationPort 10430`
  - `go run ./cmd/salad appinfo --host 127.0.0.1 --port 10430`

### What I'd do differently next time
- N/A

## Step 2: Vendor proto + generate Go bindings (w/ go_package mapping)

This step made the project “real” by pinning the upstream Saleae proto and generating Go bindings into `gen/`. That unblocks actual client code: once the bindings compile, the first CLI command (`appinfo`) is just dial + RPC + print.

It also surfaced a subtle tooling change: `protoc-gen-go-grpc` is now versioned as its own module path, so a naïve `go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@<grpc-version>` fails.

### What I did
- Vendored `proto/saleae/grpc/saleae.proto` from upstream `saleae/logic2-automation` and recorded the pinned SHA.
- Initialized `go.mod` (`module github.com/go-go-golems/salad`).
- Installed pinned generators:
  - `google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11`
  - `google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.0`
- Generated bindings using an explicit `M...=...` mapping (because upstream proto has no `option go_package`):
  - output: `gen/saleae/automation/{saleae.pb.go,saleae_grpc.pb.go}`

### Why
- The automation API is proto-first; pinning and generation must be deterministic and reviewable.
- Using `M...` mapping avoids maintaining a local patch to upstream `saleae.proto`.

### What worked
- `protoc` is available (`libprotoc 3.21.12`), and generation produced the expected Go files under `gen/`.

### What didn't work
- First attempt to install the gRPC plugin failed:

```
go: google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.78.0: module google.golang.org/grpc@v1.78.0 found, but does not contain package google.golang.org/grpc/cmd/protoc-gen-go-grpc
```

Fix was to use the plugin’s own module path/version (`google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.0`).

### What I learned
- `protoc-gen-go-grpc` should be treated as an independent tool dependency (separate version stream) from the runtime `google.golang.org/grpc` dependency.

### What was tricky to build
- Getting generation right without `go_package`: it requires both `--go_opt=Msaleae/grpc/saleae.proto=...` and `--go-grpc_opt=Msaleae/grpc/saleae.proto=...`, plus `--*_opt=module=...` so output paths land under `gen/`.

### What warrants a second pair of eyes
- Confirm the chosen Go import path mapping (`github.com/go-go-golems/salad/gen/saleae/automation`) is acceptable before we write lots of code against it.

### What should be done in the future
- If we later rename the Go module path, we must re-run `protoc` with the updated `--*_opt=module=...` and mapping, and regenerate the bindings.

### Code review instructions
- Review the vendored proto pin: `proto/saleae/grpc/` (README + SHA file + proto).
- Review generated output: `gen/saleae/automation/`.

### Technical details
- Pinned upstream commit: `0d7ca19dcc667ca8420ec748d98cf86d4c1f8b78`
- Generation command shape (conceptually):
  - `protoc -I proto ... saleae/grpc/saleae.proto`

### What I'd do differently next time
- Add a Makefile target early to make regeneration a single obvious command, and document the exact generator versions next to it.

## Step 3: Add Cobra CLI + gRPC client wrapper + `appinfo`

This step introduced the first “vertical slice” that actually compiles and can be run against a real Logic 2 instance: a Cobra-based CLI (`cmd/salad`) plus an internal `saleae` client wrapper that dials gRPC and calls `GetAppInfo`.

The key goal here was to keep the CLI thin and push all gRPC concerns (dial, credentials, error wrapping) into `internal/saleae`, so future commands are incremental additions rather than copy/paste.

### What I did
- Created a new CLI binary at `cmd/salad`.
- Added global flags:
  - `--host`, `--port`, `--timeout`, `--log-level`
- Implemented `salad appinfo` calling `Manager.GetAppInfo` and printing:
  - `application_version`, `api_version`, `launch_pid`
- Added an internal client wrapper:
  - `internal/saleae.New(ctx, Config)` (dial + `pb.NewManagerClient`)
  - `internal/saleae.Client.GetAppInfo(ctx)` (RPC + basic nil checks)
- Ran `go mod tidy` and verified compilation with `go test ./...`.

### Why
- `appinfo` is the smallest useful RPC and validates the entire stack: flags → dial → RPC → output.
- Centralizing gRPC logic keeps upcoming capture/export commands simple.

### What worked
- `go test ./...` passes (no tests yet, but compilation is clean).

### What didn't work
- N/A

### What I learned
- The generated proto types match the proto fields (e.g. `launch_pid` → `GetLaunchPid()`), so it’s safest to lean on generated accessors instead of assumptions from older forum snippets.

### What was tricky to build
- Keeping timeouts consistent: we currently reuse `--timeout` for both dialing (when no context deadline exists) and RPC calls.

### What warrants a second pair of eyes
- The dial behavior (`grpc.WithBlock()` + timeout semantics): confirm this is the desired UX (fail fast) versus allowing background connection attempts.

### What should be done in the future
- Split `--timeout` into `--dial-timeout` and `--rpc-timeout` if we find we need different behavior.

### Code review instructions
- Start at `cmd/salad/cmd/root.go` for flags + logging.
- Then review `cmd/salad/cmd/appinfo.go` for the first RPC command.
- Finally check `internal/saleae/client.go` for dial + error wrapping decisions.

### Technical details
- Manual run (once Logic 2 is running with automation enabled):
  - `go run ./cmd/salad appinfo --host 127.0.0.1 --port 10430`

### What I'd do differently next time
- Add a minimal playbook document at the same time as the first runnable command (so testers don’t need to infer the invocation).

## Step 4: Add `devices` command + write the manual smoke-test playbook

This step expanded the “vertical slice” beyond `appinfo` by adding a second RPC-backed CLI command: `devices` (`GetDevices`). In parallel, I wrote a short playbook so validating the tool against a running Logic 2 instance is repeatable without tribal knowledge.

I also removed `.git-commit-message.yaml` after feedback that we should not auto-generate commit-message artifacts in this repo.

### What I did
- Deleted `.git-commit-message.yaml` (we will not generate these going forward).
- Added `salad devices`:
  - optional `--include-simulation-devices`
  - prints `device_id`, `device_type`, `is_simulation`
- Added `Client.GetDevices(...)` wrapper in `internal/saleae`.
- Created playbook: `playbook/01-manual-smoke-test-logic2-automation.md` with exact run commands and troubleshooting notes.
- Confirmed compilation still works:
  - `gofmt -w ...`
  - `go test ./...`

### Why
- `GetDevices` is a practical next RPC to verify the connection and confirm the automation API can see attached hardware.
- The playbook turns “it works on my machine” into a repeatable checklist for anyone onboarding.

### What worked
- `go test ./...` remains clean after adding the new command.

### What didn't work
- N/A

### What I learned
- Adding new commands is now mostly “plumbing”: wrapper method + Cobra command + output formatting.

### What was tricky to build
- N/A (straightforward extension of the existing pattern)

### What warrants a second pair of eyes
- Output format: confirm key/value lines are the desired default (vs JSON/table).

### What should be done in the future
- Decide on stable machine-readable output (`--json`) once we add more commands (captures/exports).

### Code review instructions
- Start with `cmd/salad/cmd/devices.go`.
- Then check `internal/saleae/client.go` for the new wrapper method.
- Finally review `playbook/01-manual-smoke-test-logic2-automation.md`.

### Technical details
- Manual runs:
  - `go run ./cmd/salad appinfo --host 127.0.0.1 --port 10430`
  - `go run ./cmd/salad devices --host 127.0.0.1 --port 10430`

### What I'd do differently next time
- Add a small integration test harness once we can run Logic2 in CI or mock the server (probably not feasible immediately).

## Step 5: Live test against a running Logic 2 automation server

This step validated the CLI end-to-end against a real Logic 2 instance you launched with automation enabled. The key outcome is that our gRPC dial settings and request wiring are correct: we can fetch app metadata and enumerate devices (including simulation devices).

This also gives us a “known good” baseline output shape to preserve as we add more commands (capture/analyzer/export).

### What I did
- Ran the CLI against `127.0.0.1:10430`:
  - `go run ./cmd/salad appinfo --host 127.0.0.1 --port 10430`
  - `go run ./cmd/salad devices --host 127.0.0.1 --port 10430`
  - `go run ./cmd/salad devices --include-simulation-devices --host 127.0.0.1 --port 10430`

### What worked
- `appinfo` returned:

```
application_version=2.4.40
api_version=1.0.0
launch_pid=3691383
```

- `devices` returned a real device:

```
device_id=9E7A9F3533975A55 device_type=DEVICE_TYPE_LOGIC_PRO_8 is_simulation=false
```

- `devices --include-simulation-devices` returned simulation devices as expected.

### What didn't work
- N/A

### What I learned
- The API version reported by Logic 2 (`api_version=1.0.0`) matches the vendored proto header expectations.

### What was tricky to build
- N/A (validation-only step)

### What warrants a second pair of eyes
- Confirm the human-readable output format is what we want to stabilize before adding more commands (key/value lines vs structured output).

### What should be done in the future
- Add a short “expected output example” section to the playbook if we want a stricter smoke-test checklist.

### Code review instructions
- N/A (no code changes in this step)

**Commit (code):** b4e51a4db6c9477bae367c21805e0b98ef176aea — "Add Saleae Logic2 gRPC client (appinfo/devices)"

**Commit (docs):** add0340564db240c2eab408251bdef8b14e85ade — "Docs: record live test + task bookkeeping"

## Step 6: Add capture/load/save/stop/wait/close and raw export skeleton commands

This step expands the CLI surface area into the next “useful” operations beyond discovery: working with capture files and invoking raw data exports. The focus is on having commands that compile and map cleanly to the underlying RPCs, even if we’re not yet driving `StartCapture` (which requires a full `CaptureConfiguration` + `LogicDeviceConfiguration`).

With this in place, the next work becomes incremental: add `capture start` (manual/timed/trigger) and then analyzer/table export support.

### What I did
- Added capture subcommands:
  - `salad capture load --filepath /abs/path/to/file.sal`
  - `salad capture save --capture-id <id> --filepath /abs/path/to/out.sal`
  - `salad capture stop|wait|close --capture-id <id>`
- Added export subcommands:
  - `salad export raw-csv --capture-id <id> --directory /abs/dir --digital 0,1,2 [--analog 0,1]`
  - `salad export raw-binary --capture-id <id> --directory /abs/dir --digital 0,1,2 [--analog 0,1]`
- Implemented corresponding wrapper methods in `internal/saleae.Client`:
  - `LoadCapture`, `SaveCapture`, `StopCapture`, `WaitCapture`, `CloseCapture`
  - `ExportRawDataCsv`, `ExportRawDataBinary`
- Verified compilation:
  - `gofmt -w ...`
  - `go test ./...`

### Why
- These RPCs have simple request shapes (filepath and capture_id) and are a safe step before the more complex `StartCapture` configuration story.
- Export commands force us to handle proto `oneof` fields correctly (channels).

### What worked
- The Cobra command tree renders as expected (`salad capture --help`, `salad export --help`).

### What didn't work
- N/A

### What I learned
- The generated Go types for `oneof channels` require setting:
  - `Channels: &pb.ExportRawDataCsvRequest_LogicChannels{LogicChannels: ...}`

### What was tricky to build
- Ensuring the channel parsing is strict enough to avoid silently exporting “nothing” (we error if both `--digital` and `--analog` are empty).

### What warrants a second pair of eyes
- Flag UX: confirm we want comma-separated channel lists as the primary interface vs repeatable flags.

### What should be done in the future
- Add `capture start` once we decide on an ergonomic way to express `CaptureConfiguration` and `LogicDeviceConfiguration` (likely via JSON/YAML input).

### Code review instructions
- Start at:
  - `cmd/salad/cmd/capture.go`
  - `cmd/salad/cmd/export.go`
- Then review:
  - `internal/saleae/client.go`
