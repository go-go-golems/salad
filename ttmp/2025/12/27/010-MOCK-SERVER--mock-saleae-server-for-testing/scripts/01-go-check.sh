#!/usr/bin/env bash
set -euo pipefail

TICKET_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SALAD_DIR="$(cd "$TICKET_DIR/../../../../.." && pwd)"

cd "$SALAD_DIR"

echo "cwd=$(pwd)"
echo "go=$(go version)"

echo "--- gofmt (check) ---"
# Only check formatting for the salad code paths relevant to the mock server + CLI.
# (The repo contains other unrelated Go code that may not be kept gofmt-clean.)
unformatted="$(
  find \
    cmd/salad \
    cmd/salad-mock \
    internal/mock/saleae \
    internal/saleae \
    -name '*.go' -print0 \
  | xargs -0 gofmt -l
)"
if [[ -n "$unformatted" ]]; then
  echo "Unformatted files:"
  echo "$unformatted"
  exit 1
fi

echo "--- go test ---"
GOWORK=off go test ./... -count=1

echo "--- go vet ---"
GOWORK=off go vet ./...

echo "ok"


