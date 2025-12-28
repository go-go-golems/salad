---
Title: Debugging the mock server
Ticket: 010-MOCK-SERVER
Status: active
Topics:
    - go
    - saleae
    - testing
    - mock
    - grpc
DocType: playbook
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/scripts/README.md
      Note: Entry point explaining how to run the repeatable scripts
    - Path: ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/scripts/01-go-check.sh
      Note: Formatting + go test + go vet checks (module-local, uses GOWORK=off)
    - Path: ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/scripts/02-smoke-happy-path.sh
      Note: Happy-path end-to-end smoke test against configs/mock/happy-path.yaml
    - Path: ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/scripts/03-smoke-faults.sh
      Note: Fault injection smoke test against configs/mock/faults.yaml (defaults to port 10432)
    - Path: ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/scripts/09-kill-mock-on-port.sh
      Note: Kill stale listener by port (prevents wrong-server / port collision confusion)
    - Path: ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/scripts/10-tmux-mock-start.sh
      Note: Start salad-mock in tmux with persistent log file in scripts/logs/
    - Path: ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/scripts/11-tmux-mock-stop.sh
      Note: Stop tmux session
    - Path: configs/mock/happy-path.yaml
      Note: Reference scenario used by happy-path smoke test
    - Path: configs/mock/faults.yaml
      Note: Reference scenario used by faults smoke test
    - Path: ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/analysis/05-bug-report-faults-yaml-does-not-inject-savecapture-unavailable-capture-10-not-found.md
      Note: Prior investigation; root cause was port collision / wrong server answering
ExternalSources: []
Summary: "Reusable playbook for debugging salad-mock behavior: ports, tmux lifecycle, log collection, and repeatable smoke tests."
LastUpdated: 2025-12-27T16:13:54.83904524-05:00
WhatFor: "A repeatable procedure to validate and debug the YAML-driven mock server and the salad CLI against it"
WhenToUse: "When scenario behavior looks wrong (faults not firing, wrong fixtures), when ports are stuck, or when collecting logs for a bug report"
---

# Debugging the mock server

## Purpose

This playbook is a repeatable procedure for debugging `salad-mock` (the mock Saleae gRPC server) and validating `salad` CLI behavior against it. The key idea is to eliminate the most common source of confusion—**port collisions leading to the wrong server answering**—by using tmux lifecycle scripts and persistent logs.

## Environment Assumptions

- You are in the `salad-pass` repo workspace.
- `go` is installed.
- `tmux` is installed.
- You can run scripts from:
  - `ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/scripts/`

## Commands

This section is written as a practical “general method” you can follow every time.

```bash
cd salad/ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/scripts

# 0) Always clear ports first (port collisions are the #1 source of false conclusions)
PORT=10431 ./09-kill-mock-on-port.sh
PORT=10432 ./09-kill-mock-on-port.sh

# 1) Run module-local checks (format + compile/vet)
./01-go-check.sh

# 2) Run self-contained smoke tests
./02-smoke-happy-path.sh
./03-smoke-faults.sh

# 3) For iterative debugging, run the server in tmux + tail logs
CFG=configs/mock/happy-path.yaml PORT=10431 SESSION=salad-mock-010 ./10-tmux-mock-start.sh
PORT=10431 ./13-tmux-mock-tail-log.sh
SESSION=salad-mock-010 ./11-tmux-mock-stop.sh
```

## Exit Criteria

- `01-go-check.sh` ends with `ok`.
- `02-smoke-happy-path.sh` ends with `ok` and prints placeholder outputs (`digital.csv`, `analog.csv`, etc.).
- `03-smoke-faults.sh` ends with `ok` and shows:
  - first save: `code = Unavailable desc = temporary mock failure`
  - second save: `ok`
- When using tmux: server logs are present under `scripts/logs/salad-mock-<port>.log`.

## Notes

- If behavior looks like the “wrong YAML”, assume a port collision until proven otherwise.
- Collect evidence for bug reports:
  - tmux log: `scripts/logs/salad-mock-<port>.log`
  - `ss -ltnp | egrep ':(10431|10432)\\b' || true`
  - `ps aux | grep '[s]alad-mock' || true`
