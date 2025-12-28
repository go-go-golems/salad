package saleae

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestCLI_ExportTable_AgainstMockServer(t *testing.T) {
	// Locate salad module root from this test file location: internal/mock/saleae -> ../../..
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	moduleRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))

	// Start mock server from existing happy path config.
	cfgPath := filepath.Join(moduleRoot, "configs", "mock", "happy-path.yaml")
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig(%s): %v", cfgPath, err)
	}
	// Ensure the export writes a placeholder file so we can assert it exists.
	cfg.Behavior.ExportDataTableCsv.SideEffect.WritePlaceholderFile = ptrBool(true)
	cfg.Behavior.ExportDataTableCsv.SideEffect.IncludeRequestInFile = ptrBool(true)

	plan, err := Compile(cfg)
	if err != nil {
		t.Fatalf("Compile(%s): %v", cfgPath, err)
	}

	_, _, listener, cleanup, err := StartMockServer(plan)
	if err != nil {
		t.Fatalf("StartMockServer: %v", err)
	}
	defer cleanup()

	host, port := splitHostPort(t, listener.Addr().String())

	// Build salad binary (faster + less noisy than running "go run" repeatedly).
	bin := filepath.Join(t.TempDir(), "salad")
	buildSaladBinary(t, moduleRoot, bin)

	// 1) Create a capture via CLI (LoadCapture).
	loadOut := runCLI(t, bin, []string{
		"--host", host,
		"--port", strconv.Itoa(port),
		"--timeout", "2s",
		"capture", "load",
		"--filepath", "/tmp/mock.sal",
	})
	captureID := parseUint64KV(t, loadOut, "capture_id")

	// 2) Add analyzer to capture so we have a real analyzer_id to export.
	spiTemplate := filepath.Join(moduleRoot, "configs", "analyzers", "spi.yaml")
	addOut := runCLI(t, bin, []string{
		"--host", host,
		"--port", strconv.Itoa(port),
		"--timeout", "2s",
		"analyzer", "add",
		"--capture-id", strconv.FormatUint(captureID, 10),
		"--name", "SPI",
		"--label", "template",
		"--settings-yaml", spiTemplate,
	})
	analyzerID := parseUint64KV(t, addOut, "analyzer_id")
	if analyzerID == 0 {
		t.Fatalf("expected analyzer_id to be non-zero")
	}

	// 3) Export decoded data table to CSV.
	outPath := filepath.Join(t.TempDir(), "out.csv")
	_ = runCLI(t, bin, []string{
		"--host", host,
		"--port", strconv.Itoa(port),
		"--timeout", "2s",
		"export", "table",
		"--capture-id", strconv.FormatUint(captureID, 10),
		"--filepath", outPath,
		"--analyzer", strconv.FormatUint(analyzerID, 10) + ":hex",
		"--iso8601-timestamp",
		"--columns", "time,data,address",
		"--filter-query", "0xAA",
		"--filter-columns", "data,address",
	})

	// Assert file exists and contains placeholder marker.
	payload, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read exported file %s: %v", outPath, err)
	}
	s := string(payload)
	if !strings.Contains(s, "SALAD_MOCK_DATA_TABLE_CSV") {
		t.Fatalf("expected placeholder marker in %s, got:\n%s", outPath, s)
	}
	if !strings.Contains(s, "filter.query=0xAA") {
		t.Fatalf("expected filter marker in %s, got:\n%s", outPath, s)
	}
}

func ptrBool(v bool) *bool { return &v }

func buildSaladBinary(t *testing.T, moduleRoot string, outPath string) {
	t.Helper()
	// XXX: If we consolidate helpers, we can remove this and share build logic across tests.
	buildSaladBinaryInternal(t, moduleRoot, outPath)
}

func buildSaladBinaryInternal(t *testing.T, moduleRoot string, outPath string) {
	t.Helper()
	build := exec.Command("go", "build", "-o", outPath, "./cmd/salad")
	build.Dir = moduleRoot
	build.Env = append(os.Environ(), "GOWORK=off")
	var combined bytes.Buffer
	build.Stdout = &combined
	build.Stderr = &combined
	if err := build.Run(); err != nil {
		t.Fatalf("go build ./cmd/salad: %v\n%s", err, combined.String())
	}
}
