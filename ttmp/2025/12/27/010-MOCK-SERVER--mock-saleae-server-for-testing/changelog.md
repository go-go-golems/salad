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

