// Package diagnostics — сборка диагностического отчёта.
// Отчёт только читает состояние и конфигурацию; он не содержит секретов и не
// изменяет систему.
package diagnostics

import (
	"github.com/vvidomina/vpn-manager-openwrt/internal/config"
	"github.com/vvidomina/vpn-manager-openwrt/internal/state"
)

// SchemaVersion — версия схемы отчёта.
const SchemaVersion = 1

// Build собирает Report из текущего снимка и конфигурации.
func Build(version string, cfg config.Config, snap state.Snapshot, warnings ...string) state.Report {
	all := make([]string, 0, len(warnings)+2)
	all = append(all, warnings...)
	if cfg.Intent == config.IntentEnabled && snap.State != state.StateActive {
		all = append(all, "intent=enabled, но состояние "+string(snap.State)+": сетевые признаки в этапе 1 не наблюдаются")
	}
	all = append(all, "этап 1: управление Xray, nftables и сетью не реализовано")
	return state.Report{
		Version:       version,
		SchemaVersion: SchemaVersion,
		State:         snap.Clone(),
		Warnings:      all,
	}
}
