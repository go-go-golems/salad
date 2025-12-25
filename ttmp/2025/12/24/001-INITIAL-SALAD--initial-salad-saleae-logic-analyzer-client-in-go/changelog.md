# Changelog

## 2025-12-24

- Initial workspace created


## 2025-12-24

Created build-plan analysis doc, diary, and seeded tasks; linked key docs from index.

### Related Files

- /home/manuel/workspaces/2025-12-21/echo-base-documentation/salad/ttmp/2025/12/24/001-INITIAL-SALAD--initial-salad-saleae-logic-analyzer-client-in-go/analysis/01-build-plan-saleae-logic2-grpc-client-go.md — Initial architecture and proto strategy
- /home/manuel/workspaces/2025-12-21/echo-base-documentation/salad/ttmp/2025/12/24/001-INITIAL-SALAD--initial-salad-saleae-logic-analyzer-client-in-go/index.md — Index now links to research/build plan/diary
- /home/manuel/workspaces/2025-12-21/echo-base-documentation/salad/ttmp/2025/12/24/001-INITIAL-SALAD--initial-salad-saleae-logic-analyzer-client-in-go/reference/01-diary.md — Step 1 diary entry
- /home/manuel/workspaces/2025-12-21/echo-base-documentation/salad/ttmp/2025/12/24/001-INITIAL-SALAD--initial-salad-saleae-logic-analyzer-client-in-go/tasks.md — Added concrete tasks


## 2025-12-24

Vendored saleae.proto at pinned upstream commit and generated Go bindings into gen/ (resolved protoc-gen-go-grpc module path mismatch).

### Related Files

- /home/manuel/workspaces/2025-12-21/echo-base-documentation/salad/gen/saleae/automation/saleae.pb.go — Generated protobuf types
- /home/manuel/workspaces/2025-12-21/echo-base-documentation/salad/gen/saleae/automation/saleae_grpc.pb.go — Generated gRPC stubs
- /home/manuel/workspaces/2025-12-21/echo-base-documentation/salad/proto/saleae/grpc/README.md — Pin recorded
- /home/manuel/workspaces/2025-12-21/echo-base-documentation/salad/proto/saleae/grpc/UPSTREAM_COMMIT_SHA.txt — Pinned upstream SHA
- /home/manuel/workspaces/2025-12-21/echo-base-documentation/salad/proto/saleae/grpc/saleae.proto — Vendored proto from Saleae upstream
- /home/manuel/workspaces/2025-12-21/echo-base-documentation/salad/ttmp/2025/12/24/001-INITIAL-SALAD--initial-salad-saleae-logic-analyzer-client-in-go/reference/01-diary.md — Step 2 diary entry


## 2025-12-24

Implemented initial Cobra CLI (cmd/salad) + internal gRPC client wrapper; added appinfo command and pinned Go deps.

### Related Files

- /home/manuel/workspaces/2025-12-21/echo-base-documentation/salad/cmd/salad/cmd/appinfo.go — appinfo RPC command
- /home/manuel/workspaces/2025-12-21/echo-base-documentation/salad/cmd/salad/cmd/root.go — Root command
- /home/manuel/workspaces/2025-12-21/echo-base-documentation/salad/cmd/salad/main.go — CLI entry point
- /home/manuel/workspaces/2025-12-21/echo-base-documentation/salad/go.mod — Module and deps
- /home/manuel/workspaces/2025-12-21/echo-base-documentation/salad/go.sum — Dependency lock
- /home/manuel/workspaces/2025-12-21/echo-base-documentation/salad/internal/saleae/client.go — gRPC dial + GetAppInfo wrapper
- /home/manuel/workspaces/2025-12-21/echo-base-documentation/salad/internal/saleae/config.go — Host/port/timeout config
- /home/manuel/workspaces/2025-12-21/echo-base-documentation/salad/ttmp/2025/12/24/001-INITIAL-SALAD--initial-salad-saleae-logic-analyzer-client-in-go/reference/01-diary.md — Step 3 diary entry


## 2025-12-24

Added devices command (GetDevices) and a manual smoke-test playbook; removed .git-commit-message.yaml per request.

### Related Files

- /home/manuel/workspaces/2025-12-21/echo-base-documentation/salad/cmd/salad/cmd/devices.go — devices CLI command
- /home/manuel/workspaces/2025-12-21/echo-base-documentation/salad/internal/saleae/client.go — GetDevices wrapper
- /home/manuel/workspaces/2025-12-21/echo-base-documentation/salad/ttmp/2025/12/24/001-INITIAL-SALAD--initial-salad-saleae-logic-analyzer-client-in-go/playbook/01-manual-smoke-test-logic2-automation.md — Manual validation steps
- /home/manuel/workspaces/2025-12-21/echo-base-documentation/salad/ttmp/2025/12/24/001-INITIAL-SALAD--initial-salad-saleae-logic-analyzer-client-in-go/reference/01-diary.md — Step 4 diary entry


## 2025-12-24

Validated CLI against running Logic 2 automation server (appinfo + devices succeeded).

### Related Files

- /home/manuel/workspaces/2025-12-21/echo-base-documentation/salad/ttmp/2025/12/24/001-INITIAL-SALAD--initial-salad-saleae-logic-analyzer-client-in-go/reference/01-diary.md — Step 5 live test outputs


## 2025-12-24

Committed initial Go client + docs (commit b4e51a4d); added and checked off explicit tasks for devices + live test.

### Related Files

- /home/manuel/workspaces/2025-12-21/echo-base-documentation/salad/ttmp/2025/12/24/001-INITIAL-SALAD--initial-salad-saleae-logic-analyzer-client-in-go/reference/01-diary.md — Recorded commit hash
- /home/manuel/workspaces/2025-12-21/echo-base-documentation/salad/ttmp/2025/12/24/001-INITIAL-SALAD--initial-salad-saleae-logic-analyzer-client-in-go/tasks.md — Added tasks 8-9 and checked them


## 2025-12-24

Added capture (load/save/stop/wait/close) and export (raw-csv/raw-binary) command skeletons with internal client wrappers.

### Related Files

- /home/manuel/workspaces/2025-12-21/echo-base-documentation/salad/cmd/salad/cmd/capture.go — Capture subcommands
- /home/manuel/workspaces/2025-12-21/echo-base-documentation/salad/cmd/salad/cmd/export.go — Export raw commands
- /home/manuel/workspaces/2025-12-21/echo-base-documentation/salad/cmd/salad/cmd/root.go — Wired capture/export into CLI
- /home/manuel/workspaces/2025-12-21/echo-base-documentation/salad/cmd/salad/cmd/util.go — Channel parsing helper
- /home/manuel/workspaces/2025-12-21/echo-base-documentation/salad/internal/saleae/client.go — Capture/export wrapper methods
- /home/manuel/workspaces/2025-12-21/echo-base-documentation/salad/ttmp/2025/12/24/001-INITIAL-SALAD--initial-salad-saleae-logic-analyzer-client-in-go/reference/01-diary.md — Step 6 diary entry

