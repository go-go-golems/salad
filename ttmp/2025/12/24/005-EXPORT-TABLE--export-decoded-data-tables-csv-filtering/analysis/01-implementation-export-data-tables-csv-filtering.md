---
Title: 'Implementation: export data tables (CSV) + filtering'
Ticket: 005-EXPORT-TABLE
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
Summary: "Implementation approach for `salad export table` using ExportDataTableCsvRequest (multi-analyzer export, radix selection, column selection, filtering, timestamps)."
LastUpdated: 2025-12-24T22:42:12.674457974-05:00
WhatFor: ""
WhenToUse: ""
---

# Implementation: export data tables (CSV) + filtering

## Goal

Add `salad export table` to export decoded analyzer frames into CSV with:
- multi-analyzer selection
- radix selection per analyzer
- optional column selection
- optional query filtering
- ISO8601 timestamps option

This is the “power move” for CLI debugging: export decoded frames and then grep/parse in scripts.

## Proto grounding

Relevant RPC and message types:
- `Manager.ExportDataTableCsv(ExportDataTableCsvRequest) returns (ExportDataTableCsvReply)`
- `ExportDataTableCsvRequest`:
  - `capture_id`
  - `filepath` (output CSV file)
  - `analyzers: repeated DataTableAnalyzerConfiguration`
    - `analyzer_id`
    - `radix_type`
  - `iso8601_timestamp`
  - `export_columns` (optional)
  - `filter` (`DataTableFilter` with `query` and `columns`)

## CLI surface

### Minimal command

- `salad export table --capture-id <id> --filepath /abs/out.csv --analyzer 123:hex`

### Flags

- `--analyzer <id>:<radix>` (repeatable)
  - radix: `hex|dec|bin|ascii` (map to `pb.RadixType`)
- `--iso8601-timestamp`
- `--columns time,data,address` (optional; if empty, export all)
- Filter:
  - `--filter-query "0xAA"`
  - `--filter-columns data,address` (optional; if empty, filter all columns)

## Implementation approach

### Parsing analyzer selectors

- Parse repeatable `--analyzer` into `[]pb.DataTableAnalyzerConfiguration`.
- Strict validation:
  - analyzer id non-zero
  - known radix

### Client wrapper

- Add `ExportDataTableCsv(...)` wrapper to `internal/saleae/client.go`.

### Files likely to change/add

- `cmd/salad/cmd/export_table.go` (new subcommand under `export`)
- `cmd/salad/cmd/util.go` (add helper to parse analyzer selectors + csv column lists)
- `internal/saleae/client.go` (RPC wrapper)

## Testing strategy

- Unit tests for flag parsing:
  - analyzer selectors, columns, filters
- Manual tests:
  - load capture → add analyzer(s) → export table → verify CSV exists and is non-empty

## Open questions / decisions

- Output handling: should we print `ok` only, or print the output filepath and row count (row count would require reading file)?
