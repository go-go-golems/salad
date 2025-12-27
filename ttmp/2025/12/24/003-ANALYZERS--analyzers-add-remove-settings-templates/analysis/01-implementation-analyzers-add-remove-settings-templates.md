---
Title: 'Implementation: analyzers (add/remove + settings/templates)'
Ticket: 003-ANALYZERS
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
Summary: "Implementation approach for analyzer commands (AddAnalyzer/RemoveAnalyzer) including typed settings parsing and optional template files."
LastUpdated: 2025-12-24T22:42:12.216124562-05:00
WhatFor: ""
WhenToUse: ""
---

# Implementation: analyzers (add/remove + settings/templates)

## Goal

Provide CLI-first protocol decoding “as code”:
- `salad analyzer add` → returns `analyzer_id`
- `salad analyzer remove`
- Make analyzer settings reproducible via JSON/YAML files and/or templates.

## Proto grounding

Relevant RPC and message types:
- `Manager.AddAnalyzer(AddAnalyzerRequest) returns (AddAnalyzerReply)`
- `Manager.RemoveAnalyzer(RemoveAnalyzerRequest) returns (RemoveAnalyzerReply)`
- `AddAnalyzerRequest`:
  - `capture_id`
  - `analyzer_name` (must match UI name, e.g. `"SPI"`, `"I2C"`, `"Async Serial"`)
  - `analyzer_label`
  - `settings: map<string, AnalyzerSettingValue>`
- `AnalyzerSettingValue` is a `oneof`:
  - `string_value`, `int64_value`, `bool_value`, `double_value`

Key constraint: **the API does not expose analyzer schemas** (setting keys/options), so the CLI must lean on templates and user-provided settings.

## CLI surface

### Minimal commands

- `salad analyzer add --capture-id <id> --name "SPI" --label "boot spi" --settings-json /abs/settings.json`
- `salad analyzer remove --capture-id <id> --analyzer-id <id>`

### Settings input options

Prefer deterministic file input:
- `--settings-json /abs/settings.json`
- `--settings-yaml /abs/settings.yaml`

Also allow quick overrides:
- `--set key=value` (string by default)
- `--set-bool key=true`
- `--set-int key=123`
- `--set-float key=12.34`

We should avoid type-guessing heuristics as the default (too error-prone in debugging workflows).

## Templates (optional but high leverage)

Add a curated template directory:
- `configs/analyzers/spi.yaml`
- `configs/analyzers/i2c.yaml`
- `configs/analyzers/async-serial.yaml`

Template commands:
- `salad analyzer template list`
- `salad analyzer template show spi`

Implementation detail: templates are **our** conventions, not Saleae’s API.

## Implementation approach

### Data model

- Parse settings input into:
  - `map[string]pb.AnalyzerSettingValue`
- If a template is used:
  - load template → merge overrides → build pb map

## Mock server support (to test analyzers without Logic 2)

To test `salad analyzer add/remove` reliably in CI, extend the mock Saleae gRPC server (ticket 010-MOCK-SERVER) to implement analyzer RPCs and track analyzer state in-memory.

### Required RPCs to implement in the mock server

- `Manager.AddAnalyzer(AddAnalyzerRequest) returns (AddAnalyzerReply)`
- `Manager.RemoveAnalyzer(RemoveAnalyzerRequest) returns (RemoveAnalyzerReply)`

These should follow the existing mock server “exec pipeline” pattern (fault injection + plan + state) used by capture and export RPCs.

### Minimal analyzer state model (mock)

Add a simple in-memory model keyed by `(capture_id, analyzer_id)`:

- `AnalyzerState`:
  - `CaptureID uint64`
  - `AnalyzerID uint64` (or string, but prefer uint64 for parity with proto)
  - `AnalyzerName string`
  - `AnalyzerLabel string`
  - `Settings map[string]*pb.AnalyzerSettingValue` (store the request map)
  - `CreatedAt time.Time` (optional; useful for debugging)

Server state needs:

- `NextAnalyzerID uint64` (deterministic when configured)
- `Analyzers map[uint64]map[uint64]*AnalyzerState` (captures → analyzers)

### Behavior + validation knobs (YAML DSL)

Add YAML behavior sections mirroring existing mock patterns (validate + on_call + defaults):

- `behavior.AddAnalyzer`:
  - validate:
    - `require_capture_exists: true|false`
    - `require_analyzer_name_non_empty: true|false`
  - on_call:
    - `create_analyzer:` (optional; allows forcing returned analyzer_id, or controlling behavior)
      - `analyzer_id_start:` (optional if we want separate namespace)

- `behavior.RemoveAnalyzer`:
  - validate:
    - `require_capture_exists: true|false`
    - `require_analyzer_exists: true|false`
  - on_call:
    - `delete: true|false` (or “mark removed”; for now delete is fine)

Fault injection support should include:

- Always error on method
- Error Nth call
- Matchers:
  - `capture_id`
  - `analyzer_id` (for RemoveAnalyzer)
  - optional: `analyzer_name` (for AddAnalyzer) to simulate per-analyzer failures

### Happy-path semantics (mock)

- **AddAnalyzer**:
  - Validate capture exists (unless disabled)
  - Allocate `analyzer_id` from `NextAnalyzerID` and store state
  - Return `AddAnalyzerReply{ analyzer_id: <id> }` (or whatever the proto reply field is)

- **RemoveAnalyzer**:
  - Validate capture exists (unless disabled)
  - Validate analyzer exists (unless disabled)
  - Delete from map
  - Return empty success reply

### Test strategy (against mock)

Add table-driven integration tests that run the CLI against the mock server:

- Setup:
  - Start mock server from YAML scenario with:
    - one device
    - capture fixture (or call `StartCapture` / `LoadCapture` first)
  - Use deterministic IDs

- Cases:
  - Add analyzer success: returns analyzer_id, can remove it
  - Remove analyzer not found: returns configured error code/message
  - Add analyzer with missing capture: returns error (when require_capture_exists=true)
  - Optional: validate settings map round-trip (server stores the settings as received)

### Files likely to change/add

- `cmd/salad/cmd/analyzer.go` (new Cobra subtree)
- `internal/saleae/client.go` (add `AddAnalyzer`, `RemoveAnalyzer` wrappers)
- `internal/config/analyzer_settings.go` (parse settings files/flags)
- `configs/analyzers/` (templates; optional)
  - Mock support: `internal/mock/saleae/*` (new analyzer state + RPCs)
  - Mock scenarios: `configs/mock/*` (add analyzer-focused YAML scenarios)

### Error UX

- When `AddAnalyzer` fails, wrap error with:
  - analyzer name, capture id
  - hint that keys must match UI-visible setting names

## Testing strategy

- Unit tests for settings parsing:
  - file parsing, typed overrides, merge rules
- Manual test:
  - capture load → add SPI analyzer with known-good template → export table (once implemented)
 - Mock test:
  - Start mock server → load/start capture → add/remove analyzer (table-driven, deterministic)

## Open questions / decisions

- Template storage format: YAML only vs JSON+YAML.
- Whether to ship templates in-repo or in `ttmp/.../reference` as initial experiments.
