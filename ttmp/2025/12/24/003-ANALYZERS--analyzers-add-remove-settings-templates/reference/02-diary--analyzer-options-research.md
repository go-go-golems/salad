---
Title: "Diary: Research — analyzer options/settings discovery"
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
RelatedFiles:
  - Path: /home/manuel/workspaces/2025-12-27/salad-pass/salad/proto/saleae/grpc/saleae.proto
    Note: Defines the gRPC surface (AddAnalyzer/RemoveAnalyzer) and the settings value types.
  - Path: /home/manuel/workspaces/2025-12-27/salad-pass/salad/internal/saleae/client.go
    Note: Our client wrapper that calls AddAnalyzer/RemoveAnalyzer.
  - Path: /home/manuel/workspaces/2025-12-27/salad-pass/salad/internal/config/analyzer_settings.go
    Note: Our settings file/override parser (templates are just settings files).
  - Path: /home/manuel/workspaces/2025-12-27/salad-pass/salad/configs/analyzers/spi.yaml
    Note: Existing SPI settings template (channels only).
ExternalSources: []
Summary: "Research notes answering: what analyzer options can be templated, what protocol methods exist, and how to discover analyzer setting keys/values."
LastUpdated: 2025-12-27T00:00:00Z
WhatFor: ""
WhenToUse: ""
---

# Diary: Research — analyzer options/settings discovery

## Goal

Figure out whether the “analyzer options” visible in the Saleae Logic 2 UI (e.g., SPI CPOL/CPHA, Bits per Transfer) can be set via our YAML templates, which protocol methods are involved, and how to discover what keys/values are available for a given analyzer.

## Step 1: Confirm protocol surface and whether schema introspection exists

I started by validating what the Saleae automation API actually exposes over gRPC. The primary question was: is there any API method that can enumerate an analyzer’s available settings/options (schema introspection), or are we expected to “know” these keys and values out-of-band?

### What I did
- Read `saleae/proto/saleae/grpc/saleae.proto` around `service Manager` and the analyzer request/setting message types.
- Cross-checked our client wrapper in `internal/saleae/client.go`.

### What worked
- Confirmed the only analyzer-related gRPC RPCs are:
  - `Manager.AddAnalyzer`
  - `Manager.RemoveAnalyzer`
  - `Manager.AddHighLevelAnalyzer`
  - `Manager.RemoveHighLevelAnalyzer`
- Confirmed there is **no** `ListAnalyzers`, `GetAnalyzerSettingsSchema`, or similar schema/introspection RPC in this proto.

### What I learned
- Analyzer settings are passed inline in `AddAnalyzerRequest.settings` as a `map<string, AnalyzerSettingValue>`.
- Therefore any “template” system on our side is necessarily an out-of-band convenience: **Saleae doesn’t define templates in the protocol**; the protocol only accepts settings maps.

### What warrants a second pair of eyes
- N/A (proto surface is straightforward), but it’s worth double-checking if newer Saleae proto versions introduce schema discovery (this repo vendors a specific proto).

## Step 2: Confirm the key/value contract (UI text is the canonical schema)

The core uncertainty was: do setting keys need to match UI labels exactly (including spaces/capitalization), and do dropdown values need to match the visible option strings, or can we use some internal code?

### What I did
- Checked the comments in `saleae.proto` for `AddAnalyzerRequest.settings`.
- Consulted the official Saleae Logic 2 Automation docs:
  - `https://saleae.github.io/logic2-automation/getting_started.html`
  - `https://saleae.github.io/logic2-automation/automation.html`
- Compared with the existing real-server smoke test notes in the ticket’s main diary.

### What worked
- The official docs for `saleae.automation.Capture.add_analyzer(..., settings=...)` state:
  - “The keys and values here must exactly match the Analyzer settings as shown in the UI”.
- The Getting Started example uses SPI settings where a dropdown value is passed as the **full UI-visible string**, e.g.:
  - `{"Bits per Transfer": "8 Bits per Transfer (Standard)"}`

### What I learned
- For UI dropdowns (like “Bits per Transfer”), the value is encoded as a `string` and must match the UI option text.
- For channel selection (like “Clock”, “MISO”, “Enable”), values are `int` channel indices (0-based).

### What was tricky to build
- The Sphinx docs search required manual navigation; simple web searching for “analyzer settings” often collides with unrelated “text analyzers” (e.g., Elasticsearch).

## Step 3: Map research findings to our template/settings implementation

Now that we know the API contract, I checked whether our parser can represent the needed value types (string/int/bool/float) and whether templates can express “all the SPI UI options”.

### What I did
- Read `internal/config/analyzer_settings.go` to confirm which YAML/JSON shapes and scalar types we support.
- Confirmed the existing `configs/analyzers/spi.yaml` is parsed through that code path.

### What worked
- Templates can set any analyzer settings **that can be expressed as scalar values** (string/bool/int/float).
- The SPI UI settings shown in the screenshot are all representable with these scalar types:
  - channel selectors → int
  - dropdown selections → string

### What I learned
- Our SPI template currently only sets the channel mapping. It can be extended to include other UI options (Bits per Transfer, CPOL/CPHA, etc.) by adding those keys as string values matching the UI.

### What should be done in the future
- Consider adding a short “how to discover keys/values” runbook (UI → template) and optionally a “template variants” pack for common SPI modes (CPOL/CPHA combos).

## Step 4: Answer “can we extract settings from a UI-configured analyzer?”

This step clarifies an important limitation: even though the UI is the canonical source for setting keys/values, the automation API does not provide a way to read back a configured analyzer as structured settings.

### What I did
- Re-checked `saleae.proto` for any RPC that would:
  - list analyzers on a capture, or
  - fetch analyzer configuration/settings.

### What worked
- Confirmed the proto does not include any “get analyzer config” RPC.

### What I learned
- If you configure an analyzer in the UI, the automation API cannot directly tell you “what settings did you pick”.
- The only reliable ways to obtain “the right settings” are:
  - copy them from the UI, or
  - (best-effort) save setup/capture and inspect the resulting file(s) for analyzer settings.

### Technical details
- This ticket includes a helper script to extract analyzer settings from an unzipped `.sal` session `meta.json`:
  - `scripts/02-extract-analyzer-settings-from-meta-json.py`
  - It emits dropdown values as UI-visible strings by default (matching Saleae’s automation docs).

### What was tricky to build
- N/A (this is a limitation discovery, not implementation).


