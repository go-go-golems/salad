---
Title: "Diary: Analyzers (add/remove + settings/templates)"
Ticket: 003-ANALYZERS
Status: active
Topics:
    - go
    - saleae
    - logic-analyzer
    - client
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2025-12-27T17:08:57.00980015-05:00
WhatFor: ""
WhenToUse: ""
---

# Diary: Analyzers (add/remove + settings/templates)

## Goal

Implement `salad analyzer add/remove` (ticket 003) in a way that can be tested against:
- a **real** Logic 2 automation gRPC server (manual smoke test), and
- the **mock** server (ticket 010) later for CI and deterministic tests.

## Step 1: Create implementation diary + identify minimum compile-green units

This step bootstraps the diary and sets up an execution cadence: implement in small compile-green increments, commit each increment, then record what happened (including failures) while it’s still fresh. This keeps progress reviewable and reduces the risk of “one huge commit” with unclear breakpoints.

**Commit (code):** N/A — documentation scaffolding

### What I did
- Created `ttmp/2025/12/24/003-ANALYZERS--analyzers-add-remove-settings-templates/reference/01-diary.md` via `docmgr doc add`
- Confirmed analyzer RPCs and message schemas in `proto/saleae/grpc/saleae.proto`:
  - `AddAnalyzer(AddAnalyzerRequest) -> AddAnalyzerReply`
  - `RemoveAnalyzer(RemoveAnalyzerRequest) -> RemoveAnalyzerReply`
- Identified the smallest compile-green implementation units to commit:
  - Add client wrappers (`internal/saleae/client.go`)
  - Add analyzer settings parsing (`internal/config/...`)
  - Add Cobra verbs (`cmd/salad/cmd/analyzer.go` + `cmd/salad/cmd/root.go`)

### Why
- The analyzer feature spans client + CLI + config parsing; splitting into small commits makes it much easier to review and debug.

### What worked
- The proto clearly defines analyzer settings as a typed oneof (`AnalyzerSettingValue`), which maps well to a small parsing layer.

### What didn't work
- N/A

### What I learned
- `AddAnalyzerReply` returns `analyzer_id` (uint64). That value should be printed by the CLI the same way capture commands print `capture_id=...`.

### What was tricky to build
- N/A (setup step)

### What warrants a second pair of eyes
- N/A (setup step)

### What should be done in the future
- N/A

### Code review instructions
- Start at `proto/saleae/grpc/saleae.proto` around `AddAnalyzerRequest` to confirm request/response contracts.

---

## Step 2: Add analyzer RPC wrappers to the Saleae client

This step adds small, typed wrappers to the Go client so the CLI can call analyzer RPCs without duplicating proto details at the command layer. The goal is to mirror the existing capture/export client style: validate inputs, call the gRPC method, and wrap errors with a stable prefix.

**Commit (code):** a574108c310947e3d47b71ad697d997fa22838f9 — "Saleae client: add AddAnalyzer/RemoveAnalyzer"

### What I did
- Added client methods in `internal/saleae/client.go`:
  - `AddAnalyzer(ctx, captureID, analyzerName, analyzerLabel, settings) (uint64, error)`
  - `RemoveAnalyzer(ctx, captureID, analyzerID) error`
- Ran:
  - `gofmt -w internal/saleae/client.go`
  - `go test ./... -count=1`

### Why
- Keeps Cobra commands simple: the CLI should build inputs and delegate RPC calling + error wrapping to the client layer.

### What worked
- `go test ./...` stayed green after adding the methods.

### What didn't work
- N/A

### What I learned
- The analyzer settings map type is `map[string]*pb.AnalyzerSettingValue` (typed oneof), so we’ll need a dedicated parsing layer for JSON/YAML + typed flag overrides.

### What was tricky to build
- Ensuring `settings` behaves predictably when omitted: we normalize `nil` to an empty map to avoid accidental nil-map surprises.

### What warrants a second pair of eyes
- Error message consistency (prefixes + argument naming) vs existing client methods (capture/export).

### What should be done in the future
- Add unit tests for settings parsing (once the parsing package exists). The client wrappers themselves are thin enough that tests are optional.

### Code review instructions
- Start in `internal/saleae/client.go`, search for `AddAnalyzer(` and `RemoveAnalyzer(`.
- Validate with:
  - `go test ./... -count=1`

---

## Step 3: Implement analyzer settings parsing (JSON/YAML + typed overrides)

This step introduces a dedicated settings parsing layer that converts JSON/YAML settings into the proto’s typed oneof map (`map[string]*pb.AnalyzerSettingValue`). It also supports explicit typed overrides via `key=value` strings so the CLI can avoid “type guessing” for ad-hoc overrides.

