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
RelatedFiles: []
ExternalSources: []
Summary: "Implementation approach for pipeline commands (`run`, `repro`, `watch`) that orchestrate capture/analyzer/HLA/export steps from a config file and produce reproducible artifacts."
LastUpdated: 2025-12-24T22:42:12.946401849-05:00
WhatFor: ""
WhenToUse: ""
---

# Implementation: pipeline runner (run/repro/watch)

## Goal

Provide high-level “one command” workflows that chain multiple RPC operations:
- `salad run --config pipeline.yaml` (execute a pipeline)
- `salad repro` (rerun last pipeline session)
- `salad watch --config ...` (loop pipeline until condition, CI-style)

## Why this matters

Most Saleae debug loops are repetitive:
device → capture → analyzers → export → inspect.

A pipeline runner makes this:
- reproducible
- scriptable
- easy to share as a single config file

## Inputs (config-first)

Use a config file (YAML/JSON), not dozens of flags:

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

Add `internal/pipeline/` that:
- parses config
- resolves references (capture_id, analyzer_ids)
- runs steps sequentially (most operations depend on previous IDs)
- records a `manifest.json` with:
  - appinfo
  - device selection
  - capture_id
  - analyzers created
  - output paths

### Concurrency

- Use sequential execution by default.
- When safe (e.g. exporting raw + table from same capture), allow parallel exports via `errgroup`.

### CLI commands

- `cmd/salad/cmd/run.go`
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

## Testing strategy

- Unit tests: config parsing + reference resolution
- Manual test: run pipeline against local Logic2 instance

## Open questions / decisions

- Where to store “last pipeline session” pointer:
  - a global default dir (`~/.salad/…`) vs repo-local `.salad/…`
- Condition language for `watch`:
  - simple regex vs a small expression DSL
