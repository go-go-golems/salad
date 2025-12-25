---
Title: 'Initial SALAD: Saleae Logic Analyzer client in Go'
Ticket: 001-INITIAL-SALAD
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
    - Path: cmd/salad/cmd/appinfo.go
      Note: appinfo command
    - Path: cmd/salad/cmd/capture.go
      Note: Capture commands
    - Path: cmd/salad/cmd/devices.go
      Note: CLI command for GetDevices
    - Path: cmd/salad/cmd/export.go
      Note: Export commands
    - Path: cmd/salad/cmd/root.go
      Note: Root Cobra command and flags
    - Path: cmd/salad/main.go
      Note: CLI entry
    - Path: gen/saleae/automation/saleae.pb.go
      Note: Generated Go bindings
    - Path: gen/saleae/automation/saleae_grpc.pb.go
      Note: Generated Go gRPC bindings
    - Path: go.mod
      Note: Go module root
    - Path: internal/saleae/client.go
      Note: |-
        Saleae client wrapper
        Saleae client wrappers
    - Path: internal/saleae/config.go
      Note: Client config
    - Path: proto/saleae/grpc/saleae.proto
      Note: Vendored Saleae automation API proto
    - Path: ttmp/2025/12/24/001-INITIAL-SALAD--initial-salad-saleae-logic-analyzer-client-in-go/playbook/01-manual-smoke-test-logic2-automation.md
      Note: Manual smoke test playbook
    - Path: ttmp/2025/12/24/001-INITIAL-SALAD--initial-salad-saleae-logic-analyzer-client-in-go/reference/01-diary.md
      Note: Live test against Logic2 automation
ExternalSources: []
Summary: ""
LastUpdated: 2025-12-24T20:59:04.48047343-05:00
WhatFor: ""
WhenToUse: ""
---






# Initial SALAD: Saleae Logic Analyzer client in Go

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field
- **Starting research**: [sources/research-salad-salaea-go-client.md](./sources/research-salad-salaea-go-client.md)
- **Build plan (analysis)**: [analysis/01-build-plan-saleae-logic2-grpc-client-go.md](./analysis/01-build-plan-saleae-logic2-grpc-client-go.md)
- **Implementation diary**: [reference/01-diary.md](./reference/01-diary.md)

## Status

Current status: **active**

## Topics

- go
- saleae
- logic-analyzer
- client

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
