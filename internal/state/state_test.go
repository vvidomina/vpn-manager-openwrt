package state

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vvidomina/vpn-manager-openwrt/internal/vmerrors"
)

func fixedClock() time.Time { return time.Date(2026, 11, 28, 12, 0, 0, 0, time.UTC) }

func TestDeriveDisabled(t *testing.T) {
	snap := Derive(false, fixedClock())
	if snap.State != StateDisabledByUser {
		t.Fatalf("состояние = %q, ожидалось DISABLED_BY_USER", snap.State)
	}
	if snap.Config != ConfigSaved {
		t.Fatalf("config = %q, ожидалось Saved", snap.Config)
	}
	if err := snap.Validate(); err != nil {
		t.Fatalf("снимок невалиден: %v", err)
	}
}

func TestDeriveEnabledStaysStarting(t *testing.T) {
	snap := Derive(true, fixedClock())
	if snap.State != StateStarting {
		t.Fatalf("состояние = %q, ожидалось STARTING: признаки S2–S6 не наблюдаются", snap.State)
	}
	for _, sig := range snap.Signals {
		if sig.Name == SignalIntent {
			continue
		}
		if sig.Known {
			t.Fatalf("признак %s не должен быть известен на этапе 1", sig.Name)
		}
	}
}

func TestInvalidConfigIsDegradedNotActive(t *testing.T) {
	snap := NewSnapshot(ConfigInvalid, "неподдерживаемая schema_version: 9", true, fixedClock())
	if snap.State != StateDegraded {
		t.Fatalf("состояние = %q, ожидалось DEGRADED", snap.State)
	}
	if snap.Config != ConfigInvalid || snap.ConfigError == "" {
		t.Fatalf("ось конфигурации не отражена: %+v", snap)
	}
	if !strings.Contains(snap.Reason, "CONFIG_STALE") {
		t.Fatalf("причина не указывает CONFIG_STALE: %q", snap.Reason)
	}
	if err := snap.Validate(); err != nil {
		t.Fatalf("снимок невалиден: %v", err)
	}
	var s3 Signal
	for _, sig := range snap.Signals {
		if sig.Name == SignalConfig {
			s3 = sig
		}
		if sig.Name == SignalIntent && sig.Known {
			t.Fatal("при невалидной конфигурации намерение неизвестно и не должно считаться известным")
		}
	}
	if !s3.Known || s3.Value != "invalid" {
		t.Fatalf("признак S3 = %+v, ожидался известный invalid", s3)
	}
}

func TestDegradedKeepsConfigAxis(t *testing.T) {
	snap := Degraded(ConfigSaved, "", "DEGRADED: паника компонента policy", false, fixedClock())
	if snap.State != StateDegraded || snap.Config != ConfigSaved {
		t.Fatalf("снимок деградации: %+v", snap)
	}
	if err := snap.Validate(); err != nil {
		t.Fatalf("снимок невалиден: %v", err)
	}
}

func TestStoreRejectsActiveWithoutSignals(t *testing.T) {
	store := NewStore()
	err := store.Set(Snapshot{State: StateActive})
	if !vmerrors.Is(err, vmerrors.CodeInvalidState) {
		t.Fatalf("ожидался VM_ERR_INVALID_STATE, получено %v", err)
	}
	if got := store.Snapshot().State; got != StateUnknown {
		t.Fatalf("состояние изменилось при ошибочной записи: %q", got)
	}
}

func TestStoreRejectsActiveWithInvalidConfig(t *testing.T) {
	store := NewStore()
	now := fixedClock()
	snap := Snapshot{State: StateActive, Config: ConfigInvalid, Signals: []Signal{
		{Name: SignalProcess, Value: "running", Known: true, At: now},
		{Name: SignalConfig, Value: "invalid", Known: true, At: now},
		{Name: SignalPolicy, Value: "installed", Known: true, At: now},
		{Name: SignalTraffic, Value: "ok", Known: true, At: now},
	}}
	if err := store.Set(snap); !vmerrors.Is(err, vmerrors.CodeInvalidState) {
		t.Fatalf("ожидался VM_ERR_INVALID_STATE, получено %v", err)
	}
}

func TestStoreAcceptsActiveWithSignals(t *testing.T) {
	store := NewStore()
	now := fixedClock()
	snap := Snapshot{State: StateActive, Config: ConfigApplied, Signals: []Signal{
		{Name: SignalProcess, Value: "running", Known: true, At: now},
		{Name: SignalConfig, Value: "capture", Known: true, At: now},
		{Name: SignalPolicy, Value: "installed", Known: true, At: now},
		{Name: SignalTraffic, Value: "ok", Known: true, At: now},
	}}
	if err := store.Set(snap); err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if got := store.Snapshot(); got.State != StateActive || got.ObservedAt.IsZero() {
		t.Fatalf("снимок не сохранён: %+v", got)
	}
}

func TestStoreSnapshotIsCopy(t *testing.T) {
	store := NewStore()
	if err := store.Set(Derive(false, fixedClock())); err != nil {
		t.Fatal(err)
	}
	snap := store.Snapshot()
	snap.Signals[0].Value = "изменено"
	if got := store.Snapshot().Signals[0].Value; got == "изменено" {
		t.Fatal("Snapshot вернул ссылку на внутренние данные")
	}
}

func TestStoreConcurrentAccess(t *testing.T) {
	store := NewStoreWithClock(fixedClock)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(enabled bool) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = store.Set(Derive(enabled, fixedClock()))
				_ = store.Snapshot()
			}
		}(i%2 == 0)
	}
	wg.Wait()
}
