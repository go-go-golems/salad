#!/usr/bin/env bash
set -euo pipefail

# Ticket: 006-PIPELINES
#
# Goal: Validate `salad run --config ...` against a real Logic 2 automation server.
#
# This script generates a small pipeline config in /tmp and executes it.
#
# Usage:
#   SAL="/tmp/Session 6.sal" ./02-real-run.sh
#
# Optional env overrides:
#   HOST=127.0.0.1
#   PORT=10430
#   TIMEOUT=120s
#   SETTINGS_YAML=configs/analyzers/spi.yaml
#   OUTDIR=/tmp/salad-pipeline-006-real

HOST="${HOST:-127.0.0.1}"
PORT="${PORT:-10430}"
TIMEOUT="${TIMEOUT:-120s}"

SAL="${SAL:-}"
if [[ -z "$SAL" ]]; then
  echo "ERROR: set SAL to an absolute path to a .sal capture file to load (e.g. SAL=/tmp/Session\\ 6.sal)" >&2
  exit 2
fi

SETTINGS_YAML="${SETTINGS_YAML:-configs/analyzers/session6-spi-spi-clk0-mosi1-miso2-cs3-nodeid-10028.yaml}"
OUTDIR="${OUTDIR:-/tmp/salad-pipeline-006-real}"

CFG="/tmp/salad-pipeline-006-real.yaml"

cat >"$CFG" <<EOF
version: 1
capture:
  load:
    filepath: ${SAL}
analyzers:
  - name: "SPI"
    label: "spi"
    settings_yaml: "${SETTINGS_YAML}"
exports:
  - type: raw-csv
    directory: "${OUTDIR}/raw"
    digital: [0, 1, 2]
    iso8601_timestamp: true
  - type: table-csv
    filepath: "${OUTDIR}/table.csv"
    iso8601_timestamp: true
    analyzers:
      - ref: "spi"
        radix: hex
cleanup:
  close_capture: true
EOF

echo "--- pipeline config ---"
echo "config=${CFG}"

echo "--- run pipeline ---"
cd "$(git rev-parse --show-toplevel)"
GOWORK=off go run ./cmd/salad --host "${HOST}" --port "${PORT}" --timeout "${TIMEOUT}" run --config "$CFG"


