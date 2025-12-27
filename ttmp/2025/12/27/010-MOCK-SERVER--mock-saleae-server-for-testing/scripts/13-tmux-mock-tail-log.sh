#!/usr/bin/env bash
set -euo pipefail

TICKET_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOG_DIR="$TICKET_DIR/scripts/logs"

PORT="${PORT:-10431}"
LOG_FILE="${LOG_FILE:-$LOG_DIR/salad-mock-${PORT}.log}"

echo "tail -f $LOG_FILE"
tail -f "$LOG_FILE"


