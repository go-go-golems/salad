---
Title: 'Feature brainstorm: powerful Saleae (Logic2) CLI workflows'
Ticket: 001-INITIAL-SALAD
Status: active
Topics:
    - go
    - saleae
    - logic-analyzer
    - client
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/salad/cmd
      Note: Existing CLI commands that we can extend into pipelines
    - Path: gen/saleae/automation/saleae.pb.go
      Note: Generated types referenced when designing CLI UX
    - Path: proto/saleae/grpc/saleae.proto
      Note: Defines the available RPC surface area and message types for brainstorming
ExternalSources: []
Summary: Brainstormed feature set for a power-user, CLI-first Saleae Logic2 debugging tool, grounded in the Automation API proto (RPCs + message types).
LastUpdated: 2025-12-24T22:29:20.984629228-05:00
WhatFor: ""
WhenToUse: ""
---


# Feature brainstorm: powerful Saleae (Logic2) CLI workflows

This document is a “menu” of CLI features and workflows we can build on top of the Saleae Logic2 Automation API. It is grounded in the current `saleae.proto` surface area (RPCs + message types) and calls out where we’d need post-processing or additional conventions.

## Automation API surface area (from `saleae.proto`)

### RPCs available

- **App + device discovery**
  - `GetAppInfo`
  - `GetDevices`
- **Capture lifecycle**
  - `StartCapture`, `StopCapture`, `WaitCapture`
  - `LoadCapture`, `SaveCapture`, `CloseCapture`
- **Analyzers**
  - `AddAnalyzer`, `RemoveAnalyzer`
  - `AddHighLevelAnalyzer`, `RemoveHighLevelAnalyzer`
- **Exports**
  - `ExportRawDataCsv`, `ExportRawDataBinary`
  - `ExportDataTableCsv`
  - `LegacyExportAnalyzer`

### Key configuration/types that matter for CLI UX

- **Device + channels**: `Device`, `DeviceType`, `LogicChannels`
- **Device configuration**: `LogicDeviceConfiguration` (sample rates, threshold volts, glitch filters)
- **Capture modes**:
  - `ManualCaptureMode` (ring buffer semantics, optional trim)
  - `TimedCaptureMode` (duration + optional trim)
  - `DigitalTriggerCaptureMode` (trigger type + pulse width bounds + linked channel conditions)
- **Analyzer settings**:
  - `AddAnalyzerRequest.settings: map<string, AnalyzerSettingValue>` (string/int64/bool/double)
  - `AddHighLevelAnalyzerRequest.settings: map<string, HighLevelAnalyzerSettingValue>` (string/number)
- **Data table export**:
  - `ExportDataTableCsvRequest` supports analyzers list, radix per analyzer, timestamp format, column selection, and filtering (`DataTableFilter`).

## CLI product vision: “logic debugging, but scripted”

### Guiding principles

- **CLI-first ergonomics** for debug loops: 1–3 commands to go from “setup” → “capture” → “export” → “inspect”.
- **Scriptable outputs**: stable `--json` (or `--format json`) for command outputs and consistent exit codes.
- **Reproducible sessions**: a “session directory” that stores configs, outputs, and a manifest (inputs, capture IDs, device IDs, timestamps).
- **Fast iteration**: default sensible values (host/port/timeout) and sharp errors when required fields are missing.

## Feature brainstorm (grouped by workflow)

### 1) Discovery / “am I connected?”

- **`salad appinfo`**: already implemented; can expand:
  - `--json`
  - `--wait-until-ready` (retry dial for N seconds)
- **`salad devices`**: already implemented; expansions:
  - filter by `--type LOGIC_PRO_8`
  - `--pick-first-physical` output only the first non-sim device (common scripting need)
  - `--smoke` command that checks connectivity and prints a short summary

### 2) Capture start (the big “power” command)

Based on `StartCaptureRequest` + `LogicDeviceConfiguration` + `CaptureConfiguration`.

- **`salad capture start`** modes:
  - `--mode manual|timed|digital-trigger`
  - `--buffer-mb 128`
  - `--channels digital=0,1,2 analog=0,1`
  - `--digital-sample-rate 10000000`
  - `--analog-sample-rate 1000000`
  - `--threshold-volts 1.8`
  - `--glitch-filter 0=50ns,1=100ns` (maps to `GlitchFilterEntry`)
- **Trigger options** for `DigitalTriggerCaptureMode`:
  - `--trigger-channel 0`
  - `--trigger rising|falling|pulse-high|pulse-low`
  - `--after-trigger 0.250s`
  - `--pulse-min 10ns --pulse-max 200ns` (only for pulse triggers)
  - `--linked 1=high,2=low` (maps to `DigitalTriggerLinkedChannel`)
- **Capture completion UX**:
  - `--wait` (calls `WaitCapture` automatically for non-manual modes)
  - `--timeout` for wait
  - print `capture_id=...` always (and optionally write it to a file)

### 3) Capture management / file-centric workflows

