# Tasks

## TODO

- [x] Add tasks here

- [ ] Phase 1: Add YAML parsing dependency (gopkg.in/yaml.v3) to go.mod
- [ ] Phase 1: Create internal/config package directory structure
- [ ] Phase 1: Define config structs in internal/config/capture_start.go (CaptureStartConfig, DeviceConfig, ChannelsConfig, CaptureConfig, etc.) mirroring YAML structure
- [ ] Phase 1: Implement LoadCaptureStartConfig function supporting both YAML and JSON formats
- [ ] Phase 1: Implement Validate method on CaptureStartConfig (channels, sample rates, mode-specific validation, pulse trigger rules)
- [ ] Phase 1: Implement ToProto method on CaptureStartConfig converting to *pb.StartCaptureRequest with correct oneof wrapper types
- [ ] Phase 1: Write unit tests for config parsing (YAML and JSON, valid and invalid cases)
- [ ] Phase 1: Write unit tests for validation (all error cases: missing channels, invalid sample rates, mode mismatches, pulse trigger validation)
- [ ] Phase 1: Write unit tests for proto conversion (verify oneof fields set correctly, enum conversions, all three capture modes)
- [ ] Phase 2: Add StartCapture method to internal/saleae/client.go following existing pattern (context handling, validation, error wrapping, nil checks)
- [ ] Phase 2: Return uint64 capture_id from StartCapture for consistency with other client methods
- [ ] Phase 3: Create cmd/salad/cmd/capture_start.go following captureLoadCmd pattern
- [ ] Phase 3: Add --config flag (required) to capture start command
- [ ] Phase 3: Wire up CLI command flow: parse config → validate → convert to proto → call client → output capture_id
- [ ] Phase 3: Register capture start command in cmd/salad/cmd/root.go
- [ ] Phase 4: Manual smoke test with real Logic 2 instance - manual capture mode
- [ ] Phase 4: Manual smoke test with real Logic 2 instance - timed capture mode
- [ ] Phase 4: Manual smoke test with real Logic 2 instance - digital trigger capture mode
- [ ] Phase 4: Test error cases (invalid config, missing device, invalid sample rates)
- [ ] Phase 4: Create example config files for each capture mode (manual.yaml, timed.yaml, trigger.yaml)
- [ ] [OPTIONAL] Future: Implement flag override system with --device-* and --capture-* flags following hierarchical naming
- [ ] [OPTIONAL] Future: Implement config-flag merge logic (config loads first, flags override)
- [ ] [OPTIONAL] Future: Add flag parsing for array fields (channels CSV, glitch filters structured format)
- [ ] [OPTIONAL] Future: Handle mode switching via flags (clear conflicting mode-specific fields when --capture-mode changes)
- [ ] [OPTIONAL] Future: Add validation after config-flag merge (validate merged result, not just individual parts)
