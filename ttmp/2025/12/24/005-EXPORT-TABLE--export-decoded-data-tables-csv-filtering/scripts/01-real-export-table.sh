#!/usr/bin/env bash
set -euo pipefail

# Ticket: 005-EXPORT-TABLE
#
# Goal: Validate `salad export table` against a real Logic 2 automation server.
#
# This script does:
#   1) `salad capture load` (load an existing .sal)
#   2) `salad analyzer add` (using a known-good template YAML)
#   3) `salad export table` (writes a CSV to /tmp)
#   4) verifies the file exists + prints the first lines
#   5) closes the capture
#
# Usage:
#   SAL=/absolute/path/to/your/capture.sal ./01-real-export-table.sh
#
# Optional env overrides:
#   HOST=127.0.0.1
#   PORT=10430
#   TIMEOUT=120s
#   SETTINGS_YAML=/abs/path/to/configs/analyzers/spi.yaml
#   OUT=/tmp/salad-export-table-real.csv
#
# Note: `SAL` must be an absolute path on the machine where Logic 2 is running.

HOST="${HOST:-127.0.0.1}"
PORT="${PORT:-10430}"
TIMEOUT="${TIMEOUT:-120s}"

SAL="${SAL:-}"
if [[ -z "$SAL" ]]; then
  echo "ERROR: set SAL to an absolute path to a .sal capture file to load (e.g. SAL=/tmp/Session\\ 6.sal)" >&2
  exit 2
fi

SETTINGS_YAML="${SETTINGS_YAML:-/home/manuel/workspaces/2025-12-27/salad-pass/salad/configs/analyzers/session6-spi-spi-clk0-mosi1-miso2-cs3-nodeid-10028.yaml}"
OUT="${OUT:-/tmp/salad-export-table-real.csv}"

SALAD="GOWORK=off go run ./cmd/salad --host ${HOST} --port ${PORT} --timeout ${TIMEOUT}"

rm -f "$OUT"

echo "--- appinfo ---"
eval "$SALAD appinfo"

echo "--- capture load ---"
CAPTURE_ID="$(
  eval "$SALAD capture load --filepath '$SAL'" \
    | awk -F= '/^capture_id=/{print $2; exit}'
)"
if [[ -z "$CAPTURE_ID" ]]; then
  echo "ERROR: failed to parse capture_id from capture load output" >&2
  exit 2
fi
echo "capture_id=$CAPTURE_ID"

echo "--- analyzer add (SPI) ---"
ANALYZER_ID="$(
  eval "$SALAD analyzer add --capture-id $CAPTURE_ID --name 'SPI' --label 'real-test' --settings-yaml '$SETTINGS_YAML'" \
    | awk -F= '/^analyzer_id=/{print $2; exit}'
)"
if [[ -z "$ANALYZER_ID" ]]; then
  echo "ERROR: failed to parse analyzer_id from analyzer add output" >&2
  exit 2
fi
echo "analyzer_id=$ANALYZER_ID"

echo "--- export table ---"
eval "$SALAD export table --capture-id $CAPTURE_ID --filepath '$OUT' --analyzer ${ANALYZER_ID}:hex --iso8601-timestamp"

echo "--- verify output file ---"
ls -l "$OUT"
head -n 25 "$OUT"

echo "--- capture close ---"
eval "$SALAD capture close --capture-id $CAPTURE_ID"

echo "ok"


