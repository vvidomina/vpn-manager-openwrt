package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersionFlag(t *testing.T) {
	var buf bytes.Buffer
	if code := run([]string{"-version"}, &buf); code != 0 {
		t.Fatalf("код выхода = %d, ожидался 0", code)
	}
	if !strings.Contains(buf.String(), Version) {
		t.Fatalf("версия не напечатана: %q", buf.String())
	}
}

func TestUnknownFlag(t *testing.T) {
	var buf bytes.Buffer
	if code := run([]string{"-nftables"}, &buf); code != 2 {
		t.Fatalf("код выхода = %d, ожидался 2", code)
	}
}

func TestBadLogLevel(t *testing.T) {
	var buf bytes.Buffer
	if code := run([]string{"-log-level", "trace"}, &buf); code != 2 {
		t.Fatalf("код выхода = %d, ожидался 2", code)
	}
}

func TestInvalidConfigFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"schema_version":9}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if code := run([]string{"-config", path, "-state-dir", dir}, &buf); code != 1 {
		t.Fatalf("код выхода = %d, ожидался 1: %s", code, buf.String())
	}
	if !strings.Contains(buf.String(), "VM_ERR_VALIDATION") {
		t.Fatalf("в журнале нет кода ошибки: %q", buf.String())
	}
}
