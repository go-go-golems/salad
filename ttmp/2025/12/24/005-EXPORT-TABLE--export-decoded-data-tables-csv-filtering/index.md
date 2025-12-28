---
Title: Export decoded data tables (CSV) + filtering
Ticket: 005-EXPORT-TABLE
Status: active
Topics:
    - go
    - saleae
    - logic-analyzer
    - client
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/salad/cmd/export.go
      Note: Existing export subtree to extend
    - Path: cmd/salad/cmd/export_table.go
      Note: `salad export table` command implementation
    - Path: internal/saleae/client.go
      Note: `Client.ExportDataTableCsv` wrapper
    - Path: internal/mock/saleae/server.go
      Note: Mock server support for `ExportDataTableCsv`
    - Path: internal/mock/saleae/export_table_cli_test.go
      Note: CLI integration test for `salad export table`
    - Path: proto/saleae/grpc/saleae.proto
      Note: ExportDataTableCsv request shape
ExternalSources: []
Summary: "Implements `salad export table` (ExportDataTableCsv) for exporting decoded analyzer frames to CSV with filtering."
LastUpdated: 2025-12-28T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Export decoded data tables (CSV) + filtering

Document workspace for 005-EXPORT-TABLE.

## Key docs

- `analysis/01-implementation-export-data-tables-csv-filtering.md`
- `reference/01-diary.md`
