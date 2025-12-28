---
Title: 'Implementation: pipeline runner (run/repro/watch)'
Ticket: 006-PIPELINES
Status: active
Topics:
    - go
    - saleae
    - logic-analyzer
    - client
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/salad/cmd/root.go
      Note: Cobra root command; place to wire new `run/repro/watch` (or `pipeline`) verbs
    - Path: cmd/salad/cmd/run.go
      Note: `salad run --config ...` entry point (pipeline v1)
    - Path: cmd/salad/cmd/capture.go
      Note: Existing capture lifecycle commands (pipelines orchestrate these today via `capture load/...`)
    - Path: cmd/salad/cmd/analyzer.go
      Note: Existing analyzer add/remove commands (pipelines orchestrate these)
    - Path: cmd/salad/cmd/export.go
      Note: Existing export commands (raw-csv/raw-binary)
    - Path: cmd/salad/cmd/export_table.go
      Note: Existing table export command (ExportDataTableCsv) pipelines can call
    - Path: internal/saleae/client.go
      Note: gRPC wrapper methods that pipeline runner will call directly
    - Path: internal/pipeline/config.go
      Note: Pipeline config model + YAML/JSON loader (v1)
    - Path: internal/pipeline/runner.go
      Note: Pipeline execution (capture load → analyzers add → exports → close)
    - Path: ttmp/2025/12/24/006-PIPELINES--pipelines-run-repro-watch-workflows/scripts/README.md
      Note: Copy/paste commands to validate pipelines against salad-mock (fast iteration loop)
    - Path: ttmp/2025/12/24/006-PIPELINES--pipelines-run-repro-watch-workflows/scripts/01-pipeline-mock.yaml
      Note: Example pipeline config exercised against salad-mock
    - Path: ttmp/2025/12/24/006-PIPELINES--pipelines-run-repro-watch-workflows/scripts/00-mock-happy-path-with-table.yaml
      Note: Mock-server scenario enabling `ExportDataTableCsv` placeholder writes for pipeline tests
    - Path: internal/mock/saleae
      Note: Mock server + CLI integration tests; good harness for pipeline tests
    - Path: ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/playbook/01-debugging-the-mock-server.md
      Note: Playbook + scripts for repeatable mock-server testing
    - Path: ttmp/2025/12/24/001-INITIAL-SALAD--initial-salad-saleae-logic-analyzer-client-in-go/playbook/01-manual-smoke-test-logic2-automation.md
      Note: Real-server smoke test guidance (GetAppInfo/GetDevices)
    - Path: ttmp/2025/12/24/005-EXPORT-TABLE--export-decoded-data-tables-csv-filtering/scripts/01-real-export-table.sh
      Note: Example of ticket-local real-server validation script pattern
ExternalSources: []
Summary: "Implementation approach for pipeline commands (`run`, `repro`, `watch`) that orchestrate capture/analyzer/HLA/export steps from a config file and produce reproducible artifacts."
LastUpdated: 2025-12-28T00:00:00Z
WhatFor: ""
WhenToUse: ""
---

# Implementation: pipeline runner (run/repro/watch)

## Goal

Provide high-level “one command” workflows that chain multiple RPC operations:
- `salad run --config pipeline.yaml` (execute a pipeline)
- `salad repro` (rerun last pipeline session)
- `salad watch --config ...` (loop pipeline until condition, CI-style)

## Current reality (as of today)

This ticket was drafted early. The current codebase now has solid building blocks (capture/analyzer/export), and an initial **pipeline v1 runner (`salad run`) now exists**.

### What exists today (concrete files + symbols)

- **CLI building blocks** (cobra commands under `cmd/salad/cmd/`):
  - Capture lifecycle: `capture load/save/stop/wait/close` (`cmd/salad/cmd/capture.go`)
  - Analyzer ops: `analyzer add/remove` (`cmd/salad/cmd/analyzer.go`)
  - Export raw: `export raw-csv`, `export raw-binary` (`cmd/salad/cmd/export.go`)
  - Export table: `export table` (`cmd/salad/cmd/export_table.go`)
  - Pipeline v1: `run --config <file>` (`cmd/salad/cmd/run.go`)

- **gRPC wrappers** (pipeline runner should call these directly, not shell out):
  - `internal/saleae.(*Client).LoadCapture`, `SaveCapture`, `StopCapture`, `WaitCapture`, `CloseCapture`
  - `internal/saleae.(*Client).AddAnalyzer`, `RemoveAnalyzer`
  - `internal/saleae.(*Client).ExportRawDataCsv`, `ExportRawDataBinary`, `ExportDataTableCsv`

