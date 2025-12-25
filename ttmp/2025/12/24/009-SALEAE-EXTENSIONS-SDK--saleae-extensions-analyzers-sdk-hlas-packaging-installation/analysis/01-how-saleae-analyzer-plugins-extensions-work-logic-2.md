---
Title: How Saleae analyzer plugins/extensions work (Logic 2)
Ticket: 009-SALEAE-EXTENSIONS-SDK
Status: active
Topics:
    - saleae
    - logic-analyzer
    - extensions
    - sdk
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "How Saleae Logic 2 is extended via (1) C++ low-level analyzers built with the Protocol Analyzer SDK and (2) Python Extensions (HLAs/Measurements) packaged with extension.json; includes how each is loaded into Logic 2."
LastUpdated: 2025-12-24T22:45:18.053463442-05:00
WhatFor: ""
WhenToUse: ""
---

# How Saleae analyzer plugins/extensions work (Logic 2)

Saleae Logic 2 has two distinct extensibility “layers” relevant to building custom protocol tooling:

- **Low-level analyzers (LLAs)**: native plugins built with the **Protocol Analyzer SDK** (C++) that decode raw samples into frames.
- **Extensions (Python)**: packaged Python modules that run inside Logic 2, including:
  - **High Level Analyzers (HLAs)**: post-process the output frames from an LLA
  - **Measurements**: compute metrics over analog/digital data ranges

These are complementary: LLAs are “decode engines”, while HLAs/measurements are “post-processing / interpretation”.

## 1) Low-level analyzers (C++ Protocol Analyzer SDK)

### What you build

- A platform-specific shared library that Logic 2 can load as a custom analyzer.
  - Windows: `.dll`
  - Linux: `.so`
  - macOS: docs mention both `.dylib` and `.so` in different contexts

### How you add it to Logic 2

Per Saleae’s “Import Custom Low Level Analyzer” guide:

- Open **Settings**
  - Windows: `Edit → Settings`
  - macOS: `Logic2 → Settings`
  - Ubuntu: `Edit → Settings`
- Scroll to **Custom Low Level Analyzers**
- Click **Browse** and select the directory containing your compiled analyzer library
- **Restart Logic 2** (analyzers are loaded on application start)

Important nuance: Logic 2 does **not** require a fixed global “plugins directory”; you explicitly pick a directory in Settings.

### Default build output locations (SampleAnalyzer-style)

Saleae’s support article lists typical default output locations:

- Windows: `\\build\\Analyzers\\Release\\<Custom Analyzer>.dll`
- macOS: `/build/Analyzers/<Custom Analyzer>.so`
- Ubuntu: `/build/Analyzers/<Custom Analyzer>.so`

Interpretation: these are **paths inside your analyzer project’s build tree**, not “system install paths”.

### When to choose an LLA

- You need to decode a protocol not supported by Saleae’s built-in analyzers.
- You need low-level control over framing/decoding performance.

## 2) Logic 2 Extensions (Python: HLAs + Measurements)

### What you build

Per “Extension File Format”:

- An **extension directory** that contains at least:
  - `extension.json`
  - `readme.md`
  - one or more Python files

Logic 2 uses **Python 3.8** for these extensions.

An extension package can contain multiple entries (HLAs, digital measurements, analog measurements) described under the `extensions` field in `extension.json`.

### `extension.json` anatomy (high signal)

From the official file-format doc:

- Metadata: `version`, `apiVersion`, `author`, `description`, `name`
- A map/object `extensions`:
  - each key is the displayed extension entry name (e.g. `"I2C EEPROM Reader"`)
  - each value includes:
    - `type` (e.g. `HighLevelAnalyzer`, `DigitalMeasurement`)
    - `entryPoint` (module + symbol, e.g. `"I2CUtilities.Eeprom"`)
    - optional fields per type (e.g. measurement `metrics`)

### How you add it to Logic 2

Per “Create and Use Extensions” quickstart:

- Use the **Extensions panel** inside Logic 2:
  - open the extensions panel
  - “Create Extension”
  - choose template (HLA or measurement)
  - “Save As…” to a location on disk
  - the extension appears as **Local**

Additionally, the Extensions overview indicates:

- Logic 2 can browse/install extensions from an **Extensions Marketplace** via the Extensions button in the app
- you can publish your own extension (distribution workflow)

### Relationship to the Automation API (important for our Go CLI)

The automation proto (`AddHighLevelAnalyzerRequest`) expects:

- `extension_directory`: the path to the extension dir containing `extension.json`
- `hla_name`: the HLA name as listed in `extension.json`
- `input_analyzer_id`: the LLA analyzer whose output frames feed the HLA

So, once we implement analyzer + HLA commands in the Go CLI, we can support workflows like:

1. `salad analyzer add` (LLA like SPI/I2C) → `analyzer_id`
2. `salad hla add --extension-dir ... --name ... --input-analyzer-id <analyzer_id>`
3. `salad export table ...` (to extract decoded frames to CSV)

## Practical guidance: choosing which “plugin type” to build

- **Need a brand-new protocol decode from raw samples** → build an **LLA** (C++ SDK).
- **Need to interpret/annotate/filter existing decode output** → build an **HLA** (Python extension).
- **Need numeric metrics over a range** → build a **measurement** (Python extension).

## Recommended next steps (for this repo)

- Add CLI support for:
  - analyzer add/remove (LLA selection + settings)
  - HLA add/remove (extension dir + hla_name + input analyzer id)
  - export table with filtering (works great with HLAs)
- Create a small “example extension” directory in-repo (or in `ttmp/`) as a living reference:
  - `extension.json`
  - minimal HLA python entrypoint

## Sources

- Protocol Analyzer SDK: `https://support.saleae.com/product/user-guide/extensions-apis-and-sdks/saleae-api-and-sdk/protocol-analyzer-sdk`
- Import Custom Low Level Analyzer: `https://support.saleae.com/getting-help/troubleshooting/technical-faq/setting-up-developer-directory`
- Extensions overview: `https://support.saleae.com/product/user-guide/extensions-apis-and-sdks/extensions`
- Create and Use Extensions: `https://support.saleae.com/product/user-guide/extensions-apis-and-sdks/extensions/extensions-quickstart`
- Extension File Format: `https://support.saleae.com/product/user-guide/extensions-apis-and-sdks/extensions/extension-file-format`
- SampleAnalyzer: `https://github.com/saleae/SampleAnalyzer`
