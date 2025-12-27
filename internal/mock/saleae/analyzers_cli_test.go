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

func TestCLI_AnalyzerAddRemove_AgainstMockServer(t *testing.T) {
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
	out, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("go build ./cmd/salad: %v\n%s", err, string(out))
	}

	// 1) Create a capture via CLI (LoadCapture).
	loadOut := runCLI(t, bin, []string{
		"--host", host,
		"--port", strconv.Itoa(port),
		"--timeout", "2s",
		"capture", "load",
		"--filepath", "/tmp/mock.sal",
	})
	captureID := parseUint64KV(t, loadOut, "capture_id")

	// 2) Add analyzer using the SPI template settings file (from repo).
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

	// 3) Remove analyzer.
	_ = runCLI(t, bin, []string{
		"--host", host,
		"--port", strconv.Itoa(port),
		"--timeout", "2s",
		"analyzer", "remove",
		"--capture-id", strconv.FormatUint(captureID, 10),
		"--analyzer-id", strconv.FormatUint(analyzerID, 10),
	})

	// 4) Removing again should fail by default (RequireAnalyzerExists=true).
	cmd := exec.Command(bin,
		"--host", host,
		"--port", strconv.Itoa(port),
		"--timeout", "2s",
		"analyzer", "remove",
		"--capture-id", strconv.FormatUint(captureID, 10),
		"--analyzer-id", strconv.FormatUint(analyzerID, 10),
	)
	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined
	err = cmd.Run()
	if err == nil {
		t.Fatalf("expected second remove to fail, got success output:\n%s", combined.String())
	}
	if !strings.Contains(combined.String(), "RemoveAnalyzer") && !strings.Contains(strings.ToLower(combined.String()), "analyzer") {
		t.Fatalf("expected analyzer-related error output, got:\n%s", combined.String())
	}
}

func runCLI(t *testing.T, bin string, args []string) string {
	t.Helper()
	cmd := exec.Command(bin, args...)
	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined
	if err := cmd.Run(); err != nil {
		t.Fatalf("command failed: %s %s\nerr=%v\noutput:\n%s", bin, strings.Join(args, " "), err, combined.String())
	}
	return combined.String()
}

func parseUint64KV(t *testing.T, output string, key string) uint64 {
	t.Helper()
	// output is expected to include a line like: key=123
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, key+"=") {
			raw := strings.TrimPrefix(line, key+"=")
			v, err := strconv.ParseUint(raw, 10, 64)
			if err != nil {
				t.Fatalf("parse %s from %q: %v", key, line, err)
			}
			return v
		}
	}
	t.Fatalf("did not find %s=... in output:\n%s", key, output)
	return 0
}

func splitHostPort(t *testing.T, addr string) (string, int) {
	t.Helper()
	host, portStr, ok := strings.Cut(addr, ":")
	if !ok {
		t.Fatalf("invalid listener addr %q", addr)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("parse port from %q: %v", addr, err)
	}
	return host, port
}