- **Testing harness**:
  - Mock server in `internal/mock/saleae/` (used by CLI integration tests)
  - Repeatable mock-server scripts + playbook:
    - `ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/scripts/02-smoke-happy-path.sh`
    - `ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/playbook/01-debugging-the-mock-server.md`

### Constraints / missing prerequisites (affects pipeline v1 scope)

- **No `StartCapture` CLI verb yet** (ticket 002).  
  So a pipeline runner **cannot** do “device → start capture” today without adding that command first.
- **No sessions/manifest (`--session`, `--last`) support yet** (ticket 007).  
  So `repro` needs a storage decision (where “last run” lives) or must be stubbed initially.

## Proposed v1 scope (implementable now)

Implement **only what can run today**:

- `salad run --config <file>` supports:
  - `capture.load` (load a `.sal`)
  - `analyzers` (LLA only via AddAnalyzer; no HLAs yet)
  - `exports`:
    - raw-csv
    - raw-binary
    - table-csv (ExportDataTableCsv)
  - `capture.close` at end (best-effort cleanup)

`salad repro` / `salad watch` can be:
- implemented as stubs with clear error messages, until ticket 007 (sessions) and ticket 002 (start capture) land.

## Why this matters

Most Saleae debug loops are repetitive:
device → capture → analyzers → export → inspect.

A pipeline runner makes this:
- reproducible
- scriptable
- easy to share as a single config file

## Inputs (config-first)

Use a config file (YAML/JSON), not dozens of flags.

**Important:** The example below shows the “end state” (device selection, capture start, HLAs). Pipeline v1 currently supports only `capture.load` + LLA analyzers + exports.

```yaml
device:
  pick: first-physical   # or explicit device_id
capture:
  start:
    config: capture.yaml
analyzers:
  - type: lla
    name: "SPI"
    label: "boot spi"
    settings: spi-settings.yaml
  - type: hla
    extension_dir: /abs/extensions/i2c-utils
    name: "I2C EEPROM Reader"
    label: "eeprom"
    input_analyzer_ref: "SPI"   # reference by label/name, resolved to analyzer_id
exports:
  - type: raw-csv
    directory: /abs/out/raw
    digital: [0,1,2]
  - type: table-csv
    filepath: /abs/out/table.csv
    analyzers:
      - ref: "boot spi"
        radix: hex
```

## Implementation approach

### Orchestration layer

Add `internal/pipeline/` (new) that:
- parses config
- resolves references (capture_id, analyzer_ids)
- runs steps sequentially (most operations depend on previous IDs)
- (future) records a manifest compatible with ticket 007

### Concurrency

- Use sequential execution by default.
- When safe (e.g. exporting raw + table from same capture), allow parallel exports via `errgroup`.

### CLI commands

- Current:
  - `cmd/salad/cmd/run.go` (implemented)
- Planned:
  - `cmd/salad/cmd/repro.go`
  - `cmd/salad/cmd/watch.go`

### Failure model

- Any step error stops the pipeline and returns non-zero.
- `watch` can evaluate a condition (future):
  - regex match on exported CSV
  - “device present” checks
  - etc.

## Dependencies / prerequisites

This ticket depends on the existence of lower-level commands:
- capture start (Ticket 002)
- analyzers + HLAs (Tickets 003/004)
- exports (raw already exists; table export in Ticket 005)
- sessions/manifest (Ticket 007)

As implemented reality today:
- Exports (raw + table) exist.
- Analyzer LLA add/remove exists (ticket 003 landed).
- Capture start does not exist yet; pipelines should start from `capture load` for now.
- Sessions/manifest does not exist yet; `repro` should be delayed or stubbed.

## Testing strategy

- Unit tests:
  - pipeline config parsing + validation
  - reference resolution (labels/refs → analyzer IDs)
- Integration tests:
  - run pipeline against the mock server (`internal/mock/saleae`)
  - reuse the mock-server playbook + scripts for rapid iteration
- Manual/real-server validation:
  - follow the real-server smoke playbook (ticket 001)
  - model a pipeline real-server script after ticket 005’s `01-real-export-table.sh` pattern

## Open questions / decisions

- Where to store “last pipeline session” pointer:
  - a global default dir (`~/.salad/…`) vs repo-local `.salad/…`
- Condition language for `watch`:
  - simple regex vs a small expression DSL
