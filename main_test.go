package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var binPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "odota_cli_test")
	if err != nil {
		panic(err)
	}
	binPath = filepath.Join(dir, "odota_cli")
	out, err := exec.Command("go", "build", "-o", binPath, ".").CombinedOutput()
	if err != nil {
		panic("go build: " + err.Error() + ": " + string(out))
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func runBin(t *testing.T, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	var so, se bytes.Buffer
	cmd := exec.Command(binPath, args...)
	cmd.Stdout, cmd.Stderr = &so, &se
	err := cmd.Run()
	if err != nil {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("run: %v", err)
		}
		code = ee.ExitCode()
	}
	return so.String(), se.String(), code
}

func TestVersionFlag(t *testing.T) {
	so, _, code := runBin(t, "--version")
	if code != 0 || strings.TrimSpace(so) != "dev" {
		t.Errorf("--version: code=%d out=%q", code, so)
	}
}

func TestVersionStampedByLDFlags(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "odota_cli")
	out, err := exec.Command("go", "build", "-ldflags", "-X main.version=9.8.7", "-o", bin, ".").CombinedOutput()
	if err != nil {
		t.Fatalf("build: %v: %s", err, out)
	}
	got, err := exec.Command(bin, "--version").Output()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(got)) != "9.8.7" {
		t.Errorf("version = %q, want 9.8.7", got)
	}
}

func TestNoArgsExitCode(t *testing.T) {
	_, se, code := runBin(t)
	if code != 1 {
		t.Errorf("exit = %d, want 1", code)
	}
	if !strings.Contains(se, "Usage:") {
		t.Errorf("stderr lacks usage: %q", se)
	}
}

func TestMissingDemFile(t *testing.T) {
	_, se, code := runBin(t, filepath.Join(t.TempDir(), "nope.dem"))
	if code != 1 || !strings.Contains(se, "read dem file") {
		t.Errorf("code=%d stderr=%q", code, se)
	}
}

func TestInvalidNDJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.ndjson")
	if err := os.WriteFile(path, []byte("{\"type\":\"epilogue\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, se, code := runBin(t, path)
	if code != 1 || !strings.Contains(se, "parse") {
		t.Errorf("code=%d stderr=%q", code, se)
	}
}

func TestNDJSONEndToEndMatchesGolden(t *testing.T) {
	so, _, code := runBin(t, "parser/testdata/mini.ndjson")
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, so)
	}
	want, err := os.ReadFile("parser/testdata/mini.golden")
	if err != nil {
		t.Fatal(err)
	}
	if so != string(want) {
		t.Errorf("CLI stdout != golden file")
	}
}

func TestJSONExtensionUsesNDJSONReader(t *testing.T) {
	path := filepath.Join(t.TempDir(), "match.json")
	data, err := os.ReadFile("parser/testdata/mini.ndjson")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	so, _, code := runBin(t, path)
	if code != 0 {
		t.Fatalf("code=%d", code)
	}
	want, err := os.ReadFile("parser/testdata/mini.golden")
	if err != nil {
		t.Fatal(err)
	}
	if so != string(want) {
		t.Errorf("json-dispatch stdout != golden file")
	}
}
