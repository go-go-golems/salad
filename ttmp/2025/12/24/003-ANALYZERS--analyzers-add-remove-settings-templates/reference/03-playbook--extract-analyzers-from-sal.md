---
Title: "Playbook: Extract analyzer settings from .sal (meta.json) and transform into templates"
Ticket: 003-ANALYZERS
Status: active
Topics:
  - saleae
  - logic-analyzer
  - go
  - client
DocType: playbook
Intent: long-term
Owners: []
RelatedFiles:
  - Path: /home/manuel/workspaces/2025-12-27/salad-pass/salad/cmd/salad/cmd/capture.go
    Note: `salad capture save` writes a .sal file on the Logic host filesystem via SaveCapture RPC.
  - Path: /home/manuel/workspaces/2025-12-27/salad-pass/salad/proto/saleae/grpc/saleae.proto
    Note: SaveCapture/AddAnalyzer contracts; analyzer settings are passed only at creation time.
  - Path: /home/manuel/workspaces/2025-12-27/salad-pass/salad/ttmp/2025/12/24/003-ANALYZERS--analyzers-add-remove-settings-templates/scripts/02-extract-analyzer-settings-from-meta-json.py
    Note: Lists analyzers and emits a `settings:` YAML block from meta.json.
  - Path: /home/manuel/workspaces/2025-12-27/salad-pass/salad/configs/analyzers/
    Note: Where we store reusable analyzer settings templates.
ExternalSources: []
Summary: "Hands-on procedure: UI-configure analyzers → save .sal → extract meta.json → generate YAML templates and apply them with salad."
LastUpdated: 2025-12-27T00:00:00Z
WhatFor: ""
WhenToUse: ""
---

# Playbook: Extract analyzer settings from .sal (meta.json) and transform into templates

## Goal

Turn a known-good analyzer configuration created in the Logic 2 UI into a reusable `salad` template:

1. Configure analyzer(s) in Logic 2 UI
2. Save session/capture as a `.sal`
3. Extract `meta.json`
4. Convert analyzer settings into YAML templates (`settings:` blocks)
5. Re-apply those templates to new captures via `salad analyzer add`

This is the best way to get **exact dropdown strings** (“Bits per Transfer”, CPOL/CPHA text, etc.) without guessing.

## Prerequisites

- Logic 2 running with Automation server enabled (default `127.0.0.1:10430`)
- `salad` CLI works against the server
- A `.sal` file that you saved from Logic 2 UI, or that you saved via `salad capture save`

## Step 1: Save a `.sal` session file

### Option A: Save from the Logic 2 UI

Use the normal “Save” flow in Logic 2 and produce a `Session X.sal`.

### Option B: Save via automation (`SaveCapture` RPC)

This writes the file on the **Logic host filesystem** (see remote notes below).

```bash
salad capture save --capture-id <id> --filepath /tmp/my-session.sal
```

## Step 2: Extract `meta.json` from the `.sal`

`.sal` behaves like a zip archive; `meta.json` is inside it.

Use Python (no `unzip` dependency):

```bash
python - <<'PY'
import sys, zipfile
sal_path = sys.argv[1]
out_path = sys.argv[2]
with zipfile.ZipFile(sal_path) as z:
    with z.open("meta.json") as f:
        b = f.read()
with open(out_path, "wb") as f:
    f.write(b)
print(f"wrote {out_path} from {sal_path}")
PY /abs/path/to/Session.sal /tmp/meta.json
```

Sanity check:

```bash
python -c 'import json; import sys; json.load(open("/tmp/meta.json")); print("meta.json ok")'
```

## Step 3: List analyzers present in `meta.json`

```bash
python /home/manuel/workspaces/2025-12-27/salad-pass/salad/ttmp/2025/12/24/003-ANALYZERS--analyzers-add-remove-settings-templates/scripts/02-extract-analyzer-settings-from-meta-json.py \
  --meta /tmp/meta.json --list
```

You’ll see rows like:

- `nodeId=10028 type='SPI' name='SPI: CLK0 MOSI1 MISO2 CS3'`

**Note:** `nodeId` usually matches the `analyzer_id` that automation returns.

## Step 4: Extract one analyzer into a YAML template

```bash
python /home/manuel/workspaces/2025-12-27/salad-pass/salad/ttmp/2025/12/24/003-ANALYZERS--analyzers-add-remove-settings-templates/scripts/02-extract-analyzer-settings-from-meta-json.py \
  --meta /tmp/meta.json --node-id 10028 --format yaml \
  > /home/manuel/workspaces/2025-12-27/salad-pass/salad/configs/analyzers/spi-from-ui.yaml
```

This emits dropdowns as **UI-visible strings** by default (recommended, matches Saleae docs).

## Step 5: Transform templates (common patterns)

### Pattern A: “Full” template

Keep all extracted settings (channels + dropdown choices). This is best for reproducibility.

### Pattern B: “Minimal channels-only” template

Delete everything except channel assignments:

```yaml
settings:
  Clock: 0
  MOSI: 1
  MISO: 2
  Enable: 3
```

Then apply CPOL/CPHA etc via typed overrides (or a second template variant).

### Pattern C: Variant templates (CPOL/CPHA)

Duplicate the file and change:
- `Clock State` (CPOL)
- `Clock Phase` (CPHA)

Naming convention suggestion:
- `spi-mode0.yaml` (CPOL=0, CPHA=0)
- `spi-mode1.yaml` (CPOL=0, CPHA=1)
- `spi-mode2.yaml` (CPOL=1, CPHA=0)
- `spi-mode3.yaml` (CPOL=1, CPHA=1)

## Step 6: Apply the template to a new capture and validate in Logic 2

1) Start a fresh capture (ensure channels exist):

```bash
go run /home/manuel/workspaces/2025-12-27/salad-pass/salad/ttmp/2025/12/24/003-ANALYZERS--analyzers-add-remove-settings-templates/scripts/01-real-start-capture.go \
  --host 127.0.0.1 --port 10430 --timeout 8s --digital 0,1,2,3
```

2) Add analyzer using the template:

```bash
salad analyzer add --capture-id <id> --name "SPI" --label "spi-from-template" \
  --settings-yaml /home/manuel/workspaces/2025-12-27/salad-pass/salad/configs/analyzers/spi-from-ui.yaml
```

3) Verify in the Logic 2 UI analyzer settings dialog that:
- keys exist and match
- dropdown selections are correct

## Remote / filesystem notes (important)

`salad capture save --filepath ...` calls `SaveCapture` on the Logic 2 automation server:
- the `filepath` is interpreted on the **machine running Logic 2**, not on your local machine
- the gRPC API does **not** provide a “download file” method

So for remote hosts you need one of:
- a shared filesystem path (NFS/SMB/SSHFS) visible to both machines
- an out-of-band fetch step (e.g. `scp` from the Logic host)