**Commit (code):** e8a1d3c254871a18feb268458103995576a1e61e — "Config: parse analyzer settings (json/yaml + typed overrides)"

### What I did
- Added `internal/config/analyzer_settings.go`:
  - Load from JSON (`.json`) and YAML (`.yaml/.yml`)
  - Accept either a top-level mapping or a `{settings: {...}}` wrapper
  - Convert scalars to `AnalyzerSettingValue` (string/bool/int64/double)
  - Apply typed overrides (`--set`, `--set-bool`, `--set-int`, `--set-float`) as last-write-wins
- Added unit tests in `internal/config/analyzer_settings_test.go`
- Ran:
  - `gofmt -w internal/config/*.go`
  - `go test ./... -count=1`

### Why
- The Saleae API does not expose analyzer schemas; we need a deterministic and reproducible way to provide settings as code.
- A dedicated parser keeps Cobra command code small and reviewable.

### What worked
- Unit tests cover both JSON and YAML inputs (top-level and `settings:` wrapper) and typed overrides.

### What didn't work
- N/A

### What I learned
- JSON numbers decode as `float64`, so we treat “integral floats” as `int64_value` and non-integral as `double_value`.

### What was tricky to build
- Balancing “no type guessing” with practical JSON/YAML decoding: file-based settings inevitably carry types, so we keep guessing limited to “int vs double for numeric scalars”.

### What warrants a second pair of eyes
- The numeric coercion rules (float → int64 when integral) and whether they match what we want long-term.

### What should be done in the future
- If numeric coercion ever becomes a compatibility hazard, lock it down explicitly in docs/tests as a contract (or require explicit typing in settings files).

### Code review instructions
- Start in `internal/config/analyzer_settings.go`:
  - `LoadAnalyzerSettings*`
  - `ApplyAnalyzerSettingOverrides`
- Validate with:
  - `go test ./... -count=1`

---

## Step 4: Add `salad analyzer add/remove` Cobra verbs

This step adds the CLI surface for analyzers. The commands follow the established `salad` pattern (create client → call wrapper → print key/value output) and integrate the new settings parsing layer so settings can come from JSON/YAML plus typed overrides.

**Commit (code):** 99d3b4b004c3836b885daacd641dad748789d67d — "CLI: add analyzer add/remove commands"

### What I did
- Added `cmd/salad/cmd/analyzer.go`:
  - `salad analyzer add`:
    - flags: `--capture-id`, `--name`, `--label`, `--settings-json`, `--settings-yaml`, `--set*`
    - output: `analyzer_id=<id>`
  - `salad analyzer remove`:
    - flags: `--capture-id`, `--analyzer-id`
    - output: `ok`
- Wired analyzer subtree into `cmd/salad/cmd/root.go`
- Ran:
  - `gofmt -w cmd/salad/cmd/analyzer.go cmd/salad/cmd/root.go`
  - `go test ./... -count=1`

### Why
- The CLI verbs are the user-facing API; once these compile, we can do real-server smoke tests immediately.

### What worked
- Commands compile and integrate cleanly with the existing root flags (`--host`, `--port`, `--timeout`).

### What didn't work
- N/A

### What I learned
- Avoided package-level variable collisions by using analyzer-specific flag variables (`analyzerCaptureID`, `analyzerID`, etc.) instead of reusing `captureID`.

### What was tricky to build
- Keeping flags ergonomic but explicit: typed overrides are supported without attempting complex type inference for `--set`.

### What warrants a second pair of eyes
- CLI UX consistency:
  - flag names and error messages vs the capture/export commands
  - merge precedence (file settings vs overrides)

### What should be done in the future
- Add `salad analyzer template ...` commands only after the core add/remove loop is stable.

### Code review instructions
- Start in `cmd/salad/cmd/analyzer.go` and scan the `RunE` logic for settings loading + overrides.
- Validate with:
  - `go test ./... -count=1`

---

## Step 5: Smoke test analyzer add/remove against a real Logic 2 server

This step validates that the analyzer verbs work end-to-end against a **real** Saleae Logic 2 Automation gRPC server (not the mock). The main unknown was analyzer settings: since the API doesn’t expose schemas, we expected some trial-and-error to discover which setting keys are required for a successful `AddAnalyzer`.

**Commit (code):** N/A — manual test run (no code changes required for the final successful run)

### What I did
- Verified connectivity to the real server:
  - `go run ./cmd/salad --host 127.0.0.1 --port 10430 --timeout 2s appinfo`
  - `go run ./cmd/salad --host 127.0.0.1 --port 10430 --timeout 2s devices`
