---
Title: Manual smoke test (Logic2 automation)
Ticket: 001-INITIAL-SALAD
Status: active
Topics:
    - go
    - saleae
    - logic-analyzer
    - client
DocType: playbook
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Manual procedure to start Logic2 with automation enabled and validate `salad` CLI commands against the local gRPC API."
LastUpdated: 2025-12-24T22:07:18.06040807-05:00
WhatFor: ""
WhenToUse: ""
---

# Manual smoke test (Logic2 automation)

## Goal

Validate that a running Saleae Logic 2 instance can be controlled via the local gRPC Automation API and that this repo’s `salad` CLI can dial and call basic RPCs (`GetAppInfo`, `GetDevices`).

## Prerequisites

- Saleae Logic 2 installed (e.g. `Logic-2.AppImage`) and runnable on this machine
- This repo builds:

```bash
cd /home/manuel/workspaces/2025-12-21/echo-base-documentation/salad && go test ./...
```

## Step 1: Start Logic 2 with automation enabled

Pick a port (defaults in this repo assume `10430`).

Local-only:

```bash
Logic-2.AppImage --automation --automationPort 10430
```

Remote-access (be careful; exposes the API to the network):

```bash
Logic-2.AppImage --automation --automationHost 0.0.0.0 --automationPort 10430
```

## Step 2: Call `appinfo`

```bash
cd /home/manuel/workspaces/2025-12-21/echo-base-documentation/salad && go run ./cmd/salad appinfo --host 127.0.0.1 --port 10430
```

Expected output (shape):

- `application_version=...`
- `api_version=MAJOR.MINOR.PATCH`
- `launch_pid=...`

## Step 3: List devices

```bash
cd /home/manuel/workspaces/2025-12-21/echo-base-documentation/salad && go run ./cmd/salad devices --host 127.0.0.1 --port 10430
```

If you want to include simulation devices:

```bash
cd /home/manuel/workspaces/2025-12-21/echo-base-documentation/salad && go run ./cmd/salad devices --include-simulation-devices --host 127.0.0.1 --port 10430
```

Expected output (0..N lines):

- `device_id=... device_type=... is_simulation=...`

## Troubleshooting

- **connection refused / deadline exceeded**
  - Confirm Logic 2 is running with `--automation`
  - Confirm the `--automationPort` matches `--port`
  - Try increasing `--timeout 15s`

- **permission/network issues**
  - If using `--automationHost 0.0.0.0`, confirm firewall rules allow the port

## Files involved

- `cmd/salad/cmd/appinfo.go`
- `cmd/salad/cmd/devices.go`
- `internal/saleae/client.go`
