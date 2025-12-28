# Tasks

## TODO

- [x] Update analysis doc (006) to match current codebase (real file paths, symbols, current constraints)
- [x] Add 006 diary entries while researching/implementing
- [x] Decide CLI surface: **root-level** `salad run/repro/watch` (start with `run`; repro/watch later)
- [x] Implement pipeline config model + loader (`internal/pipeline`, YAML/JSON)
- [x] Implement `salad run --config <file>` (v1: capture load → analyzers add → exports → close)
- [x] Add at least one minimal pipeline config example under `ttmp/.../006.../scripts/` (or `configs/pipelines/`)
- [ ] Add integration test(s) against mock server (reuse `configs/mock/happy-path.yaml` and existing smoke scripts)
- [x] Add real-server validation script/playbook (optional, but recommended) mirroring ticket 005’s approach
- [ ] Commit when `go test ./...` is green; commit docs updates separately

