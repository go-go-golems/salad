---
Title: 'Implementation: sessions + JSON output + manifests'
Ticket: 007-SESSIONS-OUTPUT
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
Summary: "Implementation approach for stable machine-readable output (`--json`) and session manifests/directories that persist capture/analyzer IDs and exported artifact paths."
LastUpdated: 2025-12-24T22:42:13.410874285-05:00
WhatFor: ""
WhenToUse: ""
---

# Implementation: sessions + JSON output + manifests

## Goal

Make the CLI genuinely “power-user/scriptable” by adding:
- consistent `--json` output mode across commands
- a session directory + manifest that tracks:
  - appinfo
  - device selection
  - capture IDs
  - analyzer IDs
  - output artifact paths

This is the backbone for `--last`, pipelines, and reproducible debugging runs.

## Output contract

### Default output (human)

Keep the current key/value lines (good for quick copy/paste):
- `capture_id=...`
- `analyzer_id=...`

### JSON output (machine)

Add `--json` global flag:
- Always emit a single JSON object per command
- Include stable keys:
  - `ok`, `error`, `host`, `port`, `timestamp`
  - command-specific payload under `data`

Example shape:

```json
{
  "ok": true,
  "data": {
    "capture_id": 123
  }
}
```

## Session model

### Session directory

Add global flags:
- `--session /abs/dir` (optional; default: none)
- `--session-new` (create a new timestamped session dir; optional)

When session is enabled:
- Write `manifest.json` (append/update) after each command that creates or mutates state.

### Manifest content (minimum)

- `appinfo`
- `device` (device_id/type/is_simulation)
- `captures`: list of `{capture_id, created_at, notes}`
- `analyzers`: list of `{analyzer_id, capture_id, name, label, kind: lla|hla}`
- `artifacts`: list of `{type, path, capture_id, analyzer_ids}`

## Implementation approach

### Internal packages

- `internal/output/`:
  - helpers to write JSON/human output consistently
- `internal/session/`:
  - load/update/write `manifest.json`
  - resolve `--last` capture/analyzer IDs (future)

### CLI integration

- Add persistent flags in `cmd/salad/cmd/root.go`:
  - `--json`
  - session flags
- Update each command to:
  - write outputs via the `internal/output` helper
  - update session manifest when applicable

## Testing strategy

- Unit tests for:
  - JSON output schema consistency
  - manifest update logic
- Manual test:
  - run a sequence of commands with `--session ./tmp/session1`
  - verify manifest contains IDs and file paths

## Open questions / decisions

- Should session dirs default to a repo-local `.salad/` or user home `~/.salad/`?
- JSON schema versioning:
  - include `schema_version` field from day 1?
