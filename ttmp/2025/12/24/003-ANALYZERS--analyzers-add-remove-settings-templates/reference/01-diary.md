---
Title: "Diary: Analyzers (add/remove + settings/templates)"
Ticket: 003-ANALYZERS
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
Summary: ""
LastUpdated: 2025-12-27T17:08:57.00980015-05:00
WhatFor: ""
WhenToUse: ""
---

# Diary: Analyzers (add/remove + settings/templates)

## Goal

Implement `salad analyzer add/remove` (ticket 003) in a way that can be tested against:
- a **real** Logic 2 automation gRPC server (manual smoke test), and
- the **mock** server (ticket 010) later for CI and deterministic tests.

## Step 1: Create implementation diary + identify minimum compile-green units

This step bootstraps the diary and sets up an execution cadence: implement in small compile-green increments, commit each increment, then record what happened (including failures) while it’s still fresh. This keeps progress reviewable and reduces the risk of “one huge commit” with unclear breakpoints.

**Commit (code):** N/A — documentation scaffolding

### What I did
- Created `ttmp/2025/12/24/003-ANALYZERS--analyzers-add-remove-settings-templates/reference/01-diary.md` via `docmgr doc add`
- Confirmed analyzer RPCs and message schemas in `proto/saleae/grpc/saleae.proto`:
  - `AddAnalyzer(AddAnalyzerRequest) -> AddAnalyzerReply`
  - `RemoveAnalyzer(RemoveAnalyzerRequest) -> RemoveAnalyzerReply`
- Identified the smallest compile-green implementation units to commit:
  - Add client wrappers (`internal/saleae/client.go`)
  - Add analyzer settings parsing (`internal/config/...`)
  - Add Cobra verbs (`cmd/salad/cmd/analyzer.go` + `cmd/salad/cmd/root.go`)

### Why
- The analyzer feature spans client + CLI + config parsing; splitting into small commits makes it much easier to review and debug.

### What worked
- The proto clearly defines analyzer settings as a typed oneof (`AnalyzerSettingValue`), which maps well to a small parsing layer.

### What didn't work
- N/A

### What I learned
- `AddAnalyzerReply` returns `analyzer_id` (uint64). That value should be printed by the CLI the same way capture commands print `capture_id=...`.

### What was tricky to build
- N/A (setup step)

### What warrants a second pair of eyes
- N/A (setup step)

### What should be done in the future
- N/A

### Code review instructions
- Start at `proto/saleae/grpc/saleae.proto` around `AddAnalyzerRequest` to confirm request/response contracts.

---

## Step 2: Add analyzer RPC wrappers to the Saleae client

This step adds small, typed wrappers to the Go client so the CLI can call analyzer RPCs without duplicating proto details at the command layer. The goal is to mirror the existing capture/export client style: validate inputs, call the gRPC method, and wrap errors with a stable prefix.

**Commit (code):** a574108c310947e3d47b71ad697d997fa22838f9 — "Saleae client: add AddAnalyzer/RemoveAnalyzer"

### What I did
- Added client methods in `internal/saleae/client.go`:
  - `AddAnalyzer(ctx, captureID, analyzerName, analyzerLabel, settings) (uint64, error)`
  - `RemoveAnalyzer(ctx, captureID, analyzerID) error`
- Ran:
  - `gofmt -w internal/saleae/client.go`
  - `go test ./... -count=1`

### Why
- Keeps Cobra commands simple: the CLI should build inputs and delegate RPC calling + error wrapping to the client layer.

### What worked
- `go test ./...` stayed green after adding the methods.

### What didn't work
- N/A

### What I learned
- The analyzer settings map type is `map[string]*pb.AnalyzerSettingValue` (typed oneof), so we’ll need a dedicated parsing layer for JSON/YAML + typed flag overrides.

### What was tricky to build
- Ensuring `settings` behaves predictably when omitted: we normalize `nil` to an empty map to avoid accidental nil-map surprises.

### What warrants a second pair of eyes
- Error message consistency (prefixes + argument naming) vs existing client methods (capture/export).

### What should be done in the future
- Add unit tests for settings parsing (once the parsing package exists). The client wrappers themselves are thin enough that tests are optional.

### Code review instructions
- Start in `internal/saleae/client.go`, search for `AddAnalyzer(` and `RemoveAnalyzer(`.
- Validate with:
  - `go test ./... -count=1`

