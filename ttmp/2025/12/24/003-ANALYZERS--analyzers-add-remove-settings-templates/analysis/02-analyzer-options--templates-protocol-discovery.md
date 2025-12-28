---
Title: "Analysis: Analyzer options — templates vs protocol, and how to discover settings"
Ticket: 003-ANALYZERS
Status: active
Topics:
  - go
  - saleae
  - logic-analyzer
  - client
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
  - Path: /home/manuel/workspaces/2025-12-27/salad-pass/salad/proto/saleae/grpc/saleae.proto
    Note: Source-of-truth for which RPCs exist and how analyzer settings are encoded.
  - Path: /home/manuel/workspaces/2025-12-27/salad-pass/salad/internal/saleae/client.go
    Note: Where `salad` calls `Manager.AddAnalyzer` / `Manager.RemoveAnalyzer`.
  - Path: /home/manuel/workspaces/2025-12-27/salad-pass/salad/internal/config/analyzer_settings.go
    Note: How settings/templates YAML/JSON are parsed into the proto’s typed settings map.
  - Path: /home/manuel/workspaces/2025-12-27/salad-pass/salad/configs/analyzers/spi.yaml
    Note: Existing SPI template (can be extended to include more UI options).
ExternalSources: []
Summary: "Explains which analyzer UI options can be set via settings templates, which Saleae gRPC methods are involved, and practical ways to discover valid setting keys/values for an analyzer."
LastUpdated: 2025-12-27T00:00:00Z
WhatFor: ""
WhenToUse: ""
---

# Analyzer options — templates vs protocol, and how to discover settings

## Question: “Can all those analyzer UI options be set in templates too?”

**Yes — if (and only if) the option is part of the analyzer’s `settings` map in `AddAnalyzer`.**

In this repo, a “template” is just a YAML/JSON file that we load and pass as `AddAnalyzerRequest.settings`. There is no separate “template” concept in the Saleae protocol.

### What this means in practice

- If a setting appears in the analyzer’s settings dialog and is accepted by `AddAnalyzer`, then it can be represented in a template file by adding:
  - the **UI-visible setting name** as the key, and
  - a scalar value (`string`, `int`, `bool`, `float`) matching what the UI expects.

- If a toggle/option is *not* expressed via `AddAnalyzerRequest.settings`, then **it cannot be templated** (because there is no RPC to set it).

#### Example: SPI analyzer (screenshot)

Based on:
- the proto’s comment (“settings should match the names shown … in the application”), and
- Saleae’s own “Getting Started” example (which sets `Bits per Transfer` using the full dropdown string),

you can template many of the screenshot’s fields by copying the UI labels and option strings exactly.

## Which protocol methods are used (and what they carry)

### Low-level protocol analyzers (SPI/I2C/Async Serial, etc.)

- `Manager.AddAnalyzer(AddAnalyzerRequest) -> AddAnalyzerReply`
  - `analyzer_name`: must match the name shown in the UI (e.g. `"SPI"`)
  - `analyzer_label`: user-friendly label shown in the UI
  - `settings`: a `map<string, AnalyzerSettingValue>`
- `Manager.RemoveAnalyzer(RemoveAnalyzerRequest) -> RemoveAnalyzerReply`

`AnalyzerSettingValue` is a typed oneof (string/int64/bool/double). Our settings parser supports those same scalar types.

### High Level Analyzers (HLA)

- `Manager.AddHighLevelAnalyzer(AddHighLevelAnalyzerRequest) -> AddHighLevelAnalyzerReply`
- `Manager.RemoveHighLevelAnalyzer(RemoveHighLevelAnalyzerRequest) -> RemoveHighLevelAnalyzerReply`

Note: HLA settings are a different message (`HighLevelAnalyzerSettingValue`) and support fewer scalar types (string + number).

### Exporting analyzer results

Once you have an `analyzer_id`, exporting analyzer results is done via:
- `Manager.ExportDataTableCsv(ExportDataTableCsvRequest)`
  - includes `DataTableAnalyzerConfiguration{ analyzer_id, radix_type }`

## How do you figure out what options exist for an analyzer?

### Key constraint

The Saleae automation gRPC API in `saleae.proto` does **not** include any schema/introspection method (no “list settings”, no “describe analyzer”). So you cannot discover valid keys/options purely via RPC.

### Practical discovery methods (ranked)

1. **Copy from the UI settings dialog**
   - Keys: use the exact UI label (including spaces/punctuation), e.g. `Bits per Transfer`.
   - Values:
     - channel selectors: integer channel index (0-based)
     - dropdowns: the full dropdown string, e.g. `8 Bits per Transfer (Standard)`

2. **Use Saleae’s official Automation docs/examples**
   - The docs explicitly state keys/values must match the UI.
   - The “Getting Started” example demonstrates passing UI strings as values (SPI’s `Bits per Transfer`).
   - Docs entry points:
     - `https://saleae.github.io/logic2-automation/getting_started.html`
     - `https://saleae.github.io/logic2-automation/automation.html` (search for `Capture.add_analyzer`)

3. **Trial-and-error using `AddAnalyzer` and reading validation errors**
   - Works, but errors can be vague (e.g., “Invalid channel(s)”).
   - Best combined with templates: start from known-good minimal settings, then add one option at a time.

4. **(Optional) “Round trip” via a saved capture**
   - Configure the analyzer in the UI, save the capture (`SaveCapture` or UI save), then inspect the saved capture file for stored analyzer settings.
   - This is useful when a dropdown value is hard to guess or has subtle punctuation.
   - Caveat: capture file format may change across Logic 2 versions; treat this as a debugging technique, not a stable contract.

### Can we “extract” the exact settings from a UI-configured analyzer automatically?

**Not via the gRPC automation API in this repo.** There is no RPC to list analyzers or read back an analyzer’s current settings (no schema/introspection, and no “get analyzer config”).

Practical options are therefore:
- **Manual**: copy keys and dropdown values from the UI into a YAML template (most reliable, but manual).
- **Best-effort round trip**: save the capture / save the “setup” (if your Logic 2 version supports it) and inspect the resulting file(s) for analyzer settings. This can yield the *exact* dropdown strings, but the file format is not part of the public gRPC contract.

#### Scripted extraction (recommended)

If you unzip a `.sal` session and obtain its `meta.json`, you can extract analyzer settings into a `salad`-compatible YAML template using:

- `scripts/02-extract-analyzer-settings-from-meta-json.py`
  - `--list` to find the analyzer `nodeId`
  - `--node-id <id> --format yaml` to print a `settings:` block

## Concrete example: extend `configs/analyzers/spi.yaml` to include UI options

The current template only sets channels. You can extend it with dropdown settings by copying strings from the UI (or from Saleae’s docs example).

Example (illustrative; strings must match your UI exactly):

```yaml
settings:
  Clock: 0
  MOSI: 1
  MISO: 2
  Enable: 3

  Significant Bit: Most Significant Bit First (Standard)
  Bits per Transfer: 8 Bits per Transfer (Standard)
  Clock State: Clock is Low when inactive (CPOL = 0)
  Clock Phase: Data is Valid on Clock Leading Edge (CPHA = 0)
  Enable Line: Enable line is Active Low (Standard)
```

## Notes / gotchas

- Settings are **free-form** on the client side; the server validates them.
- The only stable “schema” is the UI label text for keys and dropdown values.
- Some UI toggles (e.g. “Show in protocol results table”, “Stream to terminal”) are not represented in the current proto; if you can’t find it in `AddAnalyzerRequest.settings`, assume it’s not automatable via this API version.


