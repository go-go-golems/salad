---
Title: Mock Saleae Server for Testing
Ticket: 010-MOCK-SERVER
Status: complete
Topics:
    - go
    - saleae
    - testing
    - mock
    - grpc
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: gen/saleae/automation/saleae_grpc.pb.go
      Note: ManagerServer interface that mock must implement
    - Path: internal/saleae/client.go
      Note: Client code that will connect to mock server
    - Path: proto/saleae/grpc/saleae.proto
      Note: Proto definition showing all RPCs
    - Path: ttmp/2025/12/24/002-CAPTURE-START--capture-start-manual-timed-trigger/index.md
      Note: Related ticket - StartCapture implementation that needs testing
    - Path: ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/analysis/03-mock-server-yaml-dsl-configurable-behavior-scenarios.md
      Note: YAML DSL design for configuring mock behavior
    - Path: ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/analysis/04-mapping-yaml-dsl-to-go-structures-validation-and-behavior-composition.md
      Note: General method for YAML→Go mapping (config→plan→pipeline)
    - Path: ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/analysis/05-bug-report-faults-yaml-does-not-inject-savecapture-unavailable-capture-10-not-found.md
      Note: Bug report and investigation plan for faults scenario
    - Path: ttmp/2025/12/27/010-MOCK-SERVER--mock-saleae-server-for-testing/tasks.md
      Note: Simplified capability-based tasks
ExternalSources: []
Summary: ""
LastUpdated: 2025-12-27T16:36:54.756920496-05:00
WhatFor: ""
WhenToUse: ""
---







# Mock Saleae Server for Testing

## Overview

Implement a mock Saleae Logic 2 gRPC server that allows testing salad commands without requiring a real Logic 2 instance or physical hardware. The mock server implements the `Manager` service interface and tracks state (devices, captures, analyzers) in memory.

## Key Links

- **Design document**: [analysis/01-mock-server-design.md](./analysis/01-mock-server-design.md)
- **CLI verbs analysis**: [analysis/02-cli-verbs-to-grpc-methods-mapping.md](./analysis/02-cli-verbs-to-grpc-methods-mapping.md)
- **YAML DSL design**: [analysis/03-mock-server-yaml-dsl-configurable-behavior-scenarios.md](./analysis/03-mock-server-yaml-dsl-configurable-behavior-scenarios.md)
- **YAML→Go mapping method**: [analysis/04-mapping-yaml-dsl-to-go-structures-validation-and-behavior-composition.md](./analysis/04-mapping-yaml-dsl-to-go-structures-validation-and-behavior-composition.md)
- **Implementation diary**: [reference/01-diary.md](./reference/01-diary.md)
- **Tasks**: [tasks.md](./tasks.md)

## Status

Current status: **active** — Design phase complete, ready for implementation

## Progress

- ✅ Codebase analysis complete
- ✅ Server interface requirements documented
- ✅ State management design complete
- ✅ RPC implementation patterns documented
- ✅ Testing strategy defined
- ✅ CLI verbs analysis complete (8 commands mapped to 9 RPCs)
- ✅ YAML DSL design complete (fixtures + policies + fault injection + scenarios)
- ✅ Implementation tasks created (simplified capability milestones)
- 🔄 Implementation pending

## Topics

- go
- saleae
- testing
- mock
- grpc

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.
