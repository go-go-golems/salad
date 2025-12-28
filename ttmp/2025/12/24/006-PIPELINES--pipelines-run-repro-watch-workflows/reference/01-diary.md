---
Title: Diary
Ticket: 006-PIPELINES
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
LastUpdated: 2025-12-27T19:45:28.09549823-05:00
WhatFor: ""
WhenToUse: ""
---

# Diary

## Goal

Capture the step-by-step implementation of ticket **006-PIPELINES**: config-driven “one command” workflows (`run`, `repro`, `watch`) that orchestrate existing `salad` commands/RPCs into a reproducible pipeline.

## Step 1: Re-orient on current codebase + constraints

This step establishes what exists today vs what is planned in the early analysis doc, so we can update the doc to reflect reality and pick a first implementation slice that can actually run in CI.

The key observation is that “pipelines” are currently **docs-only**: there are no pipeline commands (`run/repro/watch`) or pipeline packages yet. However, the lower-level building blocks pipelines want to orchestrate do exist: capture load/save/stop/wait/close, analyzer add/remove, and exports (raw + data table).

### What I did
- Read the 006 ticket docs (`index.md`, `tasks.md`, and the initial analysis doc).
- Scanned the CLI command tree under `cmd/salad/cmd/` to confirm which verbs exist.
- Identified existing testing harnesses we can reuse for pipeline validation (mock server scripts/playbooks).

### What I found (current reality)
- **No pipeline commands exist yet**:
  - there is no `cmd/salad/cmd/run.go`, `repro.go`, or `watch.go`
  - there is no `internal/pipeline/` package
- **Pipeline building blocks that exist and are usable now**:
  - `capture load/save/stop/wait/close` in `cmd/salad/cmd/capture.go`
  - `analyzer add/remove` in `cmd/salad/cmd/analyzer.go`
  - `export raw-csv/raw-binary` in `cmd/salad/cmd/export.go`
  - `export table` in `cmd/salad/cmd/export_table.go`
  - gRPC wrappers in `internal/saleae/client.go`:
    - `(*Client).LoadCapture`, `SaveCapture`, `CloseCapture`
    - `(*Client).AddAnalyzer`, `RemoveAnalyzer`
    - `(*Client).ExportRawDataCsv`, `ExportRawDataBinary`, `ExportDataTableCsv`
- **Key missing dependencies** (affects pipeline v1 scope):
  - CLI does not expose `StartCapture` yet (ticket 002), so initial pipelines must start from `capture load` (existing `.sal` file) or from a pre-existing capture id (not supported today).
  - No sessions/manifest support yet (ticket 007), so `repro` can’t be implemented “for real” without deciding on where “last run” metadata lives.

### Testing resources worth reusing
- Mock-server smoke + debugging scripts:
  - `ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/scripts/02-smoke-happy-path.sh`
  - `ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/playbook/01-debugging-the-mock-server.md`
- Real-server smoke guidance:
  - `ttmp/2025/12/24/001-INITIAL-SALAD--initial-salad-saleae-logic-analyzer-client-in-go/playbook/01-manual-smoke-test-logic2-automation.md`
- A good example of “ticket-local real-server script”:
  - `ttmp/2025/12/24/005-EXPORT-TABLE--export-decoded-data-tables-csv-filtering/scripts/01-real-export-table.sh`

### What should be done next
- Update the 006 analysis doc to reflect the above constraints and point directly at current code/symbols.
- Implement a minimal `salad run --config` that supports: `capture.load` → LLA analyzers → exports → close.

## Step 2: Update 006 analysis doc to match the current repo

This is the documentation equivalent of “touching the ground”: ensure the analysis doc references actual files/symbols and explicitly calls out which prerequisites are not implemented yet (capture start, sessions/manifest).

### What I did
- (ongoing) Update `analysis/01-implementation-pipeline-runner-run-repro-watch.md`:
  - replace “future file list” with current CLI building blocks + planned pipeline files
  - add links to existing playbooks/scripts that matter for testing pipelines

## Step 3: Implement pipeline v1 skeleton (`salad run --config ...`)

This step starts the actual implementation with a deliberately small scope that matches what the repo can do today. The focus is a runnable `salad run` that loads a pipeline YAML/JSON, loads a capture from a `.sal`, adds LLA analyzers, runs exports, and then closes the capture best-effort.

### What I did
- Added `internal/pipeline/`:
  - `pipeline.Load(path)` parses YAML/JSON configs (versioned, currently v1)
  - `pipeline.Runner.Run(ctx,cfg)` executes: capture load → analyzers add → exports → close
- Added `salad run` cobra command (`cmd/salad/cmd/run.go`) and wired it into `cmd/salad/cmd/root.go`.

### Why
- This provides immediate value without waiting on `StartCapture` or sessions/manifest work.
- It also gives us a concrete target for tests (mock server) and future expansion (repro/watch, capture start).

### What warrants a second pair of eyes
- Config schema naming: `settings_yaml/settings_json`, `set_*` keys, export shapes.
- Analyzer “ref” semantics: currently uses label if provided, otherwise name; this requires labels to be unique if used for table export refs.

## Step 4: Add ticket-local scripts to run the pipeline against the mock server

This step stores runnable scripts/configs under the 006 ticket so it’s easy to reproduce and iterate on pipeline behavior without digging through shell history. It also intentionally reuses the mock-server tmux scripts and playbook from ticket 010 (those scripts are battle-tested).

### What I did
- Added pipeline config example:
  - `ttmp/2025/12/24/006-PIPELINES--pipelines-run-repro-watch-workflows/scripts/01-pipeline-mock.yaml`
- Added a mock-server scenario that enables `ExportDataTableCsv` placeholder writes:
  - `ttmp/2025/12/24/006-PIPELINES--pipelines-run-repro-watch-workflows/scripts/00-mock-happy-path-with-table.yaml`
- Added `scripts/README.md` with copy/paste commands for:
  - starting `salad-mock` via `ttmp/.../010.../scripts/10-tmux-mock-start.sh`
  - running `salad run --config ...` against the mock server
  - running a wrapper script that does “kill port → start tmux mock → run pipeline → verify artifacts → stop mock”

## Step 5: Add a ticket-local real-server validation script

This step mirrors the ticket 005 pattern: keep a runnable “real server” script in the ticket so we can quickly validate the pipeline runner against an actual Logic 2 instance.

### What I did
- Added `ttmp/2025/12/24/006-PIPELINES--pipelines-run-repro-watch-workflows/scripts/02-real-run.sh`:
  - generates a small pipeline config in `/tmp`
  - runs `salad run --config ...` against a real server (defaults to `127.0.0.1:10430`)
  - requires you to set `SAL=/abs/path/to/capture.sal`
