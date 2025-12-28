---
Title: Compare mock server against real Saleae Logic automation server
Ticket: 011-MOCK-AGAINST-REAL
Status: active
Topics:
    - saleae
    - mock-server
    - testing
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: salad/ttmp/2025/12/27/011-MOCK-AGAINST-REAL--compare-mock-server-against-real-saleae-logic-automation-server/scripts/01-probe-real-vs-mock.sh
      Note: Wrapper to run probe and write output into ticket various/
    - Path: salad/ttmp/2025/12/27/011-MOCK-AGAINST-REAL--compare-mock-server-against-real-saleae-logic-automation-server/scripts/probe_real_vs_mock.go
      Note: RPC-level probe tool that queries real+mock and emits JSON + focused diff
    - Path: ttmp/2025/12/27/011-MOCK-AGAINST-REAL--compare-mock-server-against-real-saleae-logic-automation-server/scripts/02-probe-with-mock-start.sh
      Note: 'One-shot runner: start mock (configurable)'
    - Path: ttmp/2025/12/27/011-MOCK-AGAINST-REAL--compare-mock-server-against-real-saleae-logic-automation-server/various/probe-20251227-165937.json
      Note: First real-vs-mock probe evidence (shows status-code differences like Aborted vs InvalidArgument)
ExternalSources: []
Summary: ""
LastUpdated: 2025-12-27T16:51:53.520636429-05:00
WhatFor: ""
WhenToUse: ""
---



# Compare mock server against real Saleae Logic automation server

## Overview

Now that we have a YAML-driven `salad-mock` gRPC server (ticket `010-MOCK-SERVER`), this ticket focuses on **validating its behavioral correctness** by comparing it against the real Saleae Logic 2 automation server.

The goal is to end up with:
- a repeatable “comparison suite” (CLI-level and/or RPC-level),
- a documented list of intentional divergences, and
- a clear loop for fixing mock drift when the real server behavior changes (or when we misunderstood it).

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field
- **Analysis**: [analysis/01-mock-vs-real-comparison-strategy.md](./analysis/01-mock-vs-real-comparison-strategy.md)
- **Diary**: [reference/01-diary.md](./reference/01-diary.md)

## Status

Current status: **active**

## Topics

- saleae
- mock-server
- testing

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
