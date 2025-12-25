---
Title: 'Implementation: HLAs (extensions integration)'
Ticket: 004-HLA-EXTENSIONS
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
Summary: "Implementation approach for `salad hla add/remove` backed by AddHighLevelAnalyzer RPC, including extension.json-driven selection, settings parsing, and local extension workflows."
LastUpdated: 2025-12-24T22:42:12.438292231-05:00
WhatFor: ""
WhenToUse: ""
---

# Implementation: HLAs (extensions integration)

## Goal

Add CLI support to attach/detach High Level Analyzers (HLAs) to a capture:
- `salad hla add` → returns `analyzer_id`
- `salad hla remove`

This enables “decode pipelines”:
LLA (protocol analyzer) → HLA (Python extension) → export table.

## Proto grounding

Relevant RPC and message types in `proto/saleae/grpc/saleae.proto`:
- `Manager.AddHighLevelAnalyzer(AddHighLevelAnalyzerRequest) returns (AddHighLevelAnalyzerReply)`
- `Manager.RemoveHighLevelAnalyzer(RemoveHighLevelAnalyzerRequest) returns (RemoveHighLevelAnalyzerReply)`
- `AddHighLevelAnalyzerRequest`:
  - `capture_id`
  - `extension_directory` (directory containing `extension.json`)
  - `hla_name` (name of the HLA as listed in `extension.json`)
  - `hla_label`
  - `input_analyzer_id` (the LLA analyzer whose frames feed this HLA)
  - `settings: map<string, HighLevelAnalyzerSettingValue>` (`string_value` or `number_value`)

## CLI surface

### Minimal commands

- `salad hla add --capture-id <id> --extension-dir /abs/ext --name \"My HLA\" --label \"my hla\" --input-analyzer-id <id> [--settings-json /abs/settings.json]`
- `salad hla remove --capture-id <id> --analyzer-id <id>`

### Settings input

Typed settings map is limited (`string` + `number` only), so:
- Prefer `--settings-json` / `--settings-yaml`
- Allow quick overrides:
  - `--set key=value` (string)
  - `--set-number key=12.34`

Avoid auto-guessing types.

### Quality-of-life (nice to have)

- `salad hla list --extension-dir /abs/ext`
  - parse `extension.json` and print available HLAs + entrypoints

## Implementation approach

### extension.json parsing

We want friendly CLI UX (autocomplete-ish), so implement:
- `internal/extensions/manifest.go`:
  - load and parse `extension.json`
  - expose list of entries where `type == HighLevelAnalyzer`

This is a pure local-file read; no gRPC needed.

### Client wrapper

- Add wrapper methods in `internal/saleae/client.go`:
  - `AddHighLevelAnalyzer(...)`
  - `RemoveHighLevelAnalyzer(...)`

### Files likely to change/add

- `cmd/salad/cmd/hla.go` (new Cobra subtree)
- `internal/saleae/client.go` (new RPC wrappers)
- `internal/extensions/` (parse extension.json + validate selection)

## Testing strategy

- Unit tests:
  - parse extension.json samples
  - typed settings parsing (string/number)
- Manual test (requires Logic2 + a known extension directory):
  - load capture → add LLA → add HLA → export table

## Open questions / decisions

- Should `--extension-dir` be required always, or allow “installed local extension” discovery from Logic 2’s extension store (if we can locate it)?
