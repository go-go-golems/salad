#!/usr/bin/env bash
set -euo pipefail

# Ticket: 006-PIPELINES
#
# Goal: Manual, repeatable validation of `salad run --config ...` against `salad-mock`.
#
# This script intentionally reuses the battle-tested mock-server tmux lifecycle scripts from ticket 010:
# - kill port collisions
# - start mock in tmux with persistent logs
# - stop mock cleanly
#
# Usage (from salad repo root):
#   ./ttmp/2025/12/24/006-PIPELINES--pipelines-run-repro-watch-workflows/scripts/03-run-against-mock.sh
#
# Optional env overrides:
#   PORT=10431
#   SESSION=salad-mock-006
#   TIMEOUT=10s

PORT="${PORT:-10431}"
SESSION="${SESSION:-salad-mock-006}"
TIMEOUT="${TIMEOUT:-10s}"

ROOT="$(git rev-parse --show-toplevel)"

SCRIPTS_010="$ROOT/ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/scripts"
CFG_006="$ROOT/ttmp/2025/12/24/006-PIPELINES--pipelines-run-repro-watch-workflows/scripts/00-mock-happy-path-with-table.yaml"
PIPE_006="$ROOT/ttmp/2025/12/24/006-PIPELINES--pipelines-run-repro-watch-workflows/scripts/01-pipeline-mock.yaml"

echo "--- clear port (avoid wrong-server collisions) ---"
PORT="$PORT" "$SCRIPTS_010/09-kill-mock-on-port.sh" || true

echo "--- stop existing tmux session (if any) ---"
SESSION="$SESSION" "$SCRIPTS_010/11-tmux-mock-stop.sh" || true

echo "--- start salad-mock in tmux ---"
CFG="$CFG_006" PORT="$PORT" SESSION="$SESSION" "$SCRIPTS_010/10-tmux-mock-start.sh"

echo "--- run pipeline ---"
rm -rf /tmp/salad-pipeline-006 || true
GOWORK=off go run ./cmd/salad --host 127.0.0.1 --port "$PORT" --timeout "$TIMEOUT" \
  run --config "$PIPE_006"

echo "--- verify artifacts ---"
ls -la /tmp/salad-pipeline-006 /tmp/salad-pipeline-006/raw /tmp/salad-pipeline-006/table.csv
echo "--- /tmp/salad-pipeline-006/raw/digital.csv (head) ---"
head -n 25 /tmp/salad-pipeline-006/raw/digital.csv
echo "--- /tmp/salad-pipeline-006/table.csv (head) ---"
head -n 25 /tmp/salad-pipeline-006/table.csv

echo "--- stop salad-mock ---"
SESSION="$SESSION" "$SCRIPTS_010/11-tmux-mock-stop.sh"

echo "ok"


