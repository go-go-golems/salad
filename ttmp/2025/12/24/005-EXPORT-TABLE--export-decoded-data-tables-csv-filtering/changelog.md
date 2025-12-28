# Changelog

## 2025-12-24

- Initial workspace created


## 2025-12-24

Added initial implementation analysis doc (proto mapping, CLI UX, files/tests).


## 2025-12-28

- Implemented `salad export table` (Saleae `ExportDataTableCsv`) including client wrapper, mock-server support, and a CLI integration test (commit `8a72a88322ef121f15d341ab39e3481ec30a3098`).
- Unblocked `go test ./...` by splitting multi-`main()` Go scripts under ticket 003 into per-script directories (commit `78d72c97cc8f09fba2f8d98fce45cd03d7cd3218`).


## 2025-12-27

Closed: implemented export table (ExportDataTableCsv) + tests; docs updated; real-server validation script added.

