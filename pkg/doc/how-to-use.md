---
Title: How to Use Salad (Captures + Analyzers)
Slug: salad-how-to-use
Short: Practical guide to connecting to Logic 2 Automation, working with captures, and adding analyzers reproducibly.
Topics:
- saleae
- grpc
- cli
- captures
- analyzers
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: GeneralTopic
---

# How to Use Salad (Captures + Analyzers)

`salad` is a small CLI for talking to the Saleae Logic 2 Automation gRPC API. The goal is to make common workflows reproducible and scriptable: load or manipulate captures, export data, and add/remove analyzers using settings files instead of click-paths.

## Prerequisites

To use `salad` effectively you need a running gRPC endpoint (real Logic 2 or the mock server), and you need to know which capture you want to operate on.

- **Go toolchain**: most examples use `go run` directly.
- **A running Logic 2 instance** (for real workflows) or `salad-mock` (for deterministic tests).
- **Host/port access**:
  - Real Logic 2 default port is typically `10430`.
  - Mock server default port is `10431`.

## Connect to a server (sanity checks)

Before doing anything else, verify that you can connect and that you’re talking to the right instance.

### Check app info

```bash
go run ./cmd/salad --host 127.0.0.1 --port 10430 --timeout 5s appinfo
```

### List devices

```bash
go run ./cmd/salad --host 127.0.0.1 --port 10430 --timeout 5s devices
```

If these work, your connectivity and gRPC negotiation is healthy.

## Captures (load/save/stop/wait/close)

A “capture” is the unit that almost everything else attaches to (exports, analyzers, HLAs). In practice, you’ll either **load** an existing `.sal` file or **work with a capture already open in the Logic 2 UI**.

### Load a `.sal` capture file

Loading a capture is the simplest way to get a `capture_id` from the CLI.

```bash
go run ./cmd/salad --host 127.0.0.1 --port 10430 --timeout 30s capture load \
  --filepath /abs/path/to/capture.sal
```

Expected output:

```text
capture_id=<id>
```

### Save a capture

```bash
go run ./cmd/salad --host 127.0.0.1 --port 10430 --timeout 30s capture save \
  --capture-id <id> \
  --filepath /abs/path/to/output.sal
```

### Stop vs Close (important)

These are different operations, and mixing them up is a common source of “where did my stuff go?” confusion.

- **Stop** ends recording for a running capture, but keeps the capture open:

```bash
go run ./cmd/salad --host 127.0.0.1 --port 10430 --timeout 5s capture stop --capture-id <id>
```

- **Close** releases the capture resources (you should treat this as “done with this capture”):

```bash
go run ./cmd/salad --host 127.0.0.1 --port 10430 --timeout 5s capture close --capture-id <id>
```

**Practical rule:** when you’re experimenting with analyzers in the UI and want to keep the capture around, prefer **Stop**, and avoid **Close** until you’re finished.

### Wait for capture completion

`wait` only makes sense for captures that will complete on their own (timed/trigger). For manual captures, this can be invalid depending on server behavior.

```bash
go run ./cmd/salad --host 127.0.0.1 --port 10430 --timeout 30s capture wait --capture-id <id>
```

## Exports (raw CSV / raw binary)

Exports are built on top of a capture id. They are a good “end-to-end” check because they touch capture lookup, validation, and filesystem outputs.

```bash
go run ./cmd/salad --host 127.0.0.1 --port 10430 --timeout 60s export raw-csv \
  --capture-id <id> \
  --directory /abs/path/to/export-dir \
  --digital 0,1,2,3
```

## Analyzers (add/remove)

Analyzers turn raw waveforms into protocol-level events. The Automation API lets you add analyzers by name, but it does **not** expose analyzer schemas, so settings must be provided using UI-visible setting keys.

### “Name” vs “Label” (what you see in the UI)

When you add an analyzer, you provide:

- **`--name`**: the analyzer type (must match the Logic 2 UI name exactly, e.g. `"SPI"`, `"I2C"`, `"Async Serial"`)
- **`--label`**: the user-facing label shown in the UI (this is what you’ll recognize in the Analyzers panel)

If you set `--label "template"`, you will see `"template"` in the UI. That is expected.

### Add an analyzer from a settings file

```bash
go run ./cmd/salad --host 127.0.0.1 --port 10430 --timeout 30s analyzer add \
  --capture-id <id> \
  --name "SPI" \
  --label "SPI: bus0" \
  --settings-yaml /abs/path/to/settings.yaml
```

Expected output:

```text
analyzer_id=<id>
```

### Remove an analyzer

```bash
go run ./cmd/salad --host 127.0.0.1 --port 10430 --timeout 30s analyzer remove \
  --capture-id <capture_id> \
  --analyzer-id <analyzer_id>
```

### Settings formats and typed overrides

Settings can come from:

- **A YAML file** (`--settings-yaml`)
- **A JSON file** (`--settings-json`)
- **Typed overrides** (recommended for quick edits):
  - `--set key=value` (string)
  - `--set-bool key=true`
  - `--set-int key=123`
  - `--set-float key=12.34`

Overrides are applied after the file, so they win.

## Analyzer templates (our conventions)

Template files live in `configs/analyzers/`. They are not official Saleae schemas — they’re “known-good starter configs” that encode UI-visible setting keys.

### SPI (verified)

This template is known to work on a capture that has digital channels 0..3 enabled:

- Template: `configs/analyzers/spi.yaml`
- Keys: `Clock`, `MOSI`, `MISO`, `Enable`

Example:

```bash
go run ./cmd/salad --host 127.0.0.1 --port 10430 --timeout 30s analyzer add \
  --capture-id <id> \
  --name "SPI" \
  --label "SPI: CLK0 MOSI1 MISO2 CS3" \
  --settings-yaml /home/manuel/workspaces/2025-12-27/salad-pass/salad/configs/analyzers/spi.yaml
```

### How to verify in the Logic 2 UI

When you run `analyzer add`, confirm in the Logic 2 UI:

- The analyzer appears in the **Analyzers** panel.
- The label matches your `--label`.
- The channel mapping matches your template / overrides.

If you get confusing errors, the most useful first step is to **ensure the capture is stopped** (not recording) before adding/removing analyzers.

## Testing without real hardware (mock server)

If you want deterministic tests, use `salad-mock` with a scenario file under `configs/mock/`.

```bash
go run ./cmd/salad-mock --config configs/mock/happy-path.yaml --port 10431
go run ./cmd/salad --host 127.0.0.1 --port 10431 appinfo
```

For details, see `salad/pkg/doc/mock-server-user-guide.md`.

## Troubleshooting

This section explains the failures you’ll actually see when wiring captures + analyzers together.

- **“Cannot switch sessions while recording”**:
  - Stop the capture first:
    - `salad capture stop --capture-id <id>`
  - Then retry `analyzer add/remove`.

- **“Analyzer settings errors: Invalid channel(s)”**:
  - Your settings keys are wrong (don’t match UI), or your channels aren’t enabled in the capture.
  - Start by verifying you can add SPI using the known-good template, then iterate.

- **DeadlineExceeded / timeouts**:
  - Increase `--timeout` (it applies both to dialing and RPC calls).
  - Ensure you’re not racing the UI (recording sessions / switching captures).

## Reference

- Saleae Logic 2 Automation API docs: `https://saleae.github.io/logic2-automation/`
- Templates directory: `configs/analyzers/`
- Mock server user guide: `salad/pkg/doc/mock-server-user-guide.md`


