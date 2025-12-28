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
RelatedFiles:
    - Path: cmd/salad/cmd/export_table.go
      Note: '`salad export table` cobra command (flag parsing + request mapping)'
    - Path: cmd/salad/cmd/export.go
      Note: Registers `export table` under the export subtree
    - Path: cmd/salad/cmd/util.go
      Note: CSV parsing helper used by `export table` (`parseStringCSV`)
    - Path: internal/saleae/client.go
      Note: '`Client.ExportDataTableCsv` wrapper + request validation'
    - Path: internal/mock/saleae/server.go
      Note: Mock server `ExportDataTableCsv` implementation (optional placeholder file write)
    - Path: internal/mock/saleae/export_table_cli_test.go
      Note: CLI integration test for `salad export table` against mock server
    - Path: proto/saleae/grpc/saleae.proto
      Note: Upstream proto definition for request/filter/radix
ExternalSources: []
Summary: "Implemented `salad export table` (ExportDataTableCsv): multi-analyzer export with per-analyzer radix, optional column selection, optional filtering, and ISO8601 timestamps."
LastUpdated: 2025-12-28T00:00:00Z
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

- `salad export table --capture-id <id> --filepath /abs/out.csv --analyzer 10025:hex`

### Flags (as implemented)

- **Required**
  - `--capture-id <id>`
  - `--filepath </abs/out.csv>`
  - `--analyzer <id>:<radix>` (repeatable)
    - radix: `hex|dec|bin|ascii`

- **Optional**
  - `--iso8601-timestamp`
  - `--columns time,data,address`  
    If empty, `export_columns` is left empty and Logic 2 exports all columns.
  - `--filter-query "0xAA"`
  - `--filter-columns data,address`  
    If empty, Logic 2 searches all columns. If provided, **requires** `--filter-query`.

### Output

- On success prints `ok` (consistent with existing `export raw-*` commands).

## Implementation details (as shipped)

### Request mapping

- `salad export table` maps to `pb.ExportDataTableCsvRequest`:
  - `capture_id` ← `--capture-id`
  - `filepath` ← `--filepath`
  - `analyzers` ← parsed from repeatable `--analyzer <id>:<radix>` into `[]*pb.DataTableAnalyzerConfiguration`
  - `iso8601_timestamp` ← `--iso8601-timestamp`
  - `export_columns` ← parsed from `--columns` (CSV list)
  - `filter` ← optional `DataTableFilter{query, columns}` derived from `--filter-*`

### Strict validation

- CLI validates:
  - at least one `--analyzer` entry
  - each analyzer selector matches `<id>:<radix>`
  - analyzer id is non-zero
  - radix is known (`hex|dec|bin|ascii`)
  - `--filter-columns` implies `--filter-query`

- Client validates again (defense-in-depth):
  - `capture_id != 0`
  - non-empty `filepath`
  - non-empty analyzers list, no nil entries
  - `radix_type != RADIX_TYPE_UNSPECIFIED`

### CSV list parsing

- `--columns` and `--filter-columns` are parsed as comma-separated lists:
  - trim whitespace
  - drop empty items

## Testing

### Mock-server integration test

- `internal/mock/saleae/export_table_cli_test.go` runs:
  - `salad capture load` → `salad analyzer add` → `salad export table`
  - against the repo’s mock server, with placeholder file writing enabled for `ExportDataTableCsv`
  - and asserts the output CSV file exists and contains expected markers (including filter markers)

### Command to run locally

- `cd salad && go test ./... -count=1`

## Decisions

- **Output**: prints `ok` only (no row count).
