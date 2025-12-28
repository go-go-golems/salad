---
Title: 'Capture start: manual/timed/trigger'
Ticket: 002-CAPTURE-START
Status: active
Topics:
    - go
    - saleae
    - logic-analyzer
    - client
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ttmp/2025/12/24/002-CAPTURE-START--capture-start-manual-timed-trigger/analysis/01-implementation-capture-start-manual-timed-trigger.md
      Note: Initial implementation approach and design
    - Path: ttmp/2025/12/24/002-CAPTURE-START--capture-start-manual-timed-trigger/analysis/02-comprehensive-proto-codebase-analysis.md
      Note: Complete proto structure analysis and codebase integration guide
    - Path: ttmp/2025/12/24/002-CAPTURE-START--capture-start-manual-timed-trigger/reference/01-diary.md
      Note: Implementation diary documenting analysis phase and learnings
    - Path: proto/saleae/grpc/saleae.proto
      Note: Source proto defining StartCapture RPC and message types
    - Path: gen/saleae/automation/saleae.pb.go
      Note: Generated Go types for StartCapture
    - Path: internal/saleae/client.go
      Note: Client wrapper where StartCapture method will be added
    - Path: cmd/salad/cmd/capture.go
      Note: Existing capture commands showing CLI structure
ExternalSources: []
Summary: ""
LastUpdated: 2025-12-27T00:00:00Z
WhatFor: ""
WhenToUse: ""
---

# Capture start: manual/timed/trigger

## Overview

Implement `salad capture start` command supporting three capture modes:
- **Manual mode**: Ring buffer semantics, stop manually
- **Timed mode**: Capture ends after specified duration
- **Digital trigger mode**: Capture ends when trigger condition is met

## Key Links

- **Initial design**: [analysis/01-implementation-capture-start-manual-timed-trigger.md](./analysis/01-implementation-capture-start-manual-timed-trigger.md)
- **Comprehensive analysis**: [analysis/02-comprehensive-proto-codebase-analysis.md](./analysis/02-comprehensive-proto-codebase-analysis.md)
- **Implementation diary**: [reference/01-diary.md](./reference/01-diary.md)

## Status

Current status: **active** — Analysis phase complete, ready for implementation

## Progress

- ✅ Proto structure analysis complete
- ✅ Codebase pattern analysis complete
- ✅ Config parsing strategy designed
- ✅ Implementation plan documented
- 🔄 Implementation pending

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.
