---
Title: 'Diary: Mock Saleae Server Implementation'
Ticket: 010-MOCK-SERVER
Status: active
Topics:
    - go
    - saleae
    - testing
    - mock
    - grpc
    - implementation
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: README.md
      Note: Links to mock server docs
    - Path: cmd/salad-mock/cmd/root.go
      Note: Cobra command to run mock server from YAML
    - Path: cmd/salad-mock/main.go
      Note: Entry point for mock server CLI
    - Path: configs/mock/faults.yaml
      Note: Example fault injection scenario
    - Path: configs/mock/happy-path.yaml
      Note: Example happy-path mock scenario
    - Path: go.mod
      Note: Added yaml dependency for config parsing
    - Path: go.sum
      Note: Added yaml dependency checksums
    - Path: internal/mock/saleae/clock.go
      Note: Clock abstraction for deterministic timing
    - Path: internal/mock/saleae/config.go
      Note: YAML config schema and loader used for scenarios
    - Path: internal/mock/saleae/exec.go
      Note: Shared exec pipeline and server scaffolding
    - Path: internal/mock/saleae/helper.go
      Note: YAML-driven mock server test helper
    - Path: internal/mock/saleae/plan.go
      Note: Compile step and defaults normalization for mock server runtime
    - Path: internal/mock/saleae/server.go
      Note: gRPC handlers and capture lifecycle behavior
    - Path: internal/mock/saleae/side_effects.go
      Note: Pluggable file side effects for save/export
    - Path: internal/mock/saleae/state.go
      Note: State and enums for captures/devices
    - Path: pkg/doc/mock-server-developer-guide.md
      Note: Developer guide for extending the mock server
    - Path: pkg/doc/mock-server-user-guide.md
      Note: User guide for running the mock server
ExternalSources: []
Summary: Step-by-step narrative of designing and implementing the mock Saleae server, documenting decisions, learnings, and challenges.
LastUpdated: 2025-12-27T00:00:00Z
---



# Diary: Mock Saleae Server Implementation

## Goal

Document the design and implementation journey for a mock Saleae Logic 2 gRPC server that enables testing salad commands without requiring a real Logic 2 instance or physical hardware. This diary captures what we learned about gRPC server implementation, state management patterns, and test infrastructure design.

## Step 1: Codebase Analysis and Design

This step involved analyzing the existing client code to understand how it connects to gRPC servers, examining the generated server interface, and designing the mock server architecture. The goal was to create a comprehensive design document before starting implementation.

**Commit (code):** N/A — Design phase

### What I did

- Analyzed `internal/saleae/client.go` to understand client connection patterns
- Examined `gen/saleae/automation/saleae_grpc.pb.go` to understand the `ManagerServer` interface
- Reviewed `proto/saleae/grpc/saleae.proto` to list all RPCs that need implementation
- Searched codebase for existing test patterns (found none, so this is new infrastructure)
- Created comprehensive design document covering:
  - Mental model of what a mock server does
  - Codebase structure analysis
  - Server interface requirements
  - State management design
  - RPC implementation patterns
  - Testing strategy

### Why

- Need to understand client connection mechanism to design server startup
- Must understand all RPCs to prioritize implementation
- State management is critical—mock must track captures, devices, analyzers correctly
- Test infrastructure needs to be simple to use but realistic enough to catch bugs

### What worked

- Generated gRPC code provides `UnimplementedManagerServer` that we can embed
- Client uses standard `grpc.DialContext`—any server on any port works
- Proto definition clearly lists all 14 RPCs we need to implement
- Existing client code shows exactly what the mock needs to support

### What didn't work

- No existing test infrastructure in codebase—need to build from scratch
- No examples of gRPC server implementation in this codebase
- Need to research gRPC server patterns (standard Go patterns, but not documented here)

### What I learned

**gRPC Server Pattern:**
- Create `grpc.NewServer()`
- Implement the server interface (embed `UnimplementedManagerServer`)
- Register with `RegisterManagerServer(grpcServer, implementation)`
- Listen on port with `net.Listen("tcp", addr)`
- Serve with `grpcServer.Serve(lis)`
- Stop with `grpcServer.Stop()`

**State Management:**
- Mock server needs to track: devices, captures (by ID), analyzers (by capture+analyzer ID)
- Use mutex (`sync.RWMutex`) for thread-safe access
- Generate IDs with atomic counter or mutex-protected counter
- State must persist across RPC calls (store in Server struct, not function locals)

