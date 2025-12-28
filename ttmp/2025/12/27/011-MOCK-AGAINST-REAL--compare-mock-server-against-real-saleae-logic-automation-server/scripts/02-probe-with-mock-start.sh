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

# Relative to ${SALAD_DIR}
MOCK_CONFIG="${MOCK_CONFIG:-configs/mock/happy-path.yaml}"

PIDFILE="${PIDFILE:-/tmp/salad-mock-${MOCK_PORT}.pid}"
LOGFILE="${LOGFILE:-/tmp/salad-mock-${MOCK_PORT}.log}"

OUT_FILE="${OUT_FILE:-${TICKET_DIR}/various/probe-$(date +%Y%m%d-%H%M%S).json}"

cd "${SALAD_DIR}"

rm -f "${PIDFILE}"
echo "salad dir:    ${SALAD_DIR}"
echo "mock config:  ${MOCK_CONFIG}"
echo "mock log:     ${LOGFILE}"
echo "writing:      ${OUT_FILE}"
echo "real:         ${REAL_HOST}:${REAL_PORT}"
echo "mock:         ${MOCK_HOST}:${MOCK_PORT}"

(go run ./cmd/salad-mock --config "${MOCK_CONFIG}" --host "${MOCK_HOST}" --port "${MOCK_PORT}" --log-level info >"${LOGFILE}" 2>&1 & echo $! >"${PIDFILE}")
trap 'if [ -f "${PIDFILE}" ]; then kill "$(cat "${PIDFILE}")" >/dev/null 2>&1 || true; fi' EXIT

sleep 0.5

REAL_HOST="${REAL_HOST}" REAL_PORT="${REAL_PORT}" \
MOCK_HOST="${MOCK_HOST}" MOCK_PORT="${MOCK_PORT}" \
TIMEOUT="${TIMEOUT}" OUT_FILE="${OUT_FILE}" \
bash "${SCRIPT_DIR}/01-probe-real-vs-mock.sh"


