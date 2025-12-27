# Mock Saleae Server User Guide

## Overview

The mock Saleae server (`salad-mock`) is a gRPC server that emulates the Logic 2 Automation API.
It is configured using scenario YAML files so you can test `salad` commands without running Logic 2
or plugging in physical hardware.

## Prerequisites

- Go toolchain (use `go run ./cmd/salad-mock`).
- A mock configuration YAML file (see `configs/mock/`).

## Quick start

1. Start the mock server with a scenario file:

   ```bash
   go run ./cmd/salad-mock --config configs/mock/happy-path.yaml --port 10431
   ```

2. Point the `salad` CLI at the mock:

   ```bash
   go run ./cmd/salad --host 127.0.0.1 --port 10431 appinfo
   go run ./cmd/salad --host 127.0.0.1 --port 10431 devices
   ```

3. Try a capture flow:

   ```bash
   go run ./cmd/salad --host 127.0.0.1 --port 10431 capture load --filepath /tmp/mock.sal
   go run ./cmd/salad --host 127.0.0.1 --port 10431 capture save --capture-id 1 --filepath /tmp/saved.sal
   go run ./cmd/salad --host 127.0.0.1 --port 10431 capture close --capture-id 1
   ```

## Scenario structure (high level)

- `fixtures`: initial app info, devices, captures.
- `behavior`: per-RPC defaults (timing policies, export side effects, validation).
- `faults`: deterministic error injection rules.

Review `configs/mock/happy-path.yaml` and `configs/mock/faults.yaml` for concrete examples.

## Common workflows

### Test export file placeholders

Enable placeholder writes in the scenario, then point the CLI at a temp directory:

```bash
go run ./cmd/salad --host 127.0.0.1 --port 10431 export raw-csv --capture-id 1 --directory /tmp/mock-export --digital 0,1
```

### Inject failures

Use `faults` blocks to simulate transient failures. Example: `configs/mock/faults.yaml`
causes the first `SaveCapture` call to return `UNAVAILABLE`.

## Troubleshooting

- **"capture not found"**: Load or seed a capture in fixtures before calling save/stop/wait/close/export.
- **"file does not exist"**: Disable `require_file_exists` in `LoadCapture` behavior for tests that
  should not touch the filesystem.
- **Port conflicts**: Change `--port` if `10431` is in use.