**Client Connection:**
- Client uses `grpc.DialContext` with address string (e.g., "127.0.0.1:10430")
- Mock server can listen on any port (use random port in tests to avoid conflicts)
- Client doesn't know it's talking to a mock—just a gRPC connection

**RPC Implementation:**
- All RPCs take `context.Context` and request proto, return reply proto and error
- Use `status.Error(codes.XXX, "message")` for gRPC errors, not Go errors
- Embed `UnimplementedManagerServer` by value (not pointer) for forward compatibility
- Only implement RPCs we need—unimplemented ones return "not implemented" error

**Testing Patterns:**
- Create helper function `StartMockServer(t)` that returns server, client, cleanup
- Use random port (`:0`) to avoid conflicts
- Return cleanup function for `defer cleanup()` pattern
- Test both the mock server itself (unit tests) and client code using mock (integration tests)

### What was tricky to build

**State Management Design:**
- Deciding what state to track (captures, devices, analyzers)
- How to structure state (maps vs slices, key formats)
- Thread safety (mutex placement, read vs write locks)
- ID generation (atomic vs mutex, starting from 1 vs 0)

**WaitCapture Behavior:**
- Real `WaitCapture` blocks until capture completes
- Mock can't block tests indefinitely
- Options: return immediately (assume completion), sleep (slow tests), use channels (complex)
- Decided: return success if elapsed >= duration, error otherwise (document limitation)

**Export RPCs:**
- Real exports write files to disk
- Mock options: actually write files (test can verify), track calls (test verifies RPC), do nothing (simplest)
- Decided: start with simplest (just verify capture exists), add file I/O if tests need it

**Error Handling:**
- Need to match Logic 2's error behavior (device not found, capture not found, etc.)
- Use correct gRPC status codes (`codes.NotFound`, `codes.InvalidArgument`)
- Tests check status codes, not error strings

### What warrants a second pair of eyes

**State Management:**
- Verify mutex usage is correct (RWMutex for reads, Mutex for writes)
- Check ID generation is thread-safe
- Ensure state persists correctly across RPC calls

**RPC Implementations:**
- Verify error cases match Logic 2 behavior
- Check status codes are correct
- Ensure validation logic matches proto comments

**Test Helper:**
- Verify cleanup function stops server correctly
- Check random port selection avoids conflicts
- Ensure client connection works reliably

**WaitCapture Implementation:**
- Review decision to return immediately vs block
- Consider if tests need blocking behavior
- Document limitation clearly

### What should be done in the future

**File I/O for Exports:**
- If tests need to verify export files, implement actual file writing
- Create temporary directories for test files
- Clean up files in test cleanup

**Capture Timing:**
- If tests need realistic timing, implement time-based state transitions
- Use goroutines to simulate capture completion after duration
- Add context cancellation support

**Error Injection:**
- Add ability to inject errors for testing error handling
- Allow tests to configure mock to return specific errors
- Support testing retry logic, timeouts, etc.

**State Inspection:**
- Add methods to inspect mock server state (for test assertions)
- Allow tests to query captures, devices, analyzers
- Support state reset for test isolation

**Performance Testing:**
- If needed, add benchmarks using mock server
- Test concurrent RPC handling
- Verify mutex contention doesn't slow tests

### Code review instructions

**Start here:**
- `analysis/01-mock-server-design.md` — complete design document

**Key files to review:**
- `gen/saleae/automation/saleae_grpc.pb.go:260-294` — ManagerServer interface
- `gen/saleae/automation/saleae_grpc.pb.go:302-352` — UnimplementedManagerServer
- `gen/saleae/automation/saleae_grpc.pb.go:362` — RegisterManagerServer function
- `internal/saleae/client.go:18-38` — Client connection code
- `proto/saleae/grpc/saleae.proto:29-81` — All RPC definitions

**How to validate:**
- Read design document sections on state management and RPC patterns
- Compare server struct design with gRPC best practices
- Review test helper pattern against Go testing conventions
- Check that all 14 RPCs are accounted for in implementation plan

### Technical details

**Server Interface:**
```go
type ManagerServer interface {
    GetAppInfo(context.Context, *GetAppInfoRequest) (*GetAppInfoReply, error)
    GetDevices(context.Context, *GetDevicesRequest) (*GetDevicesReply, error)
    StartCapture(context.Context, *StartCaptureRequest) (*StartCaptureReply, error)
    // ... 11 more RPCs
    mustEmbedUnimplementedManagerServer()
}
```

