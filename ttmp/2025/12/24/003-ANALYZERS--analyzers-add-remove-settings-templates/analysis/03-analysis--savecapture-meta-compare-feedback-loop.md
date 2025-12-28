---
Title: "Analysis: SaveCapture → meta.json → compare — a feedback loop for validating analyzer templates"
Ticket: 003-ANALYZERS
Status: active
Topics:
  - saleae
  - logic-analyzer
  - go
  - client
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
  - Path: /home/manuel/workspaces/2025-12-27/salad-pass/salad/proto/saleae/grpc/saleae.proto
    Note: Defines `SaveCapture`, `AddAnalyzer`, and the analyzer settings value types.
  - Path: /home/manuel/workspaces/2025-12-27/salad-pass/salad/cmd/salad/cmd/capture.go
    Note: `salad capture save` uses SaveCapture and requires an absolute filepath on the Logic host.
  - Path: /home/manuel/workspaces/2025-12-27/salad-pass/salad/internal/config/analyzer_settings.go
    Note: Parses analyzer template settings files into the proto settings map.
  - Path: /home/manuel/workspaces/2025-12-27/salad-pass/salad/ttmp/2025/12/24/003-ANALYZERS--analyzers-add-remove-settings-templates/scripts/02-extract-analyzer-settings-from-meta-json.py
    Note: Converts meta.json analyzers into `settings:` YAML blocks (dropdownText by default).
  - Path: /home/manuel/workspaces/2025-12-27/salad-pass/salad/ttmp/2025/12/24/003-ANALYZERS--analyzers-add-remove-settings-templates/scripts/03-compare-meta-json-to-template/main.go
    Note: Compares one analyzer’s meta.json settings against a template file.
ExternalSources: []
Summary: "Describes a practical, automatable feedback loop: apply analyzer template → SaveCapture → extract meta.json → compare to expected, including remote-host filesystem constraints."
LastUpdated: 2025-12-27T00:00:00Z
WhatFor: ""
WhenToUse: ""
---

# SaveCapture → meta.json → compare — a feedback loop for validating analyzer templates

## Why this approach exists

The Saleae Automation gRPC API lets us **set analyzer settings only at creation time** (`AddAnalyzerRequest.settings`) but provides **no RPC** to:
- enumerate setting schemas/options, or
- read back the current settings for an analyzer configured in Logic 2.

However, a saved `.sal` contains `meta.json` which includes:
- analyzer list (type/name/nodeId),
- each setting key (UI title),
- the selected value, and
- for dropdowns, the full list of options including the **UI-visible string** (`dropdownText`).

This means we can build a robust “feedback loop” for template correctness.

## The feedback loop (conceptual)

1. **Apply template**: `salad analyzer add --settings-yaml <template>`
2. **Save capture**: `salad capture save --capture-id <id> --filepath /abs/path/to/out.sal`
3. **Extract meta**: unzip `.sal` and read `meta.json`
4. **Normalize** meta settings into the same “settings map” representation we use in templates
5. **Compare** normalized meta settings to the expected template (or to a “golden” canonical config)

## What you can validate with this loop

- **Key correctness**: are we using the exact UI label strings? (spaces/punctuation/case)
- **Dropdown correctness**: did we choose the intended option string (CPOL/CPHA, bits per transfer, etc.)?
- **Defaulting behavior**: what settings does Logic fill in when you omit some keys?
- **Cross-version drift**: do keys/options change between Logic versions?

## Where the data comes from (and what it means)

### AddAnalyzer settings types (gRPC)

`AddAnalyzerRequest.settings` is a `map<string, AnalyzerSettingValue>` where values are scalar:
- string
- int64
- bool
- double

This matches our settings file support (`internal/config/analyzer_settings.go`).

### meta.json settings types (Logic UI state)

`meta.json` includes richer UI-specific types (observed examples):
- `Channel` with numeric channel index
- `NumberList` with `options[]` including:
  - numeric option code (`value`)
  - UI-visible string (`dropdownText`)

For template validation we normalize dropdown selections to **dropdownText strings**, because:
- it matches Saleae’s own automation docs example style,
- it is stable/readable, and
- it avoids relying on internal numeric codes.

## Automation feasibility and constraints

### Can we save a session file “remotely”?

**Yes, but only onto the filesystem of the machine running Logic 2.**

`SaveCaptureRequest.filepath` is an absolute path interpreted by Logic 2 (the gRPC server). The automation API does not stream file contents back.

So if your Logic 2 automation server is remote:
- `salad capture save --filepath /tmp/out.sal` writes `/tmp/out.sal` on the **remote** host.
- You still need a way to retrieve it for analysis:
  - shared filesystem (NFS/SMB/SSHFS), or
  - out-of-band copy (`scp`, `rsync`), or
  - run the extraction/comparison on the same host.

### Can we close the loop fully automatically?

**Yes, if the `.sal` is locally readable by the code doing extraction.**

Two common deployment models:
- **Local**: Logic 2 + salad + extraction scripts all on the same machine → simplest.
- **Remote Logic**: salad connects remotely → you must arrange file access for the saved `.sal`.

## Recommended “golden master” workflow

### 1) Author or generate a template

- Either manually write `configs/analyzers/<name>.yaml`, or
- generate from a known-good UI session using `meta.json`:
  - `scripts/02-extract-analyzer-settings-from-meta-json.py`

### 2) Apply to a fresh capture and record the analyzer_id

```bash
salad analyzer add --capture-id <id> --name "SPI" --settings-yaml /abs/template.yaml
# => analyzer_id=12345
```

### 3) Save capture and extract meta.json

```bash
salad capture save --capture-id <id> --filepath /abs/out.sal
```

Extract meta.json (Python zipfile is the most portable):

```bash
python - <<'PY'
import sys, zipfile
with zipfile.ZipFile(sys.argv[1]) as z:
    b = z.read("meta.json")
open(sys.argv[2], "wb").write(b)
print("wrote", sys.argv[2])
PY /abs/out.sal /tmp/meta.json
```

### 4) Compare template vs meta.json

Use the comparator script:

```bash
go run ./ttmp/2025/12/24/003-ANALYZERS--analyzers-add-remove-settings-templates/scripts/03-compare-meta-json-to-template/main.go \
  --meta /tmp/meta.json \
  --node-id 12345 \
  --template /abs/template.yaml
```

Interpretation:
- `ok` means the analyzer settings in the saved session match the template after normalization.
- a diff indicates:
  - the template missed defaults (meta has more keys), or
  - a setting value differs (wrong string / wrong channel), or
  - the analyzer schema changed (keys renamed).

## Design notes / pitfalls

- **Meta includes defaults**: `meta.json` typically shows *all* settings (including defaults), while templates often specify only a subset. Decide up front:
  - strict compare (template must include everything), or
  - subset compare (only assert keys present in template).

  The included `03-compare...` script currently does a strict union diff (it will show meta-only keys).

- **Non-gRPC fields**: meta has fields like `showInDataTable` and `streamToTerminal` that are UI-level and not settable via `AddAnalyzerRequest.settings`. Ignore them in comparisons.

- **Capture lifecycle**: saving/closing requires stable capture state. In practice:
  - stop capture,
  - then save,
  - then close.

- **Format drift risk**: `.sal` and `meta.json` are not the gRPC contract. Treat them as a pragmatic tool, and be ready to revalidate on Logic upgrades.

## When this is worth it

This loop is most valuable when:
- you’re building a library of templates for many analyzers,
- you care about exact dropdown selections,
- you need reproducibility across machines/teams, or
- you want “known-good” configurations under version control.


