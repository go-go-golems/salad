# Tasks

## TODO

- [x] Add tasks here

- [x] Pin and vendor Saleae Logic2 automation proto (saleae.proto) + record upstream commit
- [x] Add Go module + deps (cobra, grpc, zerolog, pkg/errors)
- [x] Generate and commit Go gRPC bindings from saleae.proto (or document generation requirements)
- [x] Implement grpc dial + Saleae client wrapper (context, timeouts, errors wrapping)
- [ ] Implement CLI commands: appinfo (first), then capture/analyzer/export skeletons
- [x] Write a minimal manual test playbook (start Logic2 with --automation, run appinfo)
- [x] Implement devices command (GetDevices)
- [x] Live-test CLI against real Logic2 automation server
