package config

import (
	"strings"
	"testing"

	"github.com/vvidomina/vpn-manager-openwrt/internal/storage"
	"github.com/vvidomina/vpn-manager-openwrt/internal/vmerrors"
)

// memReader — минимальная реализация Reader для тестов.
type memReader map[string][]byte

func (m memReader) Read(name string) ([]byte, error) {
	if data, ok := m[name]; ok {
		return data, nil
	}
	return nil, vmerrors.New(vmerrors.CodeNotFound, "mem.read", "нет файла: "+name)
}

// failReader имитирует ошибку хранилища, отличную от отсутствия файла.
type failReader struct{}

func (failReader) Read(name string) ([]byte, error) {
	return nil, vmerrors.New(vmerrors.CodeStorage, "mem.read", "ввод-вывод недоступен")
}

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
	}{
		{"пустой файл", "   "},
		{"нет schema_version", `{"intent":"enabled"}`},
		{"версия схемы", `{"schema_version":2}`},
		{"неизвестное поле", `{"schema_version":1,"nftables":true}`},
		{"недопустимый intent", `{"schema_version":1,"intent":"maybe"}`},
		{"недопустимая стратегия", `{"schema_version":1,"strategy":"fastest"}`},
		{"порты", `{"schema_version":1,"udp_allowlist":[{"from":0,"to":53}]}`},
		{"обратный диапазон", `{"schema_version":1,"udp_allowlist":[{"from":53,"to":10}]}`},
		{"битый json", `{"schema_version":`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(tc.in))
			if err == nil {
				t.Fatal("ожидалась ошибка, получено nil")
			}
			if got := vmerrors.CodeOf(err); got != vmerrors.CodeValidation {
				t.Fatalf("код ошибки = %q, ожидался VM_ERR_VALIDATION (%v)", got, err)
			}
		})
	}
}

func TestLoadMissingFile(t *testing.T) {
	cfg, found, err := Load(memReader{}, "config.json")
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

func TestLoadInvalidFileIsReported(t *testing.T) {
	r := memReader{"config.json": []byte(`{"schema_version":9}`)}
	cfg, found, err := Load(r, "config.json")
	if err == nil {
		t.Fatal("ожидалась ошибка валидации")
	}
	if !vmerrors.Is(err, vmerrors.CodeValidation) {
		t.Fatalf("ожидался VM_ERR_VALIDATION, получено %v", err)
	}
	if !found {
		t.Fatal("файл существует: found должен быть true")
	}
	if cfg.SchemaVersion != 0 || cfg.Intent != "" || cfg.Strategy != "" || len(cfg.UDPAllowlist) != 0 {
		t.Fatalf("при ошибке валидации конфигурация не должна подменяться: %+v", cfg)
	}
}

func TestLoadStorageFailureIsNotMissingFile(t *testing.T) {
	if _, _, err := Load(failReader{}, "config.json"); !vmerrors.Is(err, vmerrors.CodeStorage) {
		t.Fatalf("ожидался VM_ERR_STORAGE, получено %v", err)
	}
}

func TestLoadRejectsEmptyArguments(t *testing.T) {
	if _, _, err := Load(nil, "config.json"); !vmerrors.Is(err, vmerrors.CodeInvalidArgument) {
		t.Fatalf("ожидался VM_ERR_INVALID_ARGUMENT для nil-хранилища, получено %v", err)
	}
	if _, _, err := Load(memReader{}, "  "); !vmerrors.Is(err, vmerrors.CodeInvalidArgument) {
		t.Fatalf("ожидался VM_ERR_INVALID_ARGUMENT для пустого имени, получено %v", err)
	}
}

func TestLoadThroughStorage(t *testing.T) {
	store, err := storage.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	body := `{"schema_version":1,"intent":"enabled","strategy":"leastPing","udp_allowlist":[{"from":53,"to":53}]}`
	if err := store.WriteAtomic("config.json", []byte(body), 0o600); err != nil {
		t.Fatalf("запись через Storage: %v", err)
	}

	cfg, found, err := Load(store, "config.json")
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