- Created a manual capture to obtain a `capture_id` using the helper script:
  - `go run ./ttmp/2025/12/24/003-ANALYZERS--analyzers-add-remove-settings-templates/scripts/01-real-start-capture.go --host 127.0.0.1 --port 10430 --timeout 5s`
  - Output: `capture_id=2`
- Attempted `AddAnalyzer` with empty settings (expected to fail):
  - `go run ./cmd/salad --host 127.0.0.1 --port 10430 --timeout 5s analyzer add --capture-id 2 --name "SPI" --label "smoke"`
  - Observed error: `Analyzer settings errors: "Invalid channel(s)"`
- Retried with explicit channel settings (guessed UI keys) and succeeded:
  - `go run ./cmd/salad --host 127.0.0.1 --port 10430 --timeout 5s analyzer add --capture-id 2 --name "SPI" --label "smoke" --set-int "Clock=0" --set-int "MOSI=1" --set-int "MISO=2" --set-int "Enable=3"`
  - Output: `analyzer_id=10009`
- Removed the analyzer:
  - `go run ./cmd/salad --host 127.0.0.1 --port 10430 --timeout 5s analyzer remove --capture-id 2 --analyzer-id 10009`
  - Output: `ok`
- Cleaned up the capture:
  - `go run ./cmd/salad --host 127.0.0.1 --port 10430 --timeout 5s capture stop --capture-id 2`
  - `go run ./cmd/salad --host 127.0.0.1 --port 10430 --timeout 5s capture close --capture-id 2`

### Why
- This is the “reality check” for the verbs: our CLI must work with real Logic 2 behavior, especially around settings validation and error codes.

### What worked
- Real server reachable locally: `application_version=2.4.40` and a physical device was present.
- `AddAnalyzer` succeeded once the required channel settings were provided.
- `RemoveAnalyzer` succeeded and returned `ok`.

### What didn't work
- `AddAnalyzer` with an empty settings map failed with:
  - `rpc error: code = Aborted desc = 10: Analyzer settings errors: "Invalid channel(s)"`

### What I learned
- For `"SPI"`, the real server accepted setting keys that match the UI labels:
  - `Clock`, `MOSI`, `MISO`, `Enable` (all `int64` channel indices via `--set-int`).
- The error message does not list which keys are missing; it only reports a summary (“Invalid channel(s)”), so having templates (or at least known-good examples) is important.

### What was tricky to build
- Determining the “correct” settings keys without schema introspection; success depended on using UI-visible names exactly.

### What warrants a second pair of eyes
- Whether we should ship a small set of known-good analyzer templates (e.g. SPI/I2C/Async Serial) as a first-class feature, or keep them as `ttmp` scripts until stabilized.

### What should be done in the future
- Add a `configs/analyzers/spi.yaml` template (or equivalent) based on the proven keys from this smoke test.
- Consider improving error UX: when AddAnalyzer fails with “Invalid channel(s)”, hint that channel-selection settings must match UI names exactly.

### Code review instructions
- No code changes in this step; validate by re-running the commands above against a real server.

---

## Step 6: Create SPI template pack entry and verify it works

This step turns the successful SPI smoke-test settings into a reusable template file under `configs/analyzers/`. The goal is to make analyzer setup reproducible without having to re-type `--set-int ...` flags every time.

**Commit (code):** N/A — template + docs commit (see git history after this step is committed)

### What I did
- Added analyzer template pack directory:
  - `configs/analyzers/README.md`
  - `configs/analyzers/spi.yaml`
- Verified the template against the real server:
  - Started a manual capture (got `capture_id=4`)
  - Added SPI analyzer using `--settings-yaml .../configs/analyzers/spi.yaml` (got `analyzer_id=10017`)
  - Removed analyzer, stopped capture, closed capture

### Why
- The API does not expose schemas and error messages are not very actionable (“Invalid channel(s)”), so a known-good template is the fastest path to repeatable workflows.

### What worked
- `salad analyzer add` succeeded using only the SPI template file (no overrides needed when using channels 0..3).
- `salad analyzer remove` succeeded with the returned analyzer id.

### What didn't work
- Initial attempt failed with `Cannot switch sessions while recording` because an earlier capture (from a failed remove attempt) was still recording. Cleaning up the stale capture resolved it.

### What I learned
- Template usage still depends on capture configuration (enabled channels). The template’s default channel indices assume a capture with at least digital channels 0..3 enabled.

### What was tricky to build
- Ensuring the “make sure it works” validation includes the full loop: start capture → add via template → remove → cleanup, and handling the “stale recording session” gotcha.

### What warrants a second pair of eyes
- Whether template defaults should be “safe but maybe not useful” (e.g., only Clock=0) vs “commonly useful” (Clock/MOSI/MISO/Enable = 0..3).

