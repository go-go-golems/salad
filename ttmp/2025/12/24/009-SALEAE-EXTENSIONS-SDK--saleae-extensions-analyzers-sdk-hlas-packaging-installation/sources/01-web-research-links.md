---
Title: Web research links
Ticket: 009-SALEAE-EXTENSIONS-SDK
Status: active
Topics:
    - saleae
    - logic-analyzer
    - extensions
    - sdk
DocType: sources
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Primary web sources on Saleae Logic 2 extensibility: C++ Protocol Analyzer SDK (low-level analyzers) and Python Extensions (HLAs/Measurements) including packaging and installation."
LastUpdated: 2025-12-24T22:45:17.902151226-05:00
WhatFor: ""
WhenToUse: ""
---

# Web research links

## Protocol Analyzer SDK (C++ low-level analyzers)

- **Protocol Analyzer SDK overview**: `https://support.saleae.com/product/user-guide/extensions-apis-and-sdks/saleae-api-and-sdk/protocol-analyzer-sdk`
  - Mentions the SDK used by Saleae internally and points to SampleAnalyzer and import instructions.

- **Import custom low level analyzer (Logic 2 UI)**: `https://support.saleae.com/getting-help/troubleshooting/technical-faq/setting-up-developer-directory`
  - Shows where in Logic 2 to configure loading:
    - Settings → “Custom Low Level Analyzers” → choose directory, then restart Logic 2
  - Shows default build output locations (SampleAnalyzer-style):
    - Windows: `\\build\\Analyzers\\Release\\<Custom Analyzer>.dll`
    - macOS: `/build/Analyzers/<Custom Analyzer>.so` (note: doc also mentions `.dylib` in error section)
    - Ubuntu: `/build/Analyzers/<Custom Analyzer>.so`

- **SampleAnalyzer repo**: `https://github.com/saleae/SampleAnalyzer`
  - Reference implementation for analyzer plugins (C++), intended starting point.

## Logic 2 Extensions (Python: HLAs + Measurements)

- **Extensions overview**: `https://support.saleae.com/product/user-guide/extensions-apis-and-sdks/extensions`
  - Describes extensions as Python modules and references:
    - marketplace browsing/installation from inside the app via the “Extensions” button
    - ability to publish extensions

- **Create and Use Extensions (quickstart)**: `https://support.saleae.com/product/user-guide/extensions-apis-and-sdks/extensions/extensions-quickstart`
  - Shows creating an extension from inside Logic 2:
    - Extensions panel → “Create Extension” → choose template → “Save As…”
    - Newly created extension appears as “Local”

- **Extension File Format**: `https://support.saleae.com/product/user-guide/extensions-apis-and-sdks/extensions/extension-file-format`
  - States: extensions are composed of at least:
    - `extension.json`
    - `readme.md`
    - one or more Python files
  - Notes: Logic 2 uses **Python 3.8**
  - Shows `extension.json` layout:
    - top-level metadata: `version`, `apiVersion`, `author`, `description`, `name`
    - an `extensions` object containing one or more entries:
      - each entry has `type` (e.g. `HighLevelAnalyzer`, `DigitalMeasurement`)
      - and an `entryPoint` like `"I2CUtilities.Eeprom"` (module + symbol)

