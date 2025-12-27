# Mock Saleae Server Developer Guide

## Goal

This document explains how to extend the YAML-driven mock Saleae gRPC server, including:
- adding new config knobs,
- compiling YAML into runtime plans,
- wiring new RPC handlers, and
- updating example scenarios.

## Architecture summary

1. **Config layer** (`internal/mock/saleae/config.go`)
   - YAML-facing structs with simple types and optional fields.
   - Parsed via `LoadConfig` with `KnownFields(true)`.

2. **Plan compiler** (`internal/mock/saleae/plan.go`)
   - `Compile` validates version, applies defaults, and normalizes enums.
   - Converts YAML strings to typed enums and `codes.Code` values.

3. **Runtime server** (`internal/mock/saleae/server.go`)
   - `Server` holds state, compiled plan, and a shared exec wrapper.
   - Each RPC uses `exec` to apply faults, validation, and side effects.

4. **Side effects** (`internal/mock/saleae/side_effects.go`)
   - Pluggable interface for save/export placeholder output.

## Adding a new behavior knob

1. **Extend config structs**
   - Add fields to `BehaviorConfig` or related sub-structs in `config.go`.

2. **Compile into the plan**
   - Update `compileBehavior` in `plan.go` to apply defaults and normalization.

3. **Use in handlers**
   - Read the compiled values in the corresponding RPC handler.

4. **Add example scenario**
   - Create or update a YAML file under `configs/mock/` to demonstrate the new knob.

## Adding a new RPC handler

1. Add a new `Method` constant in `exec.go` and update `AllMethods`.
2. Extend fault matcher compilation in `plan.go` if the RPC needs request matching.
3. Implement the RPC in `server.go`, using `exec` for shared behavior.
4. Add or update scenario YAML and tests.

## Testing changes

- Unit-style checks: `go test ./...`
- Manual flow:
  - Run `salad-mock` with a scenario file.
  - Exercise the CLI with `go run ./cmd/salad ...` to validate behavior.

## Files to start with

- `internal/mock/saleae/config.go`
- `internal/mock/saleae/plan.go`
- `internal/mock/saleae/server.go`
- `internal/mock/saleae/side_effects.go`
- `configs/mock/happy-path.yaml`
