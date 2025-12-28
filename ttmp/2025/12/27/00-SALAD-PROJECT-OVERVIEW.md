# SALAD Project Overview

**Date:** 2025-12-27  
**Status:** Active development  
**Project:** SALAD - Saleae Logic Analyzer CLI client in Go

## Executive Summary

SALAD is a Go-based CLI client for Saleae Logic 2's Automation API (gRPC). The project aims to provide scriptable, reproducible workflows for logic analyzer operations, enabling "logic debugging, but scripted" workflows.

**Current State:** Foundation complete, core capture/export commands implemented, advanced features planned.

## Project Structure

All tickets are organized under `salad/ttmp/2025/12/24/` with 9 active tickets covering different feature areas.

## Completed Work (Ticket 001-INITIAL-SALAD)

### ✅ Foundation Infrastructure

1. **Proto Vendoring & Code Generation**
   - Vendored `saleae.proto` from Saleae's Logic2 automation API
   - Generated Go gRPC bindings (`gen/saleae/automation/`)
   - Pinned upstream commit SHA for reproducibility

2. **Go Module Setup**
   - Initialized Go module with dependencies:
     - Cobra (CLI framework)
     - gRPC client libraries
     - zerolog (logging)
     - pkg/errors (error wrapping)

3. **Core Client Wrapper** (`internal/saleae/client.go`)
   - gRPC dial with configurable host/port/timeout
   - Context-aware RPC calls with proper error wrapping
   - Client connection management

### ✅ Implemented CLI Commands

1. **Discovery Commands**
   - `salad appinfo` - Get Logic 2 application info (version, API version, PID)
   - `salad devices` - List available devices (with simulation device filtering)

2. **Capture Management Commands**
   - `salad capture load` - Load a .sal capture file
   - `salad capture save` - Save capture to .sal file
   - `salad capture stop` - Stop an active capture
   - `salad capture wait` - Wait for capture completion
   - `salad capture close` - Close capture to release resources

3. **Export Commands (Raw Data)**
   - `salad export raw-csv` - Export raw digital/analog data to CSV
   - `salad export raw-binary` - Export raw data to binary format
   - Supports channel selection, analog downsampling, ISO8601 timestamps

### ✅ Documentation & Testing

- Build plan analysis document
- Feature brainstorm design doc (roadmap)
- Manual smoke test playbook
- Implementation diary tracking progress
- All validated against live Logic 2 automation server

## Planned Features (Tickets 002-009)

### 🔄 Ticket 002: Capture Start (Manual/Timed/Trigger)

**Status:** Analysis complete, implementation pending

**Goal:** Implement `salad capture start` with three capture modes:
- Manual mode (ring buffer, stop manually)
- Timed mode (duration-based)
- Digital trigger mode (trigger condition-based)

**Approach:** Config-file driven UX (YAML/JSON) to avoid flag explosion, with optional flag overrides.

**Key Files:**
- `cmd/salad/cmd/capture_start.go` (new)
- `internal/config/capture_start.go` (new config parsing)
- `internal/saleae/client.go` (add StartCapture wrapper)

### 🔄 Ticket 003: Analyzers (Add/Remove + Settings Templates)

**Status:** Analysis complete, implementation pending

**Goal:** Protocol decoding "as code" - add/remove analyzers with reproducible settings.

**Commands:**
- `salad analyzer add` - Add analyzer (SPI, I2C, Async Serial, etc.)
- `salad analyzer remove` - Remove analyzer
- `salad analyzer template list/show` - Template management

**Approach:** Settings via JSON/YAML files or typed flags (`--set`, `--set-bool`, `--set-int`). Curated templates in `configs/analyzers/`.

**Key Files:**
- `cmd/salad/cmd/analyzer.go` (new)
- `internal/config/analyzer_settings.go` (new)
- `configs/analyzers/` (templates)

### 🔄 Ticket 004: High-Level Analyzers (HLAs) Extensions Integration

**Status:** Analysis complete, implementation pending

**Goal:** Support Python extensions (HLAs) that post-process LLA output frames.

**Commands:**
- `salad hla add` - Add HLA from extension directory
- `salad hla remove` - Remove HLA

**Approach:** Uses `extension_directory` + `hla_name` + `input_analyzer_id` from proto.

**Key Files:**
- `cmd/salad/cmd/hla.go` (new)
- `internal/saleae/client.go` (add HLA wrapper methods)

### 🔄 Ticket 005: Export Data Tables (CSV) + Filtering

**Status:** Analysis complete, implementation pending

**Goal:** Export decoded analyzer output (data tables) with filtering and column selection.

**Command:**
- `salad export table` - Export decoded data tables

**Features:**
- Analyzer selection with radix per analyzer (hex, ascii, etc.)
- Column filtering (`--filter-query`, `--filter-columns`)
- Column selection (`--columns`)
- Future: SQLite/Parquet conversion

**Key Files:**
- `cmd/salad/cmd/export.go` (extend existing)
- `internal/saleae/client.go` (add ExportDataTableCsv wrapper)

### 🔄 Ticket 006: Pipelines (Run/Repro/Watch Workflows)

**Status:** Analysis complete, implementation pending

**Goal:** "One command" pipelines that chain multiple RPCs with a single config file.

**Commands:**
- `salad run` - Execute pipeline from config (device selection, capture, analyzers, exports)
- `salad repro` - Re-run last pipeline (for flaky hardware debugging)
- `salad watch` - Loop forever: capture → export → filter → exit on condition (CI-style)

**Approach:** Config-driven workflow orchestration, session manifest tracking.

**Key Files:**
- `cmd/salad/cmd/pipeline.go` (new)
- `internal/config/pipeline.go` (new)
- Session manifest management

