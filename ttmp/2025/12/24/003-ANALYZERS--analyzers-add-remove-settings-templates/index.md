---
Title: 'Analyzers: add/remove + settings templates'
Ticket: 003-ANALYZERS
Status: complete
Topics:
    - go
    - saleae
    - logic-analyzer
    - client
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
  - Path: /home/manuel/workspaces/2025-12-27/salad-pass/salad/cmd/salad/cmd/analyzer.go
    Note: CLI verbs `salad analyzer add/remove`.
  - Path: /home/manuel/workspaces/2025-12-27/salad-pass/salad/internal/config/analyzer_settings.go
    Note: Settings/template parsing and typed overrides.
  - Path: /home/manuel/workspaces/2025-12-27/salad-pass/salad/configs/analyzers/
    Note: Analyzer settings templates generated from UI sessions and hand-curated templates.
  - Path: /home/manuel/workspaces/2025-12-27/salad-pass/salad/internal/mock/saleae/server.go
    Note: Mock server includes analyzer RPC support used by CLI integration tests.
  - Path: /home/manuel/workspaces/2025-12-27/salad-pass/salad/internal/mock/saleae/analyzers_cli_test.go
    Note: CLI-vs-mock analyzer add/remove integration test.
  - Path: /home/manuel/workspaces/2025-12-27/salad-pass/salad/ttmp/2025/12/24/003-ANALYZERS--analyzers-add-remove-settings-templates/scripts/
    Note: Ticket-local scripts for extraction, validation, and real-server smoke tests.
ExternalSources: []
Summary: "Completed analyzer add/remove verbs, settings/templates plumbing, UI-derived templates, and validation workflows (real + mock)."
LastUpdated: 2025-12-28T00:00:00Z
WhatFor: ""
WhenToUse: ""
---

# Analyzers: add/remove + settings templates

Document workspace for 003-ANALYZERS.

## Key links

- **Implementation doc**: `analysis/01-implementation-analyzers-add-remove-settings-templates.md`
- **Analyzer options research**: `analysis/02-analyzer-options--templates-protocol-discovery.md`
- **Analysis: SaveCapture/meta.json feedback loop**: `analysis/03-analysis--savecapture-meta-compare-feedback-loop.md`
- **Diary**: `reference/01-diary.md`
- **Diary (research)**: `reference/02-diary--analyzer-options-research.md`
- **Playbook: Extract from .sal**: `reference/03-playbook--extract-analyzers-from-sal.md`
- **Tasks**: `tasks.md`
- **Changelog**: `changelog.md`
