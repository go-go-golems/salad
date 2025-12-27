---
Title: "Diary: Analyzers (add/remove + settings/templates)"
Ticket: 003-ANALYZERS
Status: active
Topics:
    - go
    - saleae
    - logic-analyzer
    - client
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2025-12-27T17:08:57.00980015-05:00
WhatFor: ""
WhenToUse: ""
---

# Diary: Analyzers (add/remove + settings/templates)

## Goal

Implement `salad analyzer add/remove` (ticket 003) in a way that can be tested against:
- a **real** Logic 2 automation gRPC server (manual smoke test), and
- the **mock** server (ticket 010) later for CI and deterministic tests.

## Step 1: Create implementation diary + identify minimum compile-green units

This step bootstraps the diary and sets up an execution cadence: implement in small compile-green increments, commit each increment, then record what happened (including failures) while it’s still fresh. This keeps progress reviewable and reduces the risk of “one huge commit” with unclear breakpoints.

**Commit (code):** N/A — documentation scaffolding

### What I did
- Created `ttmp/2025/12/24/003-ANALYZERS--analyzers-add-remove-settings-templates/reference/01-diary.md` via `docmgr doc add`
- Confirmed analyzer RPCs and message schemas in `proto/saleae/grpc/saleae.proto`:
  - `AddAnalyzer(AddAnalyzerRequest) -> AddAnalyzerReply`
  - `RemoveAnalyzer(RemoveAnalyzerRequest) -> RemoveAnalyzerReply`
- Identified the smallest compile-green implementation units to commit:
  - Add client wrappers (`internal/saleae/client.go`)
  - Add analyzer settings parsing (`internal/config/...`)
  - Add Cobra verbs (`cmd/salad/cmd/analyzer.go` + `cmd/salad/cmd/root.go`)

### Why
- The analyzer feature spans client + CLI + config parsing; splitting into small commits makes it much easier to review and debug.

### What worked
- The proto clearly defines analyzer settings as a typed oneof (`AnalyzerSettingValue`), which maps well to a small parsing layer.

### What didn't work
- N/A

### What I learned
- `AddAnalyzerReply` returns `analyzer_id` (uint64). That value should be printed by the CLI the same way capture commands print `capture_id=...`.

### What was tricky to build
- N/A (setup step)

### What warrants a second pair of eyes
- N/A (setup step)

### What should be done in the future
- N/A

### Code review instructions
- Start at `proto/saleae/grpc/saleae.proto` around `AddAnalyzerRequest` to confirm request/response contracts.

