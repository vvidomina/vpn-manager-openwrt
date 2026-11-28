// Package config — базовая модель конфигурации VPN Manager (U9).
// Допустимые значения полей определены документами №13 и №44 §3.2.
// Пакет только читает и валидирует конфигурацию: файлы проекта не изменяются.
package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/vvidomina/vpn-manager-openwrt/internal/vmerrors"
)

// SchemaVersion — поддерживаемая версия схемы.
const SchemaVersion = 1

// Intent — намерение пользователя.
type Intent string

const (
	IntentEnabled  Intent = "enabled"
	IntentDisabled Intent = "disabled"
)

// Strategy — штатная стратегия выбора сервера Xray (№18 §7.1).
type Strategy string

const (
	StrategyRandom     Strategy = "random"
	StrategyRoundRobin Strategy = "roundRobin"
	StrategyLeastPing  Strategy = "leastPing"
	StrategyLeastLoad  Strategy = "leastLoad"
)

// PortRange — диапазон UDP-портов allowlist (№18 §5).
type PortRange struct {
	From int `json:"from"`
	To   int `json:"to"`
}

// Config — конфигурация приложения. Неизвестные поля запрещены.
type Config struct {
	SchemaVersion int         `json:"schema_version"`
	Intent        Intent      `json:"intent"`
	Strategy      Strategy    `json:"strategy"`
	DirectOnFail  bool        `json:"direct_on_failure"`
	UDPAllowlist  []PortRange `json:"udp_allowlist,omitempty"`
}

// Default возвращает значения по умолчанию, определённые U9:
// VPN выключен пользователем, стратегия random, direct_on_failure включён.
func Default() Config {
	return Config{
		SchemaVersion: SchemaVersion,
		Intent:        IntentDisabled,
		Strategy:      StrategyRandom,
		DirectOnFail:  true,
	}
}

// Validate проверяет обязательные поля и допустимые значения.
func (c Config) Validate() error {
	if c.SchemaVersion != SchemaVersion {
		return vmerrors.New(vmerrors.CodeValidation, "config.validate",
			fmt.Sprintf("неподдерживаемая schema_version: %d (ожидается %d)", c.SchemaVersion, SchemaVersion))
	}
	switch c.Intent {
	case IntentEnabled, IntentDisabled:
	default:
		return vmerrors.New(vmerrors.CodeValidation, "config.validate",
			fmt.Sprintf("недопустимый intent: %q", c.Intent))
	}
	switch c.Strategy {
	case StrategyRandom, StrategyRoundRobin, StrategyLeastPing, StrategyLeastLoad:
	default:
		return vmerrors.New(vmerrors.CodeValidation, "config.validate",
			fmt.Sprintf("недопустимая стратегия: %q", c.Strategy))
	}
	for i, r := range c.UDPAllowlist {
		if r.From < 1 || r.To > 65535 || r.From > r.To {
			return vmerrors.New(vmerrors.CodeValidation, "config.validate",
				fmt.Sprintf("недопустимый диапазон UDP-портов в записи %d: %d-%d", i, r.From, r.To))
		}
	}
	return nil
}

// Parse разбирает конфигурацию из JSON: неизвестные поля — ошибка,
// отсутствующие — заменяются значениями по умолчанию.
func Parse(r io.Reader) (Config, error) {
	raw, err := io.ReadAll(io.LimitReader(r, 1<<20))
	if err != nil {
		return Config{}, vmerrors.Wrap(vmerrors.CodeStorage, "config.parse", err)
	}
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return Config{}, vmerrors.New(vmerrors.CodeValidation, "config.parse", "пустой файл конфигурации")
	}

	cfg := Default()
	// schema_version обязателен: нулевое значение не должно маскироваться
	// значением по умолчанию.
	var probe struct {
		SchemaVersion *int `json:"schema_version"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return Config{}, vmerrors.Wrap(vmerrors.CodeValidation, "config.parse", err)
	}
	if probe.SchemaVersion == nil {
		return Config{}, vmerrors.New(vmerrors.CodeValidation, "config.parse", "отсутствует обязательное поле schema_version")
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return Config{}, vmerrors.Wrap(vmerrors.CodeValidation, "config.parse", err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Load читает конфигурацию из файла. Отсутствие файла — штатный первый
// запуск: возвращаются значения по умолчанию и found=false.
func Load(path string) (cfg Config, found bool, err error) {
	if strings.TrimSpace(path) == "" {
		return Config{}, false, vmerrors.New(vmerrors.CodeInvalidArgument, "config.load", "пустой путь к конфигурации")
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), false, nil
		}
		return Config{}, false, vmerrors.Wrap(vmerrors.CodeStorage, "config.load", err)
	}
	defer f.Close()

	cfg, err = Parse(f)
	if err != nil {
		return Config{}, true, err
	}
	return cfg, true, nil
}
