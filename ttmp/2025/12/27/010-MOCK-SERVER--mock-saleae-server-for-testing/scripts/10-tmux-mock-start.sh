#!/usr/bin/env bash
set -euo pipefail

TICKET_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SALAD_DIR="$(cd "$TICKET_DIR/../../../../.." && pwd)"
LOG_DIR="$TICKET_DIR/scripts/logs"

SESSION="${SESSION:-salad-mock-010}"
CFG="${CFG:-configs/mock/happy-path.yaml}"
HOST="${HOST:-127.0.0.1}"
PORT="${PORT:-10431}"
LOG_LEVEL="${LOG_LEVEL:-debug}"
LOG_FILE="${LOG_FILE:-$LOG_DIR/salad-mock-${PORT}.log}"

echo "ticket_dir=$TICKET_DIR"
echo "salad_dir=$SALAD_DIR"
echo "session=$SESSION"
echo "cfg=$CFG host=$HOST port=$PORT log_level=$LOG_LEVEL"
echo "log_file=$LOG_FILE"

if ! command -v tmux >/dev/null 2>&1; then
  echo "tmux not found in PATH"
  exit 1
fi

if tmux has-session -t "$SESSION" >/dev/null 2>&1; then
  echo "tmux session already exists: $SESSION"
  echo "Use: SESSION=$SESSION $TICKET_DIR/scripts/11-tmux-mock-stop.sh"
  exit 1
fi

# Fail fast if the port is already in use (common when a previous salad-mock is still running).
if ss -ltnp 2>/dev/null | grep -q ":$PORT"; then
  echo "port already in use: $PORT"
  ss -ltnp 2>/dev/null | grep ":$PORT" || true
  echo "Consider: PORT=$PORT $TICKET_DIR/scripts/09-kill-mock-on-port.sh"
  exit 1
fi

# Start in tmux and tee output to a log file.
tmux new-session -d -s "$SESSION" \
  "cd \"$SALAD_DIR\" && echo \"--- started \$(date -Iseconds) cfg=$CFG host=$HOST port=$PORT ---\" >> \"$LOG_FILE\" && GOWORK=off go run ./cmd/salad-mock --config \"$CFG\" --host \"$HOST\" --port \"$PORT\" --log-level \"$LOG_LEVEL\" 2>&1 | tee -a \"$LOG_FILE\""

echo "ok: started tmux session $SESSION"
echo "tail: tail -f \"$LOG_FILE\""

