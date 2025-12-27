#!/usr/bin/env bash
set -euo pipefail

TICKET_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

PORT="${PORT:-10431}"

echo "port=$PORT"

if ! command -v ss >/dev/null 2>&1; then
  echo "ss not found in PATH"
  exit 1
fi

# Parse pid=... from ss output.
lines="$(ss -ltnp 2>/dev/null | grep ":$PORT" || true)"
if [[ -z "$lines" ]]; then
  echo "no listener found on :$PORT"
  exit 0
fi

echo "$lines"

pids="$(echo "$lines" | grep -oE 'pid=[0-9]+' | cut -d= -f2 | sort -u)"
if [[ -z "$pids" ]]; then
  echo "could not parse pid(s) from ss output; attempting pkill fallback"
  if command -v pkill >/dev/null 2>&1; then
    pkill -f "salad-mock.*--port ${PORT}" || true
    sleep 0.2
    ss -ltnp 2>/dev/null | grep ":$PORT" || true
    echo "ok"
    exit 0
  fi
  echo "pkill not found; cannot proceed"
  exit 1
fi

echo "killing pids: $pids"
for pid in $pids; do
  kill "$pid" || true
done

sleep 0.2
ss -ltnp 2>/dev/null | grep ":$PORT" || true
echo "ok"