### What should be done in the future
- Add a SPI template variant that omits `Enable` (common SPI configs) if the real server accepts it.
- Add templates for `I2C` and `Async Serial` once we have known-good keys from real-server smoke tests.

### Code review instructions
- Inspect templates:
  - `configs/analyzers/spi.yaml`
  - `configs/analyzers/README.md`
- Validate with the real server:
  - Start capture (ensure channels 0..3 are enabled)
  - `salad analyzer add --settings-yaml .../configs/analyzers/spi.yaml`

---

## Step 7: Recreate multiple SPI analyzers on an existing stopped capture (UI verification)

This step validates the “real workflow” you’ll use day-to-day: keep a capture open in Logic 2, ensure it is **stopped** (not recording), then add multiple analyzers with sensible labels so they’re easy to recognize in the UI.

**Commit (code):** N/A — runtime operation only

### What I did
- Ensured capture 6 was stopped:
  - `go run ./cmd/salad --host 127.0.0.1 --port 10430 --timeout 5s capture stop --capture-id 6`
- Added three SPI analyzers using the SPI template:
  - `SPI: CLK0 MOSI1 MISO2 CS3` → `analyzer_id=10028`
  - `SPI: flash` → `analyzer_id=10031`
  - `SPI: sensor` → `analyzer_id=10034`

### Why
- When you’re verifying in the Logic 2 UI, distinct labels make it obvious which analyzer was created by which template/command.

### What worked
- All three analyzers were created successfully on the stopped capture without session/recording issues.

### What didn't work
- N/A

### What I learned
- As long as the capture is **stopped**, adding analyzers is stable and doesn’t trip “switch sessions while recording” errors.

### What was tricky to build
- The main gotcha is operational: if a prior test leaves a capture recording, subsequent operations can fail in confusing ways. Stopping before analyzer work avoids that.

### What warrants a second pair of eyes
- N/A

### What should be done in the future
- Add a small helper command or script that “stop capture if running” before analyzer operations to avoid this class of UI confusion.

### Code review instructions
- N/A (no code changes). Verify in the Logic 2 UI:
  - In the Analyzers panel, you should see labels: `SPI: CLK0 MOSI1 MISO2 CS3`, `SPI: flash`, `SPI: sensor`.

---

## Step 8: Generate templates from `meta.json` and smoke test on a brand new capture

This step validates a practical “UI → file → automation” loop: configure analyzers in the Logic 2 UI, save the session as `.sal`, extract `meta.json`, then generate settings templates that can be re-applied to a brand new capture using `salad analyzer add`. This is the most reliable way to get exact dropdown strings (CPOL/CPHA, Bits per Transfer, etc.) without manual copying.

**Commit (code):** N/A — runtime smoke test (templates were generated from `/tmp/meta.json`).

### What I did
- Used `/tmp/meta.json` (unzipped from `/tmp/Session 6.sal`) to generate two templates:
  - `configs/analyzers/spi-from-session6.yaml`
  - `configs/analyzers/i2c-from-session6.yaml`
- Started a new manual capture (digital channels 0..3 enabled):
  - `capture_id=7`
- Added analyzers using those templates:
  - `SPI` with label `from-meta: spi` → `analyzer_id=10038`
  - `I2C` with label `from-meta: i2c` → `analyzer_id=10041`
- Removed both analyzers, then stopped and closed the capture.

### Why
- The gRPC automation API has no method to read back analyzer settings; `meta.json` provides a repeatable extraction path for “known-good” UI configs.

### What worked
- `AddAnalyzer` succeeded for both SPI and I2C using only the extracted templates.
- `RemoveAnalyzer` succeeded for both analyzer ids.

### What didn't work
- A *second* stop attempt after closing the capture returned “Capture Id does not exist”, which is expected once the capture is closed.

### What I learned
- `meta.json` includes:
  - setting keys (UI titles), and
  - dropdown option strings (`dropdownText`) plus the selected value,
  which makes it an excellent source for authoring templates.

### What was tricky to build
- Ensuring dropdown selections are emitted as strings matching the UI text (per Saleae automation docs), not internal numeric codes.

### What warrants a second pair of eyes
- Confirm whether all analyzers accept dropdown selections as UI strings consistently across Logic 2 versions.

### What should be done in the future
- Consider a first-class command `salad analyzer template import --meta-json ...` if this workflow becomes common (optional; script-based is fine).

### Code review instructions
- With a real server running:
  - Start capture with channels 0..3 enabled.
  - Add SPI: `salad analyzer add --capture-id <id> --name "SPI" --settings-yaml .../configs/analyzers/spi-from-session6.yaml`
  - Add I2C: `salad analyzer add --capture-id <id> --name "I2C" --settings-yaml .../configs/analyzers/i2c-from-session6.yaml`