**Server Struct:**
```go
type Server struct {
    pb.UnimplementedManagerServer  // Embed by value
    
    mu sync.RWMutex
    
    devices   []*pb.Device
    captures  map[uint64]*CaptureState
    analyzers map[string]*AnalyzerState
    
    nextCaptureID  uint64
    nextAnalyzerID uint64
}
```

**Test Helper Pattern:**
```go
func StartMockServer(t *testing.T) (*Server, *saleae.Client, func()) {
    s := NewServer()
    lis, _ := net.Listen("tcp", ":0")  // Random port
    grpcServer := grpc.NewServer()
    pb.RegisterManagerServer(grpcServer, s)
    go grpcServer.Serve(lis)
    
    client, _ := saleae.New(ctx, saleae.Config{
        Host: "127.0.0.1",
        Port: extractPort(lis.Addr().String()),
    })
    
    return s, client, func() {
        grpcServer.Stop()
        lis.Close()
        client.Close()
    }
}
```

**RPC Implementation Pattern:**
```go
func (s *Server) StartCapture(ctx context.Context, req *pb.StartCaptureRequest) (*pb.StartCaptureReply, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    // Validate
    // Create state
    // Return reply
}
```

### What I'd do differently next time

- Research gRPC server patterns earlier (standard Go patterns, but good to confirm)
- Look at other Go projects with mock gRPC servers for inspiration
- Consider using a testing library that provides gRPC server helpers (if one exists)
- Document WaitCapture blocking behavior decision earlier (it's a key design choice)

---

## Step 2: CLI Verbs Analysis and Task Creation

This step involved analyzing all existing CLI commands to understand which gRPC methods they call, how they use those methods, and what the mock server needs to implement to support testing. The goal was to create a comprehensive mapping document and detailed implementation tasks.

**Commit (code):** N/A — Analysis phase

### What I did

- Read all CLI command files (`appinfo.go`, `devices.go`, `capture.go`, `export.go`)
- Traced each command to its client wrapper method in `internal/saleae/client.go`
- Identified which gRPC RPC each command calls
- Analyzed request/response structures and command behavior
- Documented mock server requirements for each RPC
- Created comprehensive task list organized by implementation phase
- Related CLI command files to the analysis document

### Why

- Need to understand what the mock server must support before implementing
- Want to prioritize implementation based on what CLI commands are already implemented
- Need clear tasks to guide implementation work
- Want to ensure no CLI command is missed in mock server implementation

### What worked

- CLI commands are well-structured and easy to trace
- Client wrapper methods clearly map to gRPC calls
- Proto definitions provide clear request/response structures
- Existing commands cover 8 RPCs (out of 14 total in the API)
- Clear separation between discovery, capture lifecycle, and export commands

### What didn't work

- Some commands have complex flag parsing (e.g., `export` commands parse CSV channel lists)
- Export commands have optional file I/O that needs design decision
- `WaitCapture` has mode-specific behavior that needs careful implementation

### What I learned

**CLI Command Structure:**
- Commands follow consistent pattern: create client → call RPC → print output
- Commands use Cobra framework with flags for parameters
- Output format is simple (key=value pairs or "ok")
- Error handling propagates from client wrapper

**RPC Usage Patterns:**
- Discovery commands (`appinfo`, `devices`) are simple, no state
- Capture commands (`load`, `save`, `stop`, `wait`, `close`) require state tracking
- Export commands (`raw-csv`, `raw-binary`) require capture validation and optional file I/O

**Implementation Priority:**
- Phase 1: Discovery (2 RPCs) — simplest, no state
- Phase 2: Capture lifecycle (5 RPCs) — medium complexity, requires state
- Phase 3: Export (2 RPCs) — complex, requires state + optional file I/O
- Phase 4: Future RPCs (5 RPCs) — not yet needed, can be stubbed

**Mock Server Requirements:**
- Must track: devices (list), captures (map by ID), analyzers (map by key)
- Must generate IDs: capture IDs start at 1, analyzer IDs start at 1
- Must validate: capture IDs exist, devices exist, channels valid
- Must handle errors: use gRPC status codes (`codes.InvalidArgument`, `codes.NotFound`)

### What was tricky to build

**Task Organization:**
- Deciding how to break down tasks (by RPC? by phase? by feature?)
- Balancing detail (too much = overwhelming) vs clarity (too little = unclear)
- Deciding what belongs in "basic infrastructure" vs "RPC implementation"

**Export RPC Complexity:**
- Export RPCs have many parameters (capture ID, directory, channels, downsample ratio, timestamp format)
- Need to decide: validate all parameters or just capture ID?
- File I/O decision: write files (testable) vs just return success (simpler)

**WaitCapture Behavior:**
- Real `WaitCapture` blocks until capture completes
- Mock can't block tests indefinitely
- Need to document limitation clearly
- Mode-specific logic (manual vs timed vs trigger) adds complexity

**State Management:**
- Need to track capture status (running, stopped, completed)
- Need to track capture mode for `WaitCapture` logic
- Need to track start time for timed captures
- Need thread-safe access (mutex protection)

### What warrants a second pair of eyes

**Task List:**
- Verify all CLI commands are covered
- Check task breakdown is appropriate (not too granular, not too coarse)
- Ensure tasks are actionable and testable
- Verify priority order makes sense

**Analysis Document:**
- Check that all RPC requirements are documented correctly
- Verify complexity ratings are accurate
- Ensure error handling requirements are complete
- Confirm state management requirements are sufficient

**Implementation Plan:**
- Review phase ordering (discovery → capture → export)
- Check that basic infrastructure tasks come first
- Verify test tasks are included for each RPC
- Ensure future RPCs are documented but not blocking

### What should be done in the future

**Test Coverage:**
- Add integration tests for each CLI command using mock server
- Test error cases (invalid IDs, missing resources)
- Test concurrent access (multiple goroutines)
- Test state persistence across RPC calls

**File I/O for Exports:**
- If tests need to verify export files, implement actual file writing
- Create temporary directories for test files
- Clean up files in test cleanup
- Document file format expectations

**State Inspection API:**
- Add methods to inspect mock server state (`GetCapture`, `GetDevices`)
- Add methods to configure state (`AddDevice`, `SetAppInfo`)
- Add methods to reset state (for test isolation)
- Document state inspection API for test writers

**Error Injection:**
- Add ability to inject errors for testing error handling
- Allow tests to configure mock to return specific errors
- Support testing retry logic, timeouts, etc.
- Document error injection API

### Code review instructions

**Start here:**
- `analysis/02-cli-verbs-to-grpc-methods-mapping.md` — complete CLI-to-RPC mapping
- `tasks.md` — detailed implementation tasks

**Key files to review:**
- `cmd/salad/cmd/appinfo.go` — simplest command, calls `GetAppInfo`
- `cmd/salad/cmd/devices.go` — simple command, calls `GetDevices`
- `cmd/salad/cmd/capture.go` — capture lifecycle commands (5 RPCs)
- `cmd/salad/cmd/export.go` — export commands (2 RPCs)
- `internal/saleae/client.go` — client wrapper methods

**How to validate:**
- Read analysis document to understand each command's requirements
- Check that all 8 implemented CLI commands are covered
- Verify task list covers all required RPCs
- Ensure tasks are organized by priority (discovery → capture → export)

### Technical details

**CLI Commands Analyzed:**
1. `salad appinfo` → `GetAppInfo` (discovery, simple)
2. `salad devices` → `GetDevices` (discovery, simple)
3. `salad capture load` → `LoadCapture` (capture lifecycle, medium)
4. `salad capture save` → `SaveCapture` (capture lifecycle, medium)
5. `salad capture stop` → `StopCapture` (capture lifecycle, medium)
6. `salad capture wait` → `WaitCapture` (capture lifecycle, complex)
7. `salad capture close` → `CloseCapture` (capture lifecycle, medium)
8. `salad export raw-csv` → `ExportRawDataCsv` (export, complex)
9. `salad export raw-binary` → `ExportRawDataBinary` (export, complex)

**RPCs Required for Current CLI:**
- Phase 1: `GetAppInfo`, `GetDevices` (2 RPCs)
- Phase 2: `LoadCapture`, `SaveCapture`, `StopCapture`, `WaitCapture`, `CloseCapture` (5 RPCs)
- Phase 3: `ExportRawDataCsv`, `ExportRawDataBinary` (2 RPCs)
- **Total: 9 RPCs** (out of 14 total in API)

**Future RPCs (Not Yet Required):**
- `StartCapture` (for ticket 002)
- `AddAnalyzer`, `RemoveAnalyzer` (for ticket 003)
- `AddHighLevelAnalyzer`, `RemoveHighLevelAnalyzer` (for ticket 004)
- `ExportDataTableCsv` (for ticket 005)
- `LegacyExportAnalyzer` (low priority)

**Task Breakdown:**
- Phase 1: Basic Infrastructure (6 tasks)
- Phase 2: Discovery RPCs (2 RPCs, ~10 tasks)
- Phase 3: Capture Lifecycle RPCs (5 RPCs, ~25 tasks)
- Phase 4: Export RPCs (2 RPCs, ~10 tasks)
- Phase 5: Testing Infrastructure (~5 tasks)
- Phase 6: Future RPCs (5 RPCs, ~20 tasks)
- **Total: ~76 tasks** (many are subtasks)

### What I'd do differently next time

- Start with CLI command analysis before design document (understand requirements first)
- Create task list earlier in the process (helps prioritize implementation)
- Document complexity ratings for each RPC (helps estimate effort)
- Include test requirements in task list from the start

---

## Step 3: Simplify Tasks + Design YAML DSL for Configurable Mock Behavior

This step was about taking a breath and de-overengineering the “task plan” while simultaneously raising the actual leverage of the mock server: we want to change mock behavior without touching Go code, so tests can cover lots of real-world situations by swapping YAML scenario files. The outcome is a much smaller task list (capability milestones) and a thorough DSL design that focuses on fixtures, policy knobs, and deterministic fault injection.

**Commit (code):** N/A — Documentation and planning phase

### What I did

- Rewrote `tasks.md` to be a short capability checklist instead of a 70+ subtask breakdown
- Designed a YAML DSL for configuring mock server behavior (fixtures + per-RPC knobs + fault injection)
- Wrote a scenario library with multiple distinct configurations (happy path, filtering, transient errors, wait behavior, export side effects)
- Related the DSL design to current CLI verbs so it’s grounded in real coverage needs

### Why

- A long, deeply nested task list makes it harder to start; it looks “complete” but is hard to execute incrementally.
- A YAML-configurable mock server is the highest ROI lever: we can write many tests by adding YAML files rather than adding Go branches everywhere.
- We want deterministic reproduction: scenario YAML should lock down IDs, errors, and side effects.

### What worked

- The mock requirements naturally collapse into a small set of orthogonal concepts:
  - fixtures (appinfo/devices/captures)
  - behavior policies (timing model, close semantics, export side effects)
  - faults (error injection rules)
- A “first match wins” ordered `faults` list is simple and expressive enough for most test needs.
- Modeling `WaitCapture` via a policy knob (immediate vs error_if_running vs tiny block) avoids flaky tests.

### What didn't work

- The original tasks were too detailed and repetitive (every RPC repeated “validate X / write unit test / write integration test”).
- Full request matching for complex nested fields (channels, etc.) quickly becomes a rabbit hole; we intentionally scoped matching to a small subset.

### What I learned

**DSL shape that stays sane:**
- Keep it scenario-based: one YAML file = one coherent “world”.
- Separate:
  - `fixtures` (initial state)
  - `behavior` (policies + side effects)
  - `faults` (error injection)
- Make deterministic defaults explicit (`capture_id_start`, `deterministic: true`).

**Fault injection needs to be deterministic:**
- `nth_call` is a huge win for simulating transient failures without randomness.
- Request matching should start small (capture_id, filepath) and only grow when tests force it.

### What was tricky to build

- Designing `WaitCapture` semantics: tests must not hang, but we still need “running vs completed” coverage.
- Avoiding a DSL that mirrors the entire proto schema (too big), while still being expressive enough for CLI integration tests.
- Deciding how much filesystem side effect to simulate for exports/save (no-op vs placeholder files).

### What warrants a second pair of eyes

- DSL ergonomics: are the keys intuitive, or are we encoding too much “implementation detail” into YAML?
- Fault precedence rules (ordered list): confirm this is easy to reason about in tests.
- `WaitCapture` policy defaults: choose a default that won’t surprise test authors.

### What should be done in the future

- Add a small “golden scenarios” directory and ensure tests actually run against those YAML files (so docs don’t drift).
- Decide whether unknown capture IDs should default to `INVALID_ARGUMENT` or `NOT_FOUND`, and document it as a contract.
- If/when analyzer/HLA verbs land, extend the DSL with `fixtures.analyzers` + analyzer ID generation and minimal behavior knobs.

### Code review instructions

- Start with `analysis/03-mock-server-yaml-dsl-configurable-behavior-scenarios.md`
- Then check `tasks.md` for the simplified capability milestones
- Skim this diary step for the reasoning behind scoping decisions

---

## Step 4: General Method — Mapping YAML DSL to Go Runtime Behavior

This step focused on documenting a reusable engineering approach for turning our YAML scenario DSL into clean Go code: decode into config structs, compile into an immutable runtime plan, and implement RPC handlers as a consistent pipeline (faults → validate → state transitions → side effects → reply). The goal is to keep the mock server easy to extend (new RPCs, new knobs) without accumulating scattered YAML interpretation logic across handlers.

**Commit (code):** N/A — Documentation phase

### What I did

- Wrote a dedicated document describing the YAML→Go mapping method, including recommended struct shapes, defaults layering, compilation/normalization, and pseudocode for the core executor pipeline
- Documented strategies for fault injection, request matching (start small), deterministic time, and pluggable filesystem side effects

### Why

- Without a clear mapping method, DSL-driven systems tend to devolve into ad-hoc per-handler YAML lookups and duplicated validation logic.
- A “compile step” (config → plan) makes runtime code simpler, faster, and safer (validated/normalized once).
- A consistent handler pipeline makes correctness review much easier: every RPC follows the same shape.

### What worked

- The separation of `Config` (YAML-facing) vs `Plan` (compiled runtime) naturally enforces clean boundaries.
- First-match-wins fault rules + per-method call counters yield deterministic transient failure simulation without randomness.
- Introducing a `Clock` interface gives us deterministic timing semantics without sleeping tests by default.

### What was tricky to build

- Picking a request-matching strategy that is powerful enough for tests but doesn’t become “generic protobuf query language”.
- Defining `WaitCapture` policies that avoid hangs/flakes but still allow “running vs completed” behavior coverage.
- Designing side effects so we can test filesystem outputs when needed, without baking IO into core handler logic.

### What warrants a second pair of eyes

- Ensure the proposed `exec` pipeline doesn’t hide important method-specific semantics (i.e., avoid too much indirection).
- Confirm the defaults layering rule is intuitive and won’t lead to surprising overrides.
- Review whether “unknown fields rejected” during YAML decode is acceptable (it’s great for catching typos, but can be annoying during iteration).

### Code review instructions

- Start with `analysis/04-mapping-yaml-dsl-to-go-structures-validation-and-behavior-composition.md`
- Cross-check against `analysis/03-mock-server-yaml-dsl-configurable-behavior-scenarios.md` to ensure the DSL keys map cleanly into the proposed Go structs

---

## Step 5: Implement YAML-driven mock server core (config, plan, exec, RPCs)

This step turned the DSL and runtime plan design into real code. I implemented the YAML config loader, compilation into a runtime plan, and the core mock gRPC server that runs every RPC through a shared exec pipeline. This unlocked a working mock server that is fully scenario-driven and supports all CLI-required RPCs.

**Commit (code):** 6e70b3d — "✨ Add YAML-driven mock Saleae server"

### What I did
- Added `internal/mock/saleae` with config structs, plan compiler, and runtime state.
- Implemented `Server` with a shared exec wrapper and handlers for all CLI-required RPCs.
- Implemented deterministic IDs, clock injection, capture lifecycle state, and fault injection.
- Added pluggable side effects (no-op vs file placeholders) and export/save behaviors.
- Added a YAML-driven test helper to start the mock server on a random port.

### Why
- We need a mock server that can be configured via YAML scenarios to support many test cases without hardcoding behavior in Go.
- The exec pipeline keeps RPC handlers consistent and makes fault injection and validation uniform.
- Deterministic IDs and a clock abstraction are essential for predictable, non-flaky tests.

### What worked
- Strict YAML decoding (`KnownFields`) catches config typos early.
- The plan compiler centralizes defaults and type normalization, keeping handlers clean.
- The mock server compiles and runs through `go test` successfully.

### What didn't work
- The first `go test ./...` run appeared to hang with no output; I interrupted it and re-ran with `timeout 60s` to avoid stalling the session.

### What I learned
- gRPC status code parsing needed a custom map; `codes.Code_value` doesn’t exist in the grpc package.
- Centralizing side effects behind an interface makes it easy to switch between no-op and file output behavior.

### What was tricky to build
- Mapping YAML enums to Go enums in a safe, user-friendly way without leaking YAML concerns into handlers.
- Implementing a `WaitCapture` policy that is deterministic and non-blocking while still modeling “running vs completed” semantics.

### What warrants a second pair of eyes
- The `WaitCapture` policy behavior (especially `block_until_done`) should be reviewed to ensure it matches expectations.
- Fault matcher coverage: request-field matchers are intentionally minimal (capture_id, filepath). Confirm this is sufficient for initial tests.

### What should be done in the future
- Add table-driven integration tests that run the CLI against YAML scenarios.
- Add example scenario YAML files under `configs/mock/` to keep docs and tests aligned.

### Code review instructions
- Start in `internal/mock/saleae/plan.go` for the compiler and defaults.
- Review the exec pipeline in `internal/mock/saleae/exec.go`.
- Inspect RPC handlers and state transitions in `internal/mock/saleae/server.go`.
- Review side effects in `internal/mock/saleae/side_effects.go`.

### Technical details
- Config loader: `LoadConfig` uses `yaml.Decoder` with `KnownFields(true)`.
- Plan compiler: `Compile` validates version, normalizes enums, applies defaults, and compiles faults.
- Exec pipeline: increments call counters, applies fault injection, and routes to per-method logic.

### What I'd do differently next time
- Run the initial `go test` with a timeout to avoid waiting on a hang.

---

## Step 6: Add mock server docs, examples, and validate CLI against the server

This step focused on usability: I added a runnable `salad-mock` CLI, example YAML scenarios, and documentation for both users and developers. I also installed `tmux` (required by the project guidelines) so I could run the mock server and validate the `salad` CLI against it.

**Commit (code):** 3b13c72 — "✨ Add mock server CLI and docs"

### What I did
- Added `cmd/salad-mock` to run the mock server from a YAML config.
- Added example scenarios under `configs/mock/` for happy path and failure injection.
- Wrote user and developer guides in `pkg/doc/` and linked them from `README.md`.
- Installed `tmux` and ran `salad appinfo` + `salad devices` against the mock server.

### Why
- Users need a documented way to run the mock server and point the CLI at it.
- Example scenario files reduce setup friction and make it easier to write tests.
- The project guidelines require `tmux` for running servers interactively, so it needed to be installed.

### What worked
- `salad-mock` starts the server cleanly from a YAML file.
- The CLI successfully connected to the mock server and returned expected app info and device output.

### What didn't work
- `tmux` was not installed initially, so I had to install it via `apt-get` before running the server.
- `apt-get update` warned about a 403 from `https://mise.jdx.dev/deb`; the rest of the package indexes updated normally.
- The mock server exited before `tmux kill-session` ran, so the kill command reported no running server.

### What I learned
- Installing `tmux` in this environment is straightforward but requires `apt-get update` first.
- The mock server is responsive enough for quick CLI smoke tests (appinfo/devices).

### What was tricky to build
- Ensuring the new CLI uses cobra and the existing logging conventions while keeping the surface area minimal.
- Documenting both the user and developer flows without drifting from the YAML DSL design.

### What warrants a second pair of eyes
- Review the documentation flow to confirm it matches expected CLI usage and test workflows.
- Confirm the `salad-mock` flags and defaults align with how tests should invoke the mock server.

### What should be done in the future
- Add table-driven CLI integration tests using the scenario YAML files.
- Expand example scenarios to cover `WaitCapture` running vs completed behavior.

### Code review instructions
- Start with `cmd/salad-mock/cmd/root.go` for server startup logic.
- Review example scenarios in `configs/mock/`.
- Read the guides in `pkg/doc/mock-server-user-guide.md` and `pkg/doc/mock-server-developer-guide.md`.

### Technical details
- Command used to start the mock server:
  - `go run ./cmd/salad-mock --config configs/mock/happy-path.yaml --port 10431`
- CLI validation commands:
  - `go run ./cmd/salad --host 127.0.0.1 --port 10431 appinfo`
  - `go run ./cmd/salad --host 127.0.0.1 --port 10431 devices`

### What I'd do differently next time
- Install `tmux` before starting CLI validation to avoid retry steps.
