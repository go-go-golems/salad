# Tasks

The goal here is a **small number of tasks**, each representing a meaningful capability. Subtasks are intentionally avoided; details live in the diary and code.

## CLI verbs + settings plumbing (core deliverable)

- [x] Implement Saleae client wrappers for analyzer RPCs (`AddAnalyzer`, `RemoveAnalyzer`)
- [x] Implement analyzer settings parsing (JSON/YAML) + typed overrides (`--set*`)
- [x] Add `salad analyzer add/remove` Cobra verbs and wire into root
- [x] Smoke test analyzer add/remove against a real Logic 2 Automation server

## Mock + tests (to make it CI-friendly)

- [x] Extend the mock server (ticket 010) to implement `AddAnalyzer`/`RemoveAnalyzer` + in-memory analyzer state
- [x] Add table-driven tests that run the CLI against the mock for:
  - add success (returns analyzer_id)
  - remove success
  - missing capture id behavior
  - settings parsing / override precedence sanity

## Templates (optional, high leverage)

- [x] Add a minimal template pack (starting with SPI) based on the real-server smoke test:
  - `configs/analyzers/spi.yaml` (keys: `Clock`, `MOSI`, `MISO`, `Enable`)

