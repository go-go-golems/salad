# Diary

## Goal

Implement `salad export table` (Saleae `ExportDataTableCsv`) so we can export decoded analyzer frames into a single CSV file with analyzer selection, per-analyzer radix, optional column selection, and optional filtering.

## Step 1: Re-orient on ticket state + locate existing code paths

This step re-established what exists today vs what was only planned in the early analysis doc. The key outcome is that the CLI already has an `export` subtree (`raw-csv` / `raw-binary`), but **no `export table` command yet**, and the only `ExportDataTableCsv` occurrences are proto/generated bindings and docs.

That tells us the next concrete work is to add:
1) a cobra command under `export`,
2) a `saleae.Client` wrapper for the gRPC call, and
3) mock-server support so we can integration-test the CLI against the repo’s mock Saleae server.

### What I did
- Read ticket docs: `tasks.md` and the initial analysis doc.
- Grepped the repo for `ExportDataTableCsv` and `export table`.
- Located the existing cobra export subtree implementation (`raw-csv`, `raw-binary`).
- Located the mock server’s existing export implementations (raw csv/binary).

### Why
- Avoid re-implementing something already present.
- Ensure we hook into existing CLI patterns (cobra, shared `--host/--port/--timeout` flags).
- Ensure we can validate behavior in CI using the mock server.

### What worked
- Quick grep confirmed `ExportDataTableCsv` is not implemented yet in CLI/client (only proto bindings).
- The mock server already has a clean pattern for “export RPC + optional file side-effects”, which we can extend.

### What didn't work
- N/A (no implementation attempted in this step).

### What I learned
- The codebase already has a strong scaffold for export verbs; implementing `export table` should mirror `raw-csv` closely.
- The mock server is plan-driven, so adding a new RPC likely means touching `exec.go`, `plan.go`, `config.go`, `server.go`, and `side_effects.go`.

### What was tricky to build
- N/A (no code changes yet).

### What warrants a second pair of eyes
- N/A (no code changes yet).

### What should be done in the future
- N/A.

### Code review instructions
- Start in `salad/cmd/salad/cmd/export.go` for existing export patterns.
- Then inspect `salad/internal/saleae/client.go` for the RPC wrapper style.
- Finally, review `salad/internal/mock/saleae/` for how export RPCs are modeled + tested.

## Step 2: Add CLI surface + client wrapper for ExportDataTableCsv

This step wires the “real” code path for ticket 005: a new `salad export table` cobra command that maps flags into `ExportDataTableCsvRequest`, plus a `saleae.Client.ExportDataTableCsv` wrapper to centralize request validation and the gRPC call.

The main correctness risks here are argument parsing (especially the repeatable `--analyzer <id>:<radix>` selectors) and ensuring we don’t silently send malformed requests (empty filepath, missing analyzers, unspecified radix).

### What I did
- Added `salad export table` command with flags:
  - `--capture-id`
  - `--filepath`
  - repeatable `--analyzer <id>:<radix>` (radix: `hex|dec|bin|ascii`)
  - `--iso8601-timestamp`
  - `--columns` (comma-separated)
  - `--filter-query`, `--filter-columns`
- Added `saleae.Client.ExportDataTableCsv(...)` wrapper that validates inputs and calls `Manager.ExportDataTableCsv`.
- Added a small shared CSV parser helper for string columns.

### Why
- Keep CLI flag parsing localized to the command, but keep core gRPC request validation inside the client wrapper (consistent with existing `ExportRawDataCsv/Binary`).
- Provide a predictable CLI UX aligned with existing commands (required flags, prints `ok`).

### What worked
- The command structure matches existing `export raw-*` patterns closely, so it should be easy to review and consistent to use.

### What didn't work
- N/A (compile/test not run yet in this step).

### What I learned
- Cobra’s `StringArrayVar` is a nice fit for repeatable `--analyzer` selectors; we still need explicit validation because “required” on arrays doesn’t prevent empty strings.

### What was tricky to build
- Balancing strict validation with a convenient UX:
  - `--filter-columns` without `--filter-query` is treated as an error to avoid sending confusing filter requests.
  - `radix_type` is required (we reject unspecified) to prevent relying on Logic 2 defaults.

### What warrants a second pair of eyes
- Confirm the chosen error messages are actionable and consistent with other commands.
- Sanity-check the radix mapping (`hex/dec/bin/ascii`) against `saleae.proto` `RadixType` values.

### What should be done in the future
- Add richer UX sugar if desired (not required for correctness):
  - accept `RADIX_TYPE_HEXADECIMAL`-style values in addition to `hex`, etc.

### Code review instructions
- `salad/cmd/salad/cmd/export_table.go`: flag surface + request mapping
- `salad/cmd/salad/cmd/export.go`: command registration under `export`
- `salad/cmd/salad/cmd/util.go`: `parseStringCSV`
- `salad/internal/saleae/client.go`: `ExportDataTableCsv` wrapper + validation

