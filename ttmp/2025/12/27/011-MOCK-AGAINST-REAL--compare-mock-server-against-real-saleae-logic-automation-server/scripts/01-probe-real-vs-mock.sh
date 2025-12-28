#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TICKET_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# This script lives under salad/ttmp/... so 6 levels up is the salad module root.
SALAD_DIR="$(cd "${SCRIPT_DIR}/../../../../../.." && pwd)"

REAL_HOST="${REAL_HOST:-127.0.0.1}"
REAL_PORT="${REAL_PORT:-10430}"
MOCK_HOST="${MOCK_HOST:-127.0.0.1}"
MOCK_PORT="${MOCK_PORT:-10431}"
TIMEOUT="${TIMEOUT:-5s}"

OUT_FILE="${OUT_FILE:-${TICKET_DIR}/various/probe-$(date +%Y%m%d-%H%M%S).json}"

cd "${SALAD_DIR}"

echo "salad dir: ${SALAD_DIR}"
echo "writing:   ${OUT_FILE}"
echo "real:      ${REAL_HOST}:${REAL_PORT}"
echo "mock:      ${MOCK_HOST}:${MOCK_PORT}"

go run "${SCRIPT_DIR}/probe_real_vs_mock.go" \
  --real-host "${REAL_HOST}" --real-port "${REAL_PORT}" \
  --mock-host "${MOCK_HOST}" --mock-port "${MOCK_PORT}" \
  --timeout "${TIMEOUT}" \
  --out "${OUT_FILE}"


