package vmerrors

import (
	"errors"
	"strings"
	"testing"
)

func TestCodeOfWrapped(t *testing.T) {
	inner := New(CodeStorage, "storage.read", "нет доступа")
	wrapped := Wrap(CodeInternal, "core.start", inner)
	if got := CodeOf(wrapped); got != CodeInternal {
		t.Fatalf("код = %q, ожидался VM_ERR_INTERNAL", got)
	}
	if !Is(wrapped, CodeInternal) {
		t.Fatal("Is не распознал код")
	}
	if !errors.Is(wrapped, inner) {
		t.Fatal("цепочка Unwrap нарушена")
	}
	if !strings.Contains(wrapped.Error(), "VM_ERR_INTERNAL") || !strings.Contains(wrapped.Error(), "storage.read") {
		t.Fatalf("сообщение не содержит контекст: %q", wrapped.Error())
	}
}

func TestPlainErrorHasNoCode(t *testing.T) {
	if got := CodeOf(errors.New("обычная ошибка")); got != "" {
		t.Fatalf("код обычной ошибки = %q, ожидалась пустая строка", got)
	}
}

func TestUnsupported(t *testing.T) {
	err := Unsupported("xray.start", "Xray Manager")
	if !Is(err, CodeUnsupported) || !strings.Contains(err.Error(), "этапе 1") {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
}

func TestNilSafe(t *testing.T) {
	if CodeOf(nil) != "" {
		t.Fatal("CodeOf(nil) должен быть пустым")
	}
}
