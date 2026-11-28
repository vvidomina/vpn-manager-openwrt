package logging

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	log := NewWithClock(&buf, LevelInfo, func() time.Time {
		return time.Date(2026, 11, 28, 12, 0, 0, 0, time.UTC)
	})
	log.Debug("скрыто")
	log.Info("видно", "k", "v")
	log.Error("ошибка")

	out := buf.String()
	if strings.Contains(out, "скрыто") {
		t.Fatal("сообщение ниже порога не должно попадать в журнал")
	}
	if !strings.Contains(out, "INFO") || !strings.Contains(out, "k=v") {
		t.Fatalf("формат сообщения нарушен: %q", out)
	}
	if !strings.Contains(out, "ERROR") {
		t.Fatalf("сообщение уровня error потеряно: %q", out)
	}
	if strings.Count(out, "\n") != 2 {
		t.Fatalf("ожидалось две строки, получено %q", out)
	}
}

func TestSanitizeSecrets(t *testing.T) {
	in := []any{"token", "abc", "password", "p", "URL", "https://x", "keep", "value"}
	out := Sanitize(in...)
	want := []any{"token", masked, "password", masked, "URL", masked, "keep", "value"}
	for i := range want {
		if out[i] != want[i] {
			t.Fatalf("позиция %d = %v, ожидалось %v", i, out[i], want[i])
		}
	}
	if in[1] != "abc" {
		t.Fatal("Sanitize изменил исходный срез")
	}
}

func TestLoggerSanitizes(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, LevelDebug).With("uuid", "secret-uuid")
	log.Info("запуск", "token", "t0ken")
	out := buf.String()
	if strings.Contains(out, "secret-uuid") || strings.Contains(out, "t0ken") {
		t.Fatalf("секрет попал в журнал: %q", out)
	}
	if !strings.Contains(out, masked) {
		t.Fatalf("маска отсутствует: %q", out)
	}
}

func TestParseLevel(t *testing.T) {
	for _, name := range []string{"debug", "info", "WARN", "error"} {
		if _, err := ParseLevel(name); err != nil {
			t.Fatalf("%q: %v", name, err)
		}
	}
	if _, err := ParseLevel("trace"); err == nil {
		t.Fatal("неизвестный уровень должен отклоняться")
	}
}
