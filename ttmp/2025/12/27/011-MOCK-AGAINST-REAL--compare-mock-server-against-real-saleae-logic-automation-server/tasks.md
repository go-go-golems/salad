# Tasks

## TODO

- [x] Confirm the real Logic 2 automation server endpoint (host/port) and capture baseline outputs (probe JSON under `various/`)
- [ ] Define a “comparison suite” (minimal set of probes / `salad` flows) that can run against **both** endpoints and document required preconditions (real hardware vs fixtures)
- [x] Add a repeatable comparison playbook/script (start mock, probe both, store output under `various/`)
- [x] Run the suite against real vs mock (`salad/configs/mock/happy-path.yaml`) and record differences with evidence
- [ ] Classify differences: intentional divergence vs mock bug; file follow-up tickets for mock fixes as needed
- [ ] (Optional, but recommended) Add an RPC-level transcript harness (record/replay) to make this comparison a regression test

