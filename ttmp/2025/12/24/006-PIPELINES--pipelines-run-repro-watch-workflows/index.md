---
Title: 'Pipelines: run/repro/watch workflows'
Ticket: 006-PIPELINES
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
    - Path: cmd/salad/cmd
      Note: Commands that pipeline runner will orchestrate
    - Path: cmd/salad/cmd/run.go
      Note: `salad run` pipeline v1 command
    - Path: internal/pipeline
      Note: Pipeline config + runner implementation
    - Path: ttmp/2025/12/24/006-PIPELINES--pipelines-run-repro-watch-workflows/analysis/01-implementation-pipeline-runner-run-repro-watch.md
      Note: Updated analysis doc with current constraints and file pointers
    - Path: ttmp/2025/12/24/006-PIPELINES--pipelines-run-repro-watch-workflows/reference/01-diary.md
      Note: Implementation diary for ticket 006
    - Path: ttmp/2025/12/24/006-PIPELINES--pipelines-run-repro-watch-workflows/scripts/README.md
      Note: How to run the pipeline against the mock server (fast loop)
ExternalSources: []
Summary: "Config-driven pipeline runner (`salad run`) to orchestrate capture/analyzer/export workflows; repro/watch planned."
LastUpdated: 2025-12-28T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Pipelines: run/repro/watch workflows

Document workspace for 006-PIPELINES.

## Key docs

- `analysis/01-implementation-pipeline-runner-run-repro-watch.md`
- `reference/01-diary.md`
- `scripts/README.md`
