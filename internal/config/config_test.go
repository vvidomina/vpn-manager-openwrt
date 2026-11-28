package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vvidomina/vpn-manager-openwrt/internal/vmerrors"
)

func TestDefault(t *testing.T) {
	cfg := Default()
	if cfg.Intent != IntentDisabled {
		t.Fatalf("intent по умолчанию = %q, ожидался disabled", cfg.Intent)
	}
	if cfg.Strategy != StrategyRandom {
		t.Fatalf("стратегия по умолчанию = %q, ожидалась random", cfg.Strategy)
	}
	if !cfg.DirectOnFail {
		t.Fatal("direct_on_failure по умолчанию должен быть включён")
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("значения по умолчанию не проходят валидацию: %v", err)
	}
}

func TestParseMinimal(t *testing.T) {
	cfg, err := Parse(strings.NewReader(`{"schema_version":1,"intent":"enabled"}`))
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if cfg.Intent != IntentEnabled || cfg.Strategy != StrategyRandom || !cfg.DirectOnFail {
		t.Fatalf("не применены значения по умолчанию: %+v", cfg)
	}
}

func TestParseErrors(t *testing.T) {
	cases := []struct {
		name string
		in   string
		code vmerrors.Code
	}{
		{"пустой файл", "   ", vmerrors.CodeValidation},
		{"нет schema_version", `{"intent":"enabled"}`, vmerrors.CodeValidation},
		{"версия схемы", `{"schema_version":2}`, vmerrors.CodeValidation},
		{"неизвестное поле", `{"schema_version":1,"nftables":true}`, vmerrors.CodeValidation},
		{"недопустимый intent", `{"schema_version":1,"intent":"maybe"}`, vmerrors.CodeValidation},
		{"недопустимая стратегия", `{"schema_version":1,"strategy":"fastest"}`, vmerrors.CodeValidation},
		{"порты", `{"schema_version":1,"udp_allowlist":[{"from":0,"to":53}]}`, vmerrors.CodeValidation},
		{"обратный диапазон", `{"schema_version":1,"udp_allowlist":[{"from":53,"to":10}]}`, vmerrors.CodeValidation},
		{"битый json", `{"schema_version":`, vmerrors.CodeValidation},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(tc.in))
			if err == nil {
				t.Fatal("ожидалась ошибка, получено nil")
			}
			if got := vmerrors.CodeOf(err); got != tc.code {
				t.Fatalf("код ошибки = %q, ожидался %q (%v)", got, tc.code, err)
			}
		})
	}
}

func TestLoadMissingFile(t *testing.T) {
	cfg, found, err := Load(filepath.Join(t.TempDir(), "нет.json"))
	if err != nil {
		t.Fatalf("отсутствие файла не должно быть ошибкой: %v", err)
	}
	if found {
		t.Fatal("found должен быть false")
	}
	if cfg.Intent != IntentDisabled {
		t.Fatalf("ожидались значения по умолчанию, получено %+v", cfg)
	}
}

func TestLoadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"schema_version":1,"intent":"enabled","strategy":"leastPing","udp_allowlist":[{"from":53,"to":53}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, found, err := Load(path)
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if !found || cfg.Intent != IntentEnabled || cfg.Strategy != StrategyLeastPing {
		t.Fatalf("конфигурация прочитана неверно: found=%v cfg=%+v", found, cfg)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("валидация прочитанной конфигурации: %v", err)
	}
}

func TestLoadEmptyPath(t *testing.T) {
	_, _, err := Load("  ")
	if !vmerrors.Is(err, vmerrors.CodeInvalidArgument) {
		t.Fatalf("ожидался VM_ERR_INVALID_ARGUMENT, получено %v", err)
	}
}
