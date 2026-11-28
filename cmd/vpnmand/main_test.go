package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/vvidomina/vpn-manager-openwrt/internal/logging"
)

// syncBuffer безопасен для чтения при записи из другой горутины.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// closedChannel завершает ожидание демона в тестах без реальных сигналов.
func closedChannel() <-chan struct{} {
	done := make(chan struct{})
	close(done)
	return done
}

func writeConfig(t *testing.T, dir, body string) string {
	t.Helper()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

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

// Невалидная конфигурация больше не завершает демон: процесс работает в
// диагностическом режиме, причина видна в журнале, код выхода 0.
func TestInvalidConfigKeepsDaemonRunning(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, `{"schema_version":9}`)

	var buf syncBuffer
	code := runWith([]string{"-config", path, "-state-dir", dir}, &buf, closedChannel())
	if code != 0 {
		t.Fatalf("код выхода = %d, ожидался 0 (демон не должен падать): %s", code, buf.String())
	}
	out := buf.String()
	if !strings.Contains(out, "VM_ERR_VALIDATION") {
		t.Fatalf("в журнале нет кода ошибки: %q", out)
	}
	if !strings.Contains(out, "DEGRADED") {
		t.Fatalf("состояние DEGRADED не опубликовано: %q", out)
	}
	if !strings.Contains(out, "ready=false") {
		t.Fatalf("демон не должен сообщать о готовности: %q", out)
	}
}

func TestMissingConfigUsesDefaults(t *testing.T) {
	dir := t.TempDir()

	var buf syncBuffer
	code := runWith([]string{"-config", filepath.Join(dir, "нет.json"), "-state-dir", dir}, &buf, closedChannel())
	if code != 0 {
		t.Fatalf("код выхода = %d, ожидался 0: %s", code, buf.String())
	}
	out := buf.String()
	if !strings.Contains(out, "значения по умолчанию") {
		t.Fatalf("первый запуск не отражён в журнале: %q", out)
	}
	if !strings.Contains(out, "DISABLED_BY_USER") {
		t.Fatalf("состояние по умолчанию не опубликовано: %q", out)
	}
}

func TestValidConfigStarts(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, `{"schema_version":1,"intent":"enabled"}`)

	var buf syncBuffer
	code := runWith([]string{"-config", path, "-state-dir", dir, "-log-level", "debug"}, &buf, closedChannel())
	if code != 0 {
		t.Fatalf("код выхода = %d, ожидался 0: %s", code, buf.String())
	}
	out := buf.String()
	if !strings.Contains(out, "STARTING") {
		t.Fatalf("состояние STARTING не опубликовано: %q", out)
	}
	if !strings.Contains(out, "health-ping") {
		t.Fatalf("внутренний health-ping не выполнен: %q", out)
	}
}

// SIGHUP журналируется и не завершает демон; SIGINT/SIGTERM остаются
// штатным завершением через signal.NotifyContext.
func TestSIGHUPIsLoggedAndDoesNotStopDaemon(t *testing.T) {
	var buf syncBuffer
	log := logging.New(&buf, logging.LevelInfo)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hup := make(chan os.Signal, 1)
	finished := make(chan struct{})
	go func() {
		waitForSignals(ctx, log, hup)
		close(finished)
	}()

	hup <- syscall.SIGHUP

	deadline := time.Now().Add(2 * time.Second)
	for !strings.Contains(buf.String(), "SIGHUP") {
		if time.Now().After(deadline) {
			t.Fatalf("SIGHUP не зарегистрирован в журнале: %q", buf.String())
		}
		time.Sleep(5 * time.Millisecond)
	}
	select {
	case <-finished:
		t.Fatal("демон завершился по SIGHUP")
	default:
	}

	cancel()
	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		t.Fatal("ожидание сигналов не завершилось после отмены контекста")
	}
}

func TestSIGTERMStopsDaemon(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, `{"schema_version":1,"intent":"disabled"}`)

	var buf syncBuffer
	code := runWith([]string{"-config", path, "-state-dir", dir}, &buf, closedChannel())
	if code != 0 {
		t.Fatalf("код выхода = %d, ожидался 0: %s", code, buf.String())
	}
	out := buf.String()
	if !strings.Contains(out, "получен сигнал завершения") {
		t.Fatalf("завершение не зарегистрировано: %q", out)
	}
	if !strings.Contains(out, "демон остановлен") {
		t.Fatalf("остановка не завершена: %q", out)
	}
}
