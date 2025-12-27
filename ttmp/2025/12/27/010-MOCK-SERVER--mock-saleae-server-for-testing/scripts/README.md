# Scripts

These scripts exist to make it easy to **start/stop** the mock server and **repeat** common validation runs without retyping long commands.

## Go checks (module-local)

Run formatting + compile checks for the `salad` module:

```bash
./01-go-check.sh
```

Notes:
- Uses `GOWORK=off` because repo root `go.work` does not include the `salad` module.
- `gofmt` check is intentionally scoped to `cmd/salad`, `cmd/salad-mock`, `internal/mock/saleae`, `internal/saleae`.

## Smoke tests (start server, run CLI, stop server)

Happy path:

```bash
./02-smoke-happy-path.sh
```

Fault injection scenario:

```bash
./03-smoke-faults.sh
```

## tmux-based server lifecycle (recommended)

The most common source of confusion is “a previous `salad-mock` is still running on the port”.
These scripts make it easy to manage server lifecycle explicitly in a tmux session and keep logs.

### Start

```bash
./10-tmux-mock-start.sh
```

Override defaults via env:

```bash
CFG=configs/mock/faults.yaml PORT=10432 SESSION=salad-mock-faults ./10-tmux-mock-start.sh
```

### Tail logs

```bash
PORT=10431 ./13-tmux-mock-tail-log.sh
```

### Stop

```bash
./11-tmux-mock-stop.sh
```

### Restart

```bash
./12-tmux-mock-restart.sh
```

### Kill a stuck listener (port cleanup)

If a non-tmux `salad-mock` is still holding the port:

```bash
PORT=10431 ./09-kill-mock-on-port.sh
```

## Logs

Server output is written to:
- `scripts/logs/salad-mock-<port>.log`

The `scripts/logs/` directory is tracked, but actual `*.log` files are ignored.


