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

func TestCLI_PipelineRun_AgainstMockServer(t *testing.T) {
	// Locate salad module root from this test file location: internal/mock/saleae -> ../../..
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	moduleRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))

	// Start mock server from existing happy path config and enable table export placeholders.
	cfgPath := filepath.Join(moduleRoot, "configs", "mock", "happy-path.yaml")
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig(%s): %v", cfgPath, err)
	}
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
	build := exec.Command("go", "build", "-o", bin, "./cmd/salad")
	build.Dir = moduleRoot
	build.Env = append(os.Environ(), "GOWORK=off")
	var combined bytes.Buffer
	build.Stdout = &combined
	build.Stderr = &combined
	if err := build.Run(); err != nil {
		t.Fatalf("go build ./cmd/salad: %v\n%s", err, combined.String())
	}

	// Create a pipeline config file to run against the mock.
	outDir := filepath.Join(t.TempDir(), "out")
	rawDir := filepath.Join(outDir, "raw")
	tablePath := filepath.Join(outDir, "table.csv")
	pipelinePath := filepath.Join(t.TempDir(), "pipeline.yaml")

	// Use an analyzer settings file that exists in the repo.
	settingsYAML := filepath.Join(moduleRoot, "configs", "analyzers", "spi.yaml")

	pipelineYAML := strings.Join([]string{
		"version: 1",
		"capture:",
		"  load:",
		"    filepath: /tmp/mock.sal",
		"analyzers:",
		"  - name: \"SPI\"",
		"    label: \"spi\"",
		"    settings_yaml: \"" + strings.ReplaceAll(settingsYAML, "\\", "\\\\") + "\"",
		"exports:",
		"  - type: raw-csv",
		"    directory: \"" + strings.ReplaceAll(rawDir, "\\", "\\\\") + "\"",
		"    digital: [0, 1, 2]",
		"    iso8601_timestamp: true",
		"  - type: table-csv",
		"    filepath: \"" + strings.ReplaceAll(tablePath, "\\", "\\\\") + "\"",
		"    iso8601_timestamp: true",
		"    analyzers:",
		"      - ref: \"spi\"",
		"        radix: hex",
		"    filter:",
		"      query: \"0xAA\"",
		"      columns: [\"data\"]",
		"cleanup:",
		"  close_capture: true",
		"",
	}, "\n")

	if err := os.WriteFile(pipelinePath, []byte(pipelineYAML), 0o644); err != nil {
		t.Fatalf("write pipeline config %s: %v", pipelinePath, err)
	}

	// Run the pipeline via CLI.
	out := runCLI(t, bin, []string{
		"--host", host,
		"--port", strconv.Itoa(port),
		"--timeout", "3s",
		"run",
		"--config", pipelinePath,
	})
	if !strings.Contains(out, "ok") {
		t.Fatalf("expected ok output, got:\n%s", out)
	}

	// Verify placeholders were written by the mock server.
	digitalPath := filepath.Join(rawDir, "digital.csv")
	bDigital, err := os.ReadFile(digitalPath)
	if err != nil {
		t.Fatalf("read %s: %v", digitalPath, err)
	}
	if !strings.Contains(string(bDigital), "SALAD_MOCK_DIGITAL_CSV") {
		t.Fatalf("expected digital placeholder marker in %s, got:\n%s", digitalPath, string(bDigital))
	}

	bTable, err := os.ReadFile(tablePath)
	if err != nil {
		t.Fatalf("read %s: %v", tablePath, err)
	}
	if !strings.Contains(string(bTable), "SALAD_MOCK_DATA_TABLE_CSV") {
		t.Fatalf("expected table placeholder marker in %s, got:\n%s", tablePath, string(bTable))
	}
	if !strings.Contains(string(bTable), "filter.query=0xAA") {
		t.Fatalf("expected filter marker in %s, got:\n%s", tablePath, string(bTable))
	}
}
