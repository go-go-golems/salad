---
Title: 'Build plan: Saleae Logic2 gRPC client (Go)'
Ticket: 001-INITIAL-SALAD
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
Summary: "Plan for implementing a Go (Cobra) CLI that talks to Saleae Logic2 Automation API via gRPC, including proto vendoring/generation strategy and initial command set."
LastUpdated: 2025-12-24T21:13:18.649721542-05:00
WhatFor: ""
WhenToUse: ""
---

# Build plan: Saleae Logic2 gRPC client (Go)

## Goals

- **Primary**: a Go CLI that can connect to a running Logic 2 instance (automation enabled) and call core RPCs.
- **Short-term deliverable**: `salad appinfo` (GetAppInfo) working end-to-end.
- **Medium-term**: capture lifecycle (start/stop/save/load), analyzer operations, export operations.

## Non-goals (for now)

- Running Logic 2 itself (we only connect to it).
- A full-featured SDK reimplementation (we’ll wrap only what we need, incrementally).

## Assumptions

- Logic 2 is running with automation enabled (likely local):
  - `Logic-2.AppImage --automation --automationPort 10430`
- gRPC is currently expected to be plaintext/insecure on localhost (per community examples); we’ll keep transport options configurable.

## Architecture overview

### Layers

- **CLI layer (cobra)**: command parsing, flags, output formatting (human + future `--json`).
- **Client wrapper** (`internal/saleae`): dial, timeouts, retry policy (minimal), error wrapping (`github.com/pkg/errors`).
- **Generated gRPC bindings** (`gen/...`): generated from Saleae’s `saleae.proto` and committed so builds don’t require `protoc`.

### Package layout (proposed)

```
salad/
  go.mod
  cmd/salad/
    main.go
    cmd/
      root.go
      appinfo.go
  internal/saleae/
    client.go
    dial.go
  proto/
    saleae/grpc/saleae.proto   # vendored + pinned
  gen/
    saleae/grpc/...            # generated + committed
  tools.go                     # pins protoc plugins
```

Notes:
- We keep `proto/` and `gen/` at repo root so the generated import path is stable.
- We’ll keep `cmd/XXX/` intact (template), but the supported binary will be `cmd/salad`.

## Proto + generation strategy

- **Source of truth**: `saleae/logic2-automation` (`proto/saleae/grpc/saleae.proto`).
- **Pinning**: record the upstream commit SHA in:
  - `proto/saleae/grpc/README.md` (or comment header in the proto copy), and
  - the ticket changelog (so we can track upgrades).
- **Generation**:
  - Prefer `protoc` + `protoc-gen-go` + `protoc-gen-go-grpc`.
  - Commit generated code in `gen/` so `go build` works without requiring `protoc` in CI/user machines.
  - Add `go:generate` (or Makefile target) to regenerate deterministically.

Potential sharp edge:
- If `saleae.proto` lacks `option go_package`, we’ll either:
  - use `--go_opt=M...` mapping flags, or
  - maintain a tiny local patch (avoid if possible; prefer mapping).

## CLI shape

### Global flags

- `--host` (default `127.0.0.1`)
- `--port` (default `10430`)
- `--timeout` (default `5s`)
- `--log-level` (default `info`)

### First commands

- `salad appinfo`:
  - dials gRPC
  - calls `GetAppInfo`
  - prints: application version, API version, PID

### Next commands (skeleton-first)

- `salad capture start ...`
- `salad capture stop --id ...`
- `salad capture save --id ... --out capture.sal`
- `salad capture load --file capture.sal`
- `salad analyzer add ...`
- `salad export table ...`
- `salad export raw-csv ...`

## Testing strategy

- **Compilation gate**: `go build ./...`
- **Manual smoke**: start Logic 2 with automation enabled and run:
  - `go run ./cmd/salad appinfo --host 127.0.0.1 --port 10430`

## Risks / review focus

- Proto evolution (beta API): pinning + easy regeneration must be solid.
- gRPC transport assumptions (insecure vs TLS): keep dial options centralized.
- Timeouts and context cancellation: ensure every RPC call is context-bound.
