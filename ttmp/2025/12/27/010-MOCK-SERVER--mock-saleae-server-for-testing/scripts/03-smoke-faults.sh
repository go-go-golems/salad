#!/usr/bin/env bash
set -euo pipefail

TICKET_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SALAD_DIR="$(cd "$TICKET_DIR/../../../../.." && pwd)"

cd "$SALAD_DIR"
echo "cwd=$(pwd)"

PORT="${PORT:-10432}"
HOST="${HOST:-127.0.0.1}"
CFG="${CFG:-configs/mock/faults.yaml}"

tmpdir="$(mktemp -d)"

cleanup() {
  if [[ -n "${mock_pid:-}" ]]; then
    kill "$mock_pid" >/dev/null 2>&1 || true
    wait "$mock_pid" >/dev/null 2>&1 || true
  fi
  echo "--- mock log (first 250 lines) ---"
  sed -n '1,250p' "$tmpdir/mock.log" || true
  rm -rf "$tmpdir" >/dev/null 2>&1 || true
}
trap cleanup EXIT

echo "tmpdir=$tmpdir"
echo "cfg=$CFG host=$HOST port=$PORT"

echo "--- start salad-mock (faults scenario) ---"
GOWORK=off go run ./cmd/salad-mock --config "$CFG" --host "$HOST" --port "$PORT" >"$tmpdir/mock.log" 2>&1 &
mock_pid=$!
sleep 0.4

echo "--- Expected behavior: SaveCapture fails on 1st call with UNAVAILABLE, succeeds on 2nd call (unless configured otherwise) ---"

echo "--- salad capture save (1st call, expect failure) ---"
set +e
GOWORK=off go run ./cmd/salad --host "$HOST" --port "$PORT" capture save --capture-id 10 --filepath "$tmpdir/out1.sal"
rc=$?
set -e
echo "exit_code=$rc"
if [[ "$rc" -eq 0 ]]; then
  echo "expected non-zero exit code for first SaveCapture call"
  exit 1
fi

echo "--- salad capture save (2nd call, expect success) ---"
GOWORK=off go run ./cmd/salad --host "$HOST" --port "$PORT" capture save --capture-id 10 --filepath "$tmpdir/out2.sal"

echo "ok"


