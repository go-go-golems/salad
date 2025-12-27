# Changelog

## 2025-12-24

- Initial workspace created


## 2025-12-24

Added initial implementation analysis doc (proto mapping, CLI UX, files/tests).


## 2025-12-27

Expanded implementation doc with mock-server requirements to support analyzer testing (AddAnalyzer/RemoveAnalyzer): RPCs, state model, YAML DSL knobs, and test strategy.

Implemented analyzer verbs + settings parsing and validated against a real Logic 2 server:
- `Saleae client: add AddAnalyzer/RemoveAnalyzer` (commit a574108c310947e3d47b71ad697d997fa22838f9)
- `Config: parse analyzer settings (json/yaml + typed overrides)` (commit e8a1d3c254871a18feb268458103995576a1e61e)
- `CLI: add analyzer add/remove commands` (commit 99d3b4b004c3836b885daacd641dad748789d67d)
- Real-server smoke test succeeded for `SPI` using `--set-int Clock=0 MOSI=1 MISO=2 Enable=3` (see diary Step 5; commit 5f123f3d169f8f090c959afee518b9b4d12a9a21 records the run)

