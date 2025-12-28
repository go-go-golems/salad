#!/usr/bin/env bash
set -euo pipefail

TICKET_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SALAD_DIR="$(cd "$TICKET_DIR/../../../../.." && pwd)"

cd "$SALAD_DIR"
echo "cwd=$(pwd)"

PORT="${PORT:-10431}"
HOST="${HOST:-127.0.0.1}"
CFG="${CFG:-configs/mock/happy-path.yaml}"

tmpdir="$(mktemp -d)"
tmpfile="$(mktemp /tmp/salad-mock-XXXXXX.sal)"

cleanup() {
  if [[ -n "${mock_pid:-}" ]]; then
    kill "$mock_pid" >/dev/null 2>&1 || true
    wait "$mock_pid" >/dev/null 2>&1 || true
  fi
  rm -rf "$tmpdir" >/dev/null 2>&1 || true
  rm -f "$tmpfile" >/dev/null 2>&1 || true
}
trap cleanup EXIT

echo "tmpdir=$tmpdir"
echo "tmpfile=$tmpfile"

echo "--- start salad-mock ---"
GOWORK=off go run ./cmd/salad-mock --config "$CFG" --host "$HOST" --port "$PORT" >"$tmpdir/mock.log" 2>&1 &
mock_pid=$!
sleep 0.4

echo "--- salad appinfo ---"
GOWORK=off go run ./cmd/salad --host "$HOST" --port "$PORT" appinfo

echo "--- salad devices ---"
GOWORK=off go run ./cmd/salad --host "$HOST" --port "$PORT" devices

echo "--- salad capture load ---"
cid="$(GOWORK=off go run ./cmd/salad --host "$HOST" --port "$PORT" capture load --filepath "$tmpfile" | awk -F= '/capture_id=/{print $2}')"
echo "capture_id=$cid"
test -n "$cid"

echo "--- salad capture wait ---"
GOWORK=off go run ./cmd/salad --host "$HOST" --port "$PORT" capture wait --capture-id "$cid"

echo "--- salad capture save ---"
GOWORK=off go run ./cmd/salad --host "$HOST" --port "$PORT" capture save --capture-id "$cid" --filepath "$tmpdir/out.sal"

echo "--- salad export raw-csv ---"
GOWORK=off go run ./cmd/salad --host "$HOST" --port "$PORT" export raw-csv \
  --capture-id "$cid" \
  --directory "$tmpdir" \
  --digital "0,1" \
  --analog "0" \
  --analog-downsample-ratio 1 \
  --iso8601-timestamp

echo "--- salad export raw-binary ---"
GOWORK=off go run ./cmd/salad --host "$HOST" --port "$PORT" export raw-binary \
  --capture-id "$cid" \
  --directory "$tmpdir" \
  --digital "0,1" \
  --analog "0" \
  --analog-downsample-ratio 1

echo "--- outputs ---"
ls -la "$tmpdir"
echo "--- digital.csv ---"
sed -n '1,50p' "$tmpdir/digital.csv" || true
echo "--- analog.csv ---"
sed -n '1,50p' "$tmpdir/analog.csv" || true

echo "--- salad capture close ---"
GOWORK=off go run ./cmd/salad --host "$HOST" --port "$PORT" capture close --capture-id "$cid"

echo "--- mock log ---"
sed -n '1,200p' "$tmpdir/mock.log" || true

echo "ok"


