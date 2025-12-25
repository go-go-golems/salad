---
Title: 'Implementation: doctor + troubleshooting + gRPC knobs'
Ticket: 008-DOCTOR-TROUBLESHOOTING
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
Summary: "Implementation approach for `salad doctor` plus gRPC troubleshooting knobs (dial/rpc timeouts, connectivity checks, export writeability checks) to improve debugging UX."
LastUpdated: 2025-12-24T22:42:13.642488648-05:00
WhatFor: ""
WhenToUse: ""
---

# Implementation: doctor + troubleshooting + gRPC knobs

## Goal

Add first-class debugging UX for the CLI itself:
- `salad doctor` that runs a connectivity + environment sanity checklist
- clearer gRPC dialing and timeout controls
- optional verbose diagnostics to help users troubleshoot Logic2 automation issues

## `salad doctor` checklist (MVP)

- **Connectivity**
  - dial gRPC with explicit timeout
  - call `GetAppInfo`
- **Device visibility**
  - call `GetDevices` (with and without simulation)
  - warn if no physical devices
- **Export preflight**
  - if user passes `--export-dir`, validate it exists and is writable
- **Version reporting**
  - print Logic2 application_version + api_version
  - print CLI version (if we later add build versioning)

## Flags / knobs

Add global flags (or doctor-only flags):
- `--dial-timeout 5s` (separate from `--timeout` used for RPC calls)
- `--rpc-timeout 5s` (rename existing `--timeout` or add)
- `--retries 0` / `--retry-wait 200ms` for doctor connectivity checks
- `--verbose` (print stack-wrapped errors + extra context)

## Implementation approach

### Client changes

- Extend `internal/saleae.New(...)` to support separate dial timeout vs rpc timeout.
  - Keep the existing behavior as default (no breaking change unless you want it).

### Command

- `cmd/salad/cmd/doctor.go`
  - execute checks in order
  - print a short report (human) and optionally JSON

### Error messages

Standardize common failure hints:
- connection refused → “Logic2 not started with --automation or wrong port”
- deadline exceeded → “increase timeouts / check firewall if remote host”

## Testing strategy

- Unit tests for doctor report formatting
- Manual test against running Logic2

## Open questions / decisions

- Should `doctor` attempt to start Logic2? (probably no; keep it as a checker)
