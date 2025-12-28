#!/usr/bin/env bash
set -euo pipefail

TICKET_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

SESSION="${SESSION:-salad-mock-010}"

echo "--- stop ---"
SESSION="$SESSION" "$TICKET_DIR/scripts/11-tmux-mock-stop.sh" || true

echo "--- start ---"
SESSION="$SESSION" "$TICKET_DIR/scripts/10-tmux-mock-start.sh"

