package diagnostics

import (
	"strings"
	"testing"
	"time"

	"github.com/vvidomina/vpn-manager-openwrt/internal/config"
	"github.com/vvidomina/vpn-manager-openwrt/internal/state"
)

func TestBuildReport(t *testing.T) {
	cfg := config.Default()
	cfg.Intent = config.IntentEnabled
	snap := state.Derive(true, time.Date(2026, 11, 28, 12, 0, 0, 0, time.UTC))

	rep := Build("0.1.0-stage1", cfg, snap, "проверка")
	if rep.Version != "0.1.0-stage1" || rep.SchemaVersion != SchemaVersion {
		t.Fatalf("шапка отчёта: %+v", rep)
	}
	if rep.State.State != state.StateStarting {
		t.Fatalf("состояние = %q", rep.State.State)
	}
	if len(rep.Warnings) < 3 {
		t.Fatalf("ожидались предупреждения, получено %v", rep.Warnings)
	}
	joined := strings.Join(rep.Warnings, " ")
	if !strings.Contains(joined, "не наблюдаются") || !strings.Contains(joined, "не реализовано") {
		t.Fatalf("предупреждения не описывают границы этапа: %s", joined)
	}
}

func TestReportReportsInvalidConfig(t *testing.T) {
	snap := state.NewSnapshot(state.ConfigInvalid, "неподдерживаемая schema_version: 9", true,
		time.Date(2026, 11, 28, 12, 0, 0, 0, time.UTC))
	rep := Build("v", config.Default(), snap)
	joined := strings.Join(rep.Warnings, " ")
	if !strings.Contains(joined, "config=Invalid") {
		t.Fatalf("отчёт не сообщает о невалидной конфигурации: %v", rep.Warnings)
	}
	if !strings.Contains(joined, "schema_version") {
		t.Fatalf("отчёт не содержит причину: %v", rep.Warnings)
	}
	if rep.State.State != state.StateDegraded {
		t.Fatalf("состояние отчёта = %q, ожидалось DEGRADED", rep.State.State)
	}
}

func TestReportDoesNotMutateSnapshot(t *testing.T) {
	snap := state.Derive(false, time.Date(2026, 11, 28, 12, 0, 0, 0, time.UTC))
	rep := Build("v", config.Default(), snap)
	rep.State.Signals[0].Value = "изменено"
	if snap.Signals[0].Value == "изменено" {
		t.Fatal("Report делит срез признаков с исходным снимком")
	}
}