### 🔄 Ticket 007: Sessions + JSON Output + Manifests

**Status:** Analysis complete, implementation pending

**Goal:** Session management and structured output for scriptability.

**Features:**
- Session directory (`.salad/<timestamp>/`) storing configs, outputs, IDs, tool version
- `--session` flag on all commands
- `--last`/`--current` capture ID shortcuts
- `--json` output mode for all commands
- Manifest files tracking inputs/outputs/timestamps

**Key Files:**
- `internal/session/` (new package)
- `cmd/salad/cmd/root.go` (extend global flags)
- JSON output formatters

### 🔄 Ticket 008: Doctor + Troubleshooting + gRPC Knobs

**Status:** Analysis complete, implementation pending

**Goal:** Debugging and observability for the CLI itself.

**Command:**
- `salad doctor` - Health checks (dial, GetAppInfo, devices, filesystem writable)

**Features:**
- `--grpc-trace` for client-side logging
- Separate `--dial-timeout` vs `--rpc-timeout`
- Comprehensive connectivity and environment checks

**Key Files:**
- `cmd/salad/cmd/doctor.go` (new)
- `internal/saleae/client.go` (extend with trace options)

### 🔄 Ticket 009: Saleae Extensions SDK Research

**Status:** Research complete

**Goal:** Understand how Saleae extensions/analyzers work for future integration.

**Findings:**
- Two extensibility layers:
  1. **Low-level analyzers (LLAs)**: C++ Protocol Analyzer SDK (native plugins)
  2. **Extensions (Python)**: HLAs and Measurements (post-processing)
- LLAs decode raw samples → frames
- HLAs process LLA output frames
- Extensions packaged with `extension.json` + Python files
- Logic 2 loads extensions from user-selected directories

**Documentation:** Comprehensive analysis document explaining both systems.

## Technical Architecture

### Package Structure

```
salad/
├── cmd/salad/          # Cobra CLI commands
│   ├── cmd/
│   │   ├── appinfo.go  ✅
│   │   ├── devices.go  ✅
│   │   ├── capture.go  ✅ (load/save/stop/wait/close)
│   │   ├── export.go   ✅ (raw-csv/raw-binary)
│   │   └── root.go     ✅
├── internal/saleae/    # gRPC client wrapper
│   ├── client.go       ✅ (core RPC wrappers)
│   └── config.go       ✅
├── gen/saleae/         # Generated protobuf bindings
│   └── automation/     ✅
├── proto/saleae/       # Vendored proto files
│   └── grpc/           ✅
└── ttmp/               # Documentation workspace
    └── 2025/12/24/     # All tickets
```

### Design Principles

1. **CLI-first ergonomics**: 1-3 commands for setup → capture → export → inspect
2. **Scriptable outputs**: Stable `--json` format, consistent exit codes
3. **Reproducible sessions**: Session directories with manifests
4. **Fast iteration**: Sensible defaults, sharp error messages
5. **Config-driven**: YAML/JSON configs to avoid flag explosion

## Current Implementation Status

### ✅ Fully Implemented
- Core gRPC client infrastructure
- Discovery commands (appinfo, devices)
- Capture lifecycle management (load/save/stop/wait/close)
- Raw data export (CSV and binary)

### 🔄 Planned (Analysis Complete)
- Capture start (manual/timed/trigger)
- Analyzer add/remove with templates
- High-level analyzer (HLA) support
- Data table export with filtering
- Pipeline workflows (run/repro/watch)
- Session management and JSON output
- Doctor/troubleshooting commands

## Key Design Decisions

1. **Proto Strategy**: Vendored + pinned proto with committed generated code (no protoc required for builds)
2. **Config-First UX**: YAML/JSON configs preferred over flag explosion
3. **Error Handling**: Consistent use of `pkg/errors` for wrapping
4. **Context Management**: All RPCs context-aware with timeouts
5. **Template System**: Curated analyzer templates (our convention, not Saleae's API)

## Next Steps (Recommended Priority)

1. **High ROI**: Implement `capture start` (ticket 002) - enables full capture workflows
2. **High ROI**: Implement `export table` (ticket 005) - powerful for debugging decoded data
3. **Ergonomics**: Add `--json` outputs (ticket 007) - makes it truly scriptable
4. **Power User**: Implement pipelines (ticket 006) - "one command" workflows
5. **Completeness**: Analyzers and HLAs (tickets 003, 004) - full protocol decode support

## Related Files

- **Build Plan**: `ttmp/2025/12/24/001-INITIAL-SALAD--initial-salad-saleae-logic-analyzer-client-in-go/analysis/01-build-plan-saleae-logic2-grpc-client-go.md`
- **Feature Roadmap**: `ttmp/2025/12/24/001-INITIAL-SALAD--initial-salad-saleae-logic-analyzer-client-in-go/design-doc/01-feature-brainstorm-powerful-saleae-logic2-cli-workflows.md`
- **Implementation Diary**: `ttmp/2025/12/24/001-INITIAL-SALAD--initial-salad-saleae-logic-analyzer-client-in-go/reference/01-diary.md`
- **Extensions Research**: `ttmp/2025/12/24/009-SALEAE-EXTENSIONS-SDK--saleae-extensions-analyzers-sdk-hlas-packaging-installation/analysis/01-how-saleae-analyzer-plugins-extensions-work-logic-2.md`

## Notes

- All tickets created on 2025-12-24
- All tickets currently marked "active" status
- Foundation work (ticket 001) is complete and validated
- Remaining tickets have analysis documents but implementation pending
- Project follows docmgr workflow for structured documentation

