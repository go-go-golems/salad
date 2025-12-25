You can treat Logic 2 like a **local gRPC server** that your Go CLI talks to (Logic must be running; the API can’t talk to the hardware without the app). ([Saleae Support][1])

## 1) Where to get the protos

Saleae publishes the proto(s) in their repo `saleae/logic2-automation` (it contains the gRPC `.proto` and a Python wrapper). ([GitHub][2])

The file you’ll vendor/submodule is:

* `proto/saleae/grpc/saleae.proto` ([Saleae - Logic 2][3])

Note: Saleae explicitly says the API is beta and may have breaking changes, so **pin to a commit/tag**. ([GitHub][2])

## 2) Repo layout (simple + Go-friendly)

Example scaffold:

```
saleae-cli/
  go.mod
  cmd/saleae/
    main.go
  internal/saleae/
    client.go
  proto/                          # vendored or submodule
    saleae/grpc/saleae.proto
  gen/                            # generated Go from proto
    saleae/grpc/...
  Makefile
  tools.go
```

### Vendoring vs submodule

* **Vendoring**: copy `saleae.proto` into your `proto/` dir (fast, no git complexity).
* **Submodule**: add `saleae/logic2-automation` as a submodule and reference the proto path. (Nice if you want to update with a single pinned SHA.)

Either way, the “source of truth” is the Saleae repo. ([GitHub][2])

## 3) Generating Go bindings from `saleae.proto`

Install generators:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

Then generate (assuming you vendored to `./proto/...`):

```bash
protoc \
  -I ./proto \
  --go_out=./gen --go_opt=paths=source_relative \
  --go-grpc_out=./gen --go-grpc_opt=paths=source_relative \
  ./proto/saleae/grpc/saleae.proto
```

If `saleae.proto` **doesn’t** include an `option go_package = ...`, you may need to add `--go_opt=M...=...` mappings or use `buf` to set a package prefix. (Many people just add a tiny `go_package` line in a local copy; but then you’re maintaining a patch.)

A minimal `Makefile`:

```make
gen:
	protoc -I ./proto \
	  --go_out=./gen --go_opt=paths=source_relative \
	  --go-grpc_out=./gen --go-grpc_opt=paths=source_relative \
	  ./proto/saleae/grpc/saleae.proto

build:
	go build ./cmd/saleae
```

## 4) Talking to Logic 2 from Go

### Start Logic 2 with automation enabled (Linux)

From Saleae’s forum: launch with `--automation`, and optionally set bind host/port. ([Saleae - Logic 2][4])

Example:

```bash
Logic-2.AppImage --automation --automationPort 10430
# or for remote access:
Logic-2.AppImage --automation --automationHost 0.0.0.0 --automationPort 10430
```

([Saleae - Logic 2][4])

### Minimal Go client: `GetAppInfo`

There’s a `GetAppInfo` RPC used in community examples. ([Saleae - Logic 2][5])

```go
package main

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "yourmodule/gen/saleae/grpc" // <-- adjust to your generated package path
)

func main() {
	conn, _ := grpc.NewClient(
		"127.0.0.1:10430",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	defer conn.Close()

	client := pb.NewManagerClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	reply, _ := client.GetAppInfo(ctx, &pb.GetAppInfoRequest{})

	fmt.Printf("Logic app version: %s\n", reply.GetAppInfo().GetApplicationVersion())
	fmt.Printf("API proto version: %v\n", reply.GetAppInfo().GetApiVersion())
	fmt.Printf("App PID: %d\n", reply.GetAppInfo().GetApplicationPid())
}
```

## 5) CLI shape (subcommands that map to Saleae concepts)

The Python docs give you the “surface area” you’ll end up wrapping: `load_capture`, `start_capture`, `add_analyzer`, `export_data_table`, `export_raw_data_csv/binary`, etc. ([Saleae][6])

A practical CLI set is:

* `saleae appinfo`
* `saleae capture start ...` (timed/manual/trigger)
* `saleae capture stop --id ...`
* `saleae capture save --id ... --out capture.sal`
* `saleae capture load --file capture.sal`
* `saleae analyzer add --capture ... --type SPI --settings settings.json`
* `saleae export table --capture ... --analyzer ... --out out.csv`
* `saleae export raw-csv --capture ... --digital 0,1,2 --outdir ...`

Implement it as:

* `cmd/saleae/main.go`: argument parsing + printing
* `internal/saleae/client.go`: a thin wrapper around the generated `ManagerClient` + any “CaptureClient” service that exists in the proto.

---

If you paste the first ~60 lines of `saleae.proto` (the `service` definitions + package/options), I can tailor the exact `protoc` command (go_package/mappings) and stub out 2–3 concrete subcommands (appinfo + load + export) that compile as-is.

[1]: https://support.saleae.com/saleae-api-and-sdk?utm_source=chatgpt.com "Automation API & Analyzer SDK"
[2]: https://github.com/saleae/logic2-automation "GitHub - saleae/logic2-automation: Logic2 Automation API"
[3]: https://discuss.saleae.com/t/saleae-logic-2-automation-api/1685?page=2 "Saleae Logic 2 Automation API - Page 2 - Logic 2 Software - Saleae - Logic 2"
[4]: https://discuss.saleae.com/t/saleae-logic-2-automation-api/1685?page=5&utm_source=chatgpt.com "Saleae Logic 2 Automation API - Page 5"
[5]: https://discuss.saleae.com/t/logic-2-automation-via-c-and-net-framework-4-8/2171?utm_source=chatgpt.com "Logic 2 automation via C# and NET Framework 4.8 - Support"
[6]: https://saleae.github.io/logic2-automation/automation.html "Automation API — Saleae 1.0.6 documentation"
