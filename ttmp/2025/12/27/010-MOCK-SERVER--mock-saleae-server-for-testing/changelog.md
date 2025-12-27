# Changelog

## 2025-12-27

- Initial workspace created


## 2025-12-27

Created comprehensive design document and diary for mock Saleae server. Documented server interface requirements, state management design, RPC implementation patterns, and testing strategy.


## 2025-12-27

Step 2: Created CLI verbs analysis document mapping all 8 implemented CLI commands to their gRPC methods, documenting requirements for mock server implementation. Created comprehensive task list organized by implementation phase (discovery → capture → export). Updated diary with analysis findings.

### Related Files

- /home/manuel/workspaces/2025-12-27/salad-pass/salad/ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/analysis/02-cli-verbs-to-grpc-methods-mapping.md — Complete mapping of CLI commands to gRPC methods
- /home/manuel/workspaces/2025-12-27/salad-pass/salad/ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/tasks.md — 76 tasks organized by implementation phase


## 2025-12-27

Step 3: Simplified tasks to capability milestones and designed YAML DSL for configuring mock server behavior (fixtures, per-RPC policies, fault injection, export side effects) with a scenario library. Updated diary with scoping decisions and tradeoffs.

### Related Files

- /home/manuel/workspaces/2025-12-27/salad-pass/salad/ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/analysis/03-mock-server-yaml-dsl-configurable-behavior-scenarios.md — DSL design + scenario examples
- /home/manuel/workspaces/2025-12-27/salad-pass/salad/ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/reference/01-diary.md — Diary Step 3 rationale and constraints
- /home/manuel/workspaces/2025-12-27/salad-pass/salad/ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/tasks.md — Simplified capability tasks


## 2025-12-27

Step 4: Added a general method doc for mapping YAML DSL to Go structs and composable runtime behavior (config→plan compile step, defaults layering, fault injection matchers, deterministic clock, exec pipeline pseudocode). Updated diary with rationale.

### Related Files

- /home/manuel/workspaces/2025-12-27/salad-pass/salad/ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/analysis/04-mapping-yaml-dsl-to-go-structures-validation-and-behavior-composition.md — Mapping method and pseudocode
- /home/manuel/workspaces/2025-12-27/salad-pass/salad/ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/reference/01-diary.md — Diary Step 4


## 2025-12-27

Step 5: implement YAML-driven mock server pipeline (commit 6e70b3d)

### Related Files

- /workspace/salad/internal/mock/saleae/plan.go — Compile YAML configs into runtime plans and defaults
- /workspace/salad/internal/mock/saleae/server.go — Mock Manager RPC handlers and state transitions


## 2025-12-27

Step 6: add salad-mock CLI, docs, and scenarios (commit 3b13c72)

### Related Files

- /workspace/salad/cmd/salad-mock/cmd/root.go — Run mock server from YAML
- /workspace/salad/configs/mock/happy-path.yaml — Documented example scenario
- /workspace/salad/pkg/doc/mock-server-user-guide.md — User-facing mock server guide


## 2025-12-27

Step 7: guard against OK status on unknown captures

### Related Files

- /workspace/salad/internal/mock/saleae/plan.go — Compile-time validation for status_on_unknown_capture_id
- /workspace/salad/internal/mock/saleae/server.go — Runtime guard against nil capture


## 2025-12-27

Bug report: faults.yaml does not inject UNAVAILABLE for SaveCapture; observed InvalidArgument capture 10 not found. Added investigation notes, code pointers, and suggested logging.

### Related Files

- /home/manuel/workspaces/2025-12-27/salad-pass/salad/ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/analysis/05-bug-report-faults-yaml-does-not-inject-savecapture-unavailable-capture-10-not-found.md — Bug report
- /home/manuel/workspaces/2025-12-27/salad-pass/salad/ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/reference/01-diary.md — Diary Step 8


## 2025-12-27

Added tmux lifecycle scripts for salad-mock (start/stop/restart/tail) with persistent ticket-local logs, plus kill-by-port helper to avoid port-collision confusion during scenario testing. Updated faults smoke script to default to port 10432 and always dump mock.log.

### Related Files

- /home/manuel/workspaces/2025-12-27/salad-pass/salad/ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/scripts/03-smoke-faults.sh — Faults smoke improvements
- /home/manuel/workspaces/2025-12-27/salad-pass/salad/ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/scripts/09-kill-mock-on-port.sh — Kill stuck listener by port
- /home/manuel/workspaces/2025-12-27/salad-pass/salad/ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/scripts/10-tmux-mock-start.sh — Start in tmux + log to file
- /home/manuel/workspaces/2025-12-27/salad-pass/salad/ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/scripts/README.md — Script usage guide


## 2025-12-27

Re-ran go checks + happy-path + faults smoke after port cleanup: faults injection works as expected (UNAVAILABLE on first SaveCapture, ok on second). Added playbook: Debugging the mock server (ports, tmux, logs, smoke scripts).

### Related Files

- /home/manuel/workspaces/2025-12-27/salad-pass/salad/ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/analysis/05-bug-report-faults-yaml-does-not-inject-savecapture-unavailable-capture-10-not-found.md — Port collision mitigation + updated understanding
- /home/manuel/workspaces/2025-12-27/salad-pass/salad/ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/playbook/01-debugging-the-mock-server.md — Reusable debugging procedure
- /home/manuel/workspaces/2025-12-27/salad-pass/salad/ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/scripts/README.md — Script entry point


## 2025-12-27

Ticket closed: lint fixes + Go 1.25 golangci-lint pin + remove XXX placeholder


## 2025-12-27

Designed StartCapture implementation tasks for 002-CAPTURE-START testing. Added diary entry documenting analysis and task breakdown.


## 2025-12-27

Implemented StartCapture RPC handler: device selection/validation, capture mode extraction (manual/timed/trigger), capture state creation. Added config structures and compilation logic.


## 2025-12-27

Added StartCapture config examples: updated happy-path.yaml, created start-capture.yaml for testing 002-CAPTURE-START


## 2025-12-27

Updated tasks: marked StartCapture implementation tasks as complete. Remaining: tests for StartCapture.


## 2025-12-27

Added analyzer RPC support + tests:
- Implemented `AddAnalyzer` / `RemoveAnalyzer` in the mock server (commit df30f497)
- Added CLI-vs-mock integration test for analyzers (commit ab14a1c7)

