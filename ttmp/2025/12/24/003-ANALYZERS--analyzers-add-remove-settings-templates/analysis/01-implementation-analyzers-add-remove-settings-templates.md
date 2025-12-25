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

### Files likely to change/add

- `cmd/salad/cmd/analyzer.go` (new Cobra subtree)
- `internal/saleae/client.go` (add `AddAnalyzer`, `RemoveAnalyzer` wrappers)
- `internal/config/analyzer_settings.go` (parse settings files/flags)
- `configs/analyzers/` (templates; optional)

### Error UX

- When `AddAnalyzer` fails, wrap error with:
  - analyzer name, capture id
  - hint that keys must match UI-visible setting names

## Testing strategy

- Unit tests for settings parsing:
  - file parsing, typed overrides, merge rules
- Manual test:
  - capture load → add SPI analyzer with known-good template → export table (once implemented)

## Open questions / decisions

- Template storage format: YAML only vs JSON+YAML.
- Whether to ship templates in-repo or in `ttmp/.../reference` as initial experiments.
