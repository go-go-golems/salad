#!/usr/bin/env bash
set -euo pipefail

TICKET_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

SESSION="${SESSION:-salad-mock-010}"

echo "session=$SESSION"

if ! command -v tmux >/dev/null 2>&1; then
  echo "tmux not found in PATH"
  exit 1
fi

if ! tmux has-session -t "$SESSION" >/dev/null 2>&1; then
  echo "no tmux session found: $SESSION"
  exit 0
fi

tmux kill-session -t "$SESSION"
echo "ok: stopped tmux session $SESSION"

