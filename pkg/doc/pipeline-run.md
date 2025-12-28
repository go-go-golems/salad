---
Title: Run Pipelines with Salad
Slug: salad-pipeline-run
Short: Config-driven workflow runner (`salad run`) for reproducible “load capture → add analyzers → export artifacts” flows.
Topics:
- saleae
- grpc
- cli
- pipelines
- automation
- testing
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: GeneralTopic
---

# Run Pipelines with Salad

`salad run` is a config-driven “workflow runner” that chains multiple Logic 2 Automation gRPC operations into one reproducible command. It is designed for the loops you actually repeat in practice: load a capture, add analyzers with known-good settings, export raw and decoded data, and then clean up.

## What `salad run` does (and what it doesn’t)

Pipelines are only useful if they’re explicit about scope. The current `salad run` implementation is intentionally small and matches what the codebase can do today.

- **What it does**
  - **Load** a capture from an existing `.sal` file (`LoadCapture`)
  - **Add** one or more low-level analyzers (LLAs) (`AddAnalyzer`)
  - **Export** artifacts:
    - raw CSV (`ExportRawDataCsv`)
    - raw binary (`ExportRawDataBinary`)
    - decoded table CSV (`ExportDataTableCsv`)
  - **Close** the capture (best-effort) (`CloseCapture`)

- **What it does not do (yet)**
  - **Start captures** (`StartCapture` is not exposed as a CLI command yet; see ticket 002)
  - **HLAs / extensions** (no `AddHighLevelAnalyzer` support yet; see ticket 004)
  - **`repro` / `watch`** (those require a session/manifest story; see ticket 007)

## Quick start (mock server)

The fastest way to iterate on pipelines is to run against `salad-mock` with a deterministic scenario that writes placeholder artifacts. This tests your pipeline wiring without real hardware or a running Logic 2 instance.

### Use the ticket-local wrapper script (recommended)

From the `salad/` repo root:

```bash
./ttmp/2025/12/24/006-PIPELINES--pipelines-run-repro-watch-workflows/scripts/03-run-against-mock.sh
```

This wrapper script will:
- kill port collisions on `:10431`
- start `salad-mock` in tmux using the ticket’s scenario config
- run `salad run --config .../01-pipeline-mock.yaml`
- verify that `/tmp/salad-pipeline-006/...` artifacts exist and contain expected markers
- stop the tmux session

If you want the longer “manual steps” version and debugging tips, see:
- `ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/playbook/01-debugging-the-mock-server.md`

## Quick start (real Logic 2 server)

Running against the real server is the best “does it work for humans?” validation, but it requires a real `.sal` to load and a running Logic 2 instance with automation enabled.

### Prerequisites

- Logic 2 running with automation enabled (see the smoke test playbook):
  - `ttmp/2025/12/24/001-INITIAL-SALAD--initial-salad-saleae-logic-analyzer-client-in-go/playbook/01-manual-smoke-test-logic2-automation.md`
- A capture file to load (an absolute path to `*.sal` on the same machine)

### Run the ticket-local real-server script

From the `salad/` repo root:

```bash
SAL="/tmp/Session 6.sal" ./ttmp/2025/12/24/006-PIPELINES--pipelines-run-repro-watch-workflows/scripts/02-real-run.sh
```

This generates a temporary pipeline config at `/tmp/salad-pipeline-006-real.yaml` and executes `salad run` against `127.0.0.1:10430` by default.

## CLI reference

`salad run` is a top-level command (not under `export` / `capture`), and it follows the same connection flags as the rest of the CLI.

```bash
GOWORK=off go run ./cmd/salad --host 127.0.0.1 --port 10431 --timeout 10s \
  run --config path/to/pipeline.yaml
```

### Output

The output is intentionally simple and grep-friendly (ticket 007 will unify structured output):

- `capture_id=<id>`
- `analyzer_id=<id> label="<label>"`
- `artifact=<path>`
- `ok`

## Pipeline config format (v1)

Pipeline configs are YAML/JSON files with a version number. If omitted, `version` defaults to `1`.

**Important:** Relative file paths (like `configs/analyzers/spi.yaml`) are resolved relative to the process working directory. In practice, run `salad` from the repo root if you want to use repo-relative paths.

### Minimal example (YAML)

This is the smallest useful pipeline: load a capture and export a decoded data table.

```yaml
version: 1

capture:
  load:
    filepath: /abs/path/to/capture.sal

analyzers:
  - name: "SPI"
    label: "spi"
    settings_yaml: "configs/analyzers/spi.yaml"

exports:
  - type: table-csv
    filepath: /tmp/out/table.csv
    iso8601_timestamp: true
    analyzers:
      - ref: "spi"
        radix: hex

cleanup:
  close_capture: true
```

### Capture

`salad run` currently supports only capture loading:

- `capture.load.filepath` (**required**): absolute `.sal` path to load via `LoadCapture`

### Analyzers (LLA)

Each analyzer entry calls `AddAnalyzer` with the given settings.

- `analyzers[].name` (**required**): Logic 2 analyzer UI name (e.g. `"SPI"`, `"I2C"`, `"Async Serial"`)
- `analyzers[].label` (optional): human-facing label; also becomes the default reference key
- Settings file (optional; only one allowed):
  - `analyzers[].settings_yaml`
  - `analyzers[].settings_json`
- Typed overrides (optional; applied after file, so they win):
  - `analyzers[].set` (string `key=value`)
  - `analyzers[].set_bool` (bool)
  - `analyzers[].set_int` (int)
  - `analyzers[].set_float` (float)

### Exports

Exports run after analyzers are created.

#### `raw-csv`

- `type: raw-csv`
- `directory` (**required**)
- `digital` and/or `analog` (at least one required)
- `analog_downsample_ratio` (optional; default 1)
- `iso8601_timestamp` (optional)

#### `raw-binary`

- `type: raw-binary`
- `directory` (**required**)
- `digital` and/or `analog` (at least one required)
- `analog_downsample_ratio` (optional; default 1)

#### `table-csv`

- `type: table-csv`
- `filepath` (**required**)
- `analyzers` (**required**): list of `{ref, radix}` entries
  - `ref`: analyzer reference string (currently the analyzer `label`, or `name` if label is empty)
  - `radix`: one of `hex|dec|bin|ascii`
- `columns` (optional): export column selection
- `filter` (optional):
  - `query` (required if filter is present)
  - `columns` (optional)
- `iso8601_timestamp` (optional)

### Cleanup

- `cleanup.close_capture` (optional; default `true`): close capture best-effort after the run

## Implementation pointers (for developers)

This page describes the *user-facing* contract. If you’re implementing or extending pipelines, start here:

- **CLI entry point**: `cmd/salad/cmd/run.go` (`runCmd`)
- **Config loader**: `internal/pipeline/config.go` (`pipeline.Load`)
- **Runner**: `internal/pipeline/runner.go` (`(*pipeline.Runner).Run`)

For testing and debugging:

- **Mock server guide**: `salad/pkg/doc/mock-server-user-guide.md`
- **Mock server debugging playbook + tmux scripts**:
  - `ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/playbook/01-debugging-the-mock-server.md`