Based on `LoadCapture`, `SaveCapture`, `CloseCapture`, `StopCapture`, `WaitCapture`.

- **File-based command chains**:
  - `salad capture load --filepath ...` → prints capture ID
  - `salad capture save --capture-id ... --filepath ...`
  - `salad capture close --capture-id ...`
- **Quality-of-life**:
  - `salad capture save --capture-id ... --dir ./captures --name spi-boot --timestamp` (client builds filepath)
  - `salad capture close --all` (needs tracking of capture IDs we created in a session manifest)
  - `salad capture stop --last` (session-based)

### 4) Analyzer workflows (protocol decode “as code”)

Based on `AddAnalyzer` / `RemoveAnalyzer` and typed settings map.

- **`salad analyzer add`**
  - `--capture-id`
  - `--name "SPI" --label "boot spi"`
  - `--set "MOSI=0" --set "MISO=1" --set "Clock=2" ...`
  - `--settings-json settings.json` (best for complex analyzers)
  - output `analyzer_id=...`
- **`salad analyzer remove`**
  - `--capture-id --analyzer-id`
- **Settings discovery helpers (CLI-driven introspection)**
  - Not directly supported by proto, but we can add conventions:
    - “known analyzer templates” bundled in repo (`configs/analyzers/spi.yaml`, etc.)
    - a `salad analyzer template list` and `salad analyzer template show spi`

### 5) High-level analyzers (HLAs / extensions)

Based on `AddHighLevelAnalyzerRequest`.

- **`salad hla add`**
  - `--capture-id`
  - `--extension-dir /abs/path/to/extension`
  - `--name ... --label ...`
  - `--input-analyzer-id ...`
  - `--set key=value` / `--settings-json`
- **Workflows**
  - “Add base analyzer → add HLA → export table” in one command (`salad pipeline run ...`)

### 6) Export workflows (raw + decoded)

Based on `ExportRawDataCsv`, `ExportRawDataBinary`, `ExportDataTableCsv`, `LegacyExportAnalyzer`.

- **Raw data exports**
  - `salad export raw-csv --capture-id ... --directory ... --digital 0,1,2 [--analog 0,1]`
  - `salad export raw-binary ...`
  - `--analog-downsample-ratio` and `--iso8601-timestamp` (csv)
  - “export directory conventions”:
    - create per-run dir with `manifest.json` and per-channel files
- **Data table exports (decoded analyzer outputs)**
  - `salad export table --capture-id ... --filepath out.csv`
  - select analyzers + radix:
    - `--analyzer 123:hex --analyzer 124:ascii`
  - filtering:
    - `--filter-query "0xAA"` + `--filter-columns data,address`
  - column selection:
    - `--columns time,mosi,miso`
- **Power extras (post-processing)**
  - `salad export table --to sqlite` / `--to parquet` (convert after CSV export)
  - `salad export raw-binary --to sigrok` (converter tooling)

### 7) “One command” pipelines (superpower for debugging)

These are wrappers that chain multiple RPCs and exports with a single config file.

- **`salad run`** (pipeline runner)
  - config includes: device selection, channels, capture mode, analyzers, exports
  - emits: capture ID + output artifact paths
- **`salad repro`**
  - re-run the last pipeline with the same config (for flaky hardware/debugging)
- **`salad watch`**
  - loop forever: capture → export → grep/filter → exit non-zero if condition is met (CI-style hardware checks)

### 8) Session management + ergonomics

- **Session directory** (e.g. `./.salad/<timestamp>/`)
  - stores configs + output paths + IDs + tool version + Logic2 appinfo
- **`--session` flag** on all commands
  - persist capture IDs and allow `--last`/`--current`
- **`--dry-run`** for pipeline commands (prints planned RPC calls, no execution)

### 9) Output modes and UX for power users

- **`--json`** for all commands
- **`--quiet`** (only IDs / critical values)
- **`--no-color`** / consistent machine-readable lines
- **Exit codes as signals**
  - e.g. `watch` returns 0/1 depending on match condition

### 10) Debugging + observability features for the CLI itself

- **gRPC troubleshooting flags**
  - `--grpc-trace` (client-side logging hooks)
  - `--dial-timeout` vs `--rpc-timeout`
- **`salad doctor`**
  - checks: can dial, GetAppInfo works, device list non-empty, filesystem writable for exports, etc.

## What we can’t do purely from this proto (notes)

- **Analyzer “schema discovery”** (list analyzers and their setting keys/options) isn’t exposed here; we’d rely on:
  - templates we curate
  - docs from Saleae
  - user-supplied settings JSON
- **UI-level operations** (changing UI views, zooming, selecting markers, etc.) aren’t part of this API.

## Suggested near-term roadmap (high ROI)

- Implement `capture start` (manual + timed + trigger) with a config-file input (YAML/JSON) so we don’t explode flags.
- Implement `export table` with analyzer selection and filtering (very powerful for debugging).
- Add `--json` outputs + a session manifest to make this truly scriptable.