## Step 3: Extend mock server to support ExportDataTableCsv

This step makes ticket 005 testable in-repo by extending the mock Saleae server with the `ExportDataTableCsv` RPC and optional file-writing side effects. That keeps the CLI integration tests hermetic: we can assert that the CLI makes the correct RPC call shape and that an output CSV file is created when configured.

### What I did
- Added `MethodExportDataTableCsv` to the mock server method registry.
- Added mock config + plan knobs:
  - validation: `require_capture_exists`
  - side effects: `write_placeholder_file`, `include_request_in_file`
- Implemented `Server.ExportDataTableCsv` that (optionally) writes a placeholder CSV file.
- Extended the mock server side effects interface to support `ExportDataTableCSV(...)`.

### Why
- `salad` CLI integration tests already use the mock server; adding `export table` there avoids relying on a real Logic 2 instance in CI.
- The placeholder file content lets us validate that filter/analyzer settings made it into the request (via `include_request_in_file`).

### What worked
- The existing mock server architecture already had a clean “export RPC + file side effects” pattern for raw exports, so this was mostly additive and consistent.

### What didn't work
- N/A (tests/compile not run yet in this step).

### What was tricky to build
- Plumbing a new RPC through all the mock server layers (method registry, config, plan compilation, server handler, side effects) without breaking existing behavior.

### What warrants a second pair of eyes
- Confirm the new YAML keys for `ExportDataTableCsv` align with the conventions used by existing export behaviors (`ExportRawDataCsv`, `ExportRawDataBinary`).

### Code review instructions
- `salad/internal/mock/saleae/exec.go`: add method + file-side-effect detection
- `salad/internal/mock/saleae/config.go`: YAML surface for new behavior
- `salad/internal/mock/saleae/plan.go`: compile behavior + fault matcher support
- `salad/internal/mock/saleae/server.go`: RPC handler implementation
- `salad/internal/mock/saleae/side_effects.go`: placeholder writer

## Step 4: Add CLI integration test for `salad export table` (mock server)

This step adds a real end-to-end-ish test that exercises the cobra command, the internal client wrapper, and the mock server RPC handler together. The intent is to lock the CLI UX and ensure we don’t regress request shape or accidentally drop required flags.

**Commit (code):** 8a72a88322ef121f15d341ab39e3481ec30a3098 — "feat(export): add export table (ExportDataTableCsv)"

### What I did
- Added `TestCLI_ExportTable_AgainstMockServer`:
  - starts the mock server using `configs/mock/happy-path.yaml`
  - enables placeholder-file writing for `ExportDataTableCsv` in the loaded config
  - runs `salad capture load` → `salad analyzer add` → `salad export table`
  - asserts the output file exists and contains mock markers (including filter markers)

### Why
- This catches wiring bugs that unit tests won’t: command registration, flag parsing, client wrapper invocation, and mock RPC handler plumbing.

### What worked
- `cd salad && go test ./... -count=1` passed, including `TestCLI_ExportTable_AgainstMockServer`.

### What didn't work
- N/A.

### What warrants a second pair of eyes
- Confirm the test is stable and not overly coupled to placeholder formatting.

### Code review instructions
- `salad/internal/mock/saleae/export_table_cli_test.go`

## Step 5: Get repo back to “go test ./...” green (ttmp scripts packaging fix)

While validating ticket 005 with `go test ./...`, the build failed in an unrelated package under `ttmp/.../003-.../scripts` because multiple standalone Go scripts (each with `main()`) lived in the same directory. `go test` builds packages at the directory level, so this created a compile-time “main redeclared” conflict.

To unblock “compile + commit” for 005, I reorganized those scripts so each lives in its own subdirectory (still runnable via `go run`), and the parent `scripts/` directory no longer contains multiple `main()` functions.

**Commit (code):** 78d72c97cc8f09fba2f8d98fce45cd03d7cd3218 — "ttmp: split analyzer scripts into per-script dirs"

### What I did
- Observed `go test ./...` failing with `main redeclared` errors in:
  - `ttmp/2025/12/24/003-.../scripts/*.go`
- Moved these scripts into per-script directories:
  - `03-compare-meta-json-to-template.go` → `scripts/03-compare-meta-json-to-template/main.go`
  - `05-real-validate-session6-templates.go` → `scripts/05-real-validate-session6-templates/main.go`
  - `06-real-validate-template-variations.go` → `scripts/06-real-validate-template-variations/main.go`
- Re-ran `go test ./...` successfully.

### Why
- The project’s `Makefile` and `AGENT.md` expect `go test ./...` to work. Ticket 005 shouldn’t land in a state that leaves the repo non-compiling.

### What worked
- `go test ./... -count=1` is green again after the move.

### What didn't work
- `go test ./...` initially failed due to multi-`main()` scripts in one directory.

### What warrants a second pair of eyes
- Confirm the new “script directory per tool” layout is the preferred pattern for future `.go` scripts in `ttmp/.../scripts/`.


