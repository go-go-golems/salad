# Scripts (ticket 006)

## Mock-server pipeline validation

This is the fastest loop for developing pipelines: run `salad run` against `salad-mock` with a deterministic scenario that writes placeholder exports.

### 1) Start the mock server (tmux)

From the `salad/` repo root:

```bash
CFG="ttmp/2025/12/24/006-PIPELINES--pipelines-run-repro-watch-workflows/scripts/00-mock-happy-path-with-table.yaml" \
PORT=10431 SESSION=salad-mock-006 \
./ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/scripts/10-tmux-mock-start.sh
```

If you hit port collisions, use the mock-server playbook:
- `ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/playbook/01-debugging-the-mock-server.md`

### 2) Run the pipeline against the mock server

```bash
GOWORK=off go run ./cmd/salad --host 127.0.0.1 --port 10431 --timeout 10s \
  run --config ttmp/2025/12/24/006-PIPELINES--pipelines-run-repro-watch-workflows/scripts/01-pipeline-mock.yaml
```

Expected:
- `capture_id=...`
- at least one `analyzer_id=... label="spi"`
- `artifact=/tmp/salad-pipeline-006/raw`
- `artifact=/tmp/salad-pipeline-006/table.csv`
- final `ok`


