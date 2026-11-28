package core

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/vvidomina/vpn-manager-openwrt/internal/config"
	"github.com/vvidomina/vpn-manager-openwrt/internal/logging"
	"github.com/vvidomina/vpn-manager-openwrt/internal/state"
	"github.com/vvidomina/vpn-manager-openwrt/internal/vmerrors"
)

type fakeComp struct {
	name     string
	order    *[]string
	startErr error
	stopErr  error
}

func (f *fakeComp) Name() string { return f.name }

func (f *fakeComp) Start(ctx context.Context) error {
	if f.startErr != nil {
		return f.startErr
	}
	*f.order = append(*f.order, "start:"+f.name)
	return nil
}

func (f *fakeComp) Stop(ctx context.Context) error {
	*f.order = append(*f.order, "stop:"+f.name)
	return f.stopErr
}

// panicComp паникует в синхронном lifecycle-вызове.
type panicComp struct {
	name        string
	order       *[]string
	panicStart  bool
	panicStop   bool
	stopReached bool
}

func (p *panicComp) Name() string { return p.name }

func (p *panicComp) Start(ctx context.Context) error {
	if p.panicStart {
		panic("сбой старта")
	}
	*p.order = append(*p.order, "start:"+p.name)
	return nil
}

func (p *panicComp) Stop(ctx context.Context) error {
	p.stopReached = true
	if p.panicStop {
		panic("сбой остановки")
	}
	*p.order = append(*p.order, "stop:"+p.name)
	return nil
}

func newTestCore(t *testing.T, intent config.Intent) (*Core, *state.Store, *[]string) {
	t.Helper()
	cfg := config.Default()
	cfg.Intent = intent
	st := state.NewStore()
	log := logging.New(io.Discard, logging.LevelError)
	order := &[]string{}
	c := New(cfg, log, st, WithStopTimeout(200*time.Millisecond))
	for _, name := range []string{"a", "b"} {
		if err := c.Register(&fakeComp{name: name, order: order}); err != nil {
			t.Fatalf("регистрация %s: %v", name, err)
		}
	}
	return c, st, order
}

func TestLifecycleDisabled(t *testing.T) {
	c, st, order := newTestCore(t, config.IntentDisabled)
	if c.Ready() {
		t.Fatal("ядро не должно быть готово до старта")
	}
	if err := c.Start(context.Background()); err != nil {
		t.Fatalf("старт: %v", err)
	}
	if !c.Ready() {
		t.Fatal("ядро должно быть готово после старта")
	}
	if got := st.Snapshot().State; got != state.StateDisabledByUser {
		t.Fatalf("состояние = %q, ожидалось DISABLED_BY_USER", got)
	}
	if err := c.Stop(context.Background()); err != nil {
		t.Fatalf("остановка: %v", err)
	}
	if c.Ready() {
		t.Fatal("после остановки ядро не готово")
	}
	want := []string{"start:a", "start:b", "stop:b", "stop:a"}
	if len(*order) != len(want) {
		t.Fatalf("порядок вызовов = %v, ожидался %v", *order, want)
	}
	for i := range want {
		if (*order)[i] != want[i] {
			t.Fatalf("порядок вызовов = %v, ожидался %v", *order, want)
		}
	}
	if got := st.Snapshot().State; got != state.StateUnknown {
		t.Fatalf("состояние после остановки = %q, ожидалось UNKNOWN", got)
	}
}

func TestLifecycleEnabled(t *testing.T) {
	c, st, _ := newTestCore(t, config.IntentEnabled)
	if err := c.Start(context.Background()); err != nil {
		t.Fatalf("старт: %v", err)
	}
	defer func() { _ = c.Stop(context.Background()) }()
	if got := st.Snapshot().State; got != state.StateStarting {
		t.Fatalf("состояние = %q, ожидалось STARTING: ACTIVE недостижим на этапе 1", got)
	}
}

func TestStartTwiceFails(t *testing.T) {
	c, _, _ := newTestCore(t, config.IntentDisabled)
	if err := c.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Stop(context.Background()) }()
	if err := c.Start(context.Background()); !vmerrors.Is(err, vmerrors.CodeInvalidState) {
		t.Fatalf("ожидался VM_ERR_INVALID_STATE, получено %v", err)
	}
}

func TestStartFailureRollsBack(t *testing.T) {
	st := state.NewStore()
	log := logging.New(io.Discard, logging.LevelError)
	order := &[]string{}
	c := New(config.Default(), log, st)
	if err := c.Register(&fakeComp{name: "ok", order: order}); err != nil {
		t.Fatal(err)
	}
	failing := &fakeComp{name: "bad", order: order, startErr: vmerrors.New(vmerrors.CodeStorage, "test", "нет доступа")}
	if err := c.Register(failing); err != nil {
		t.Fatal(err)
	}
	err := c.Start(context.Background())
	if !vmerrors.Is(err, vmerrors.CodeStorage) {
		t.Fatalf("код ошибки не сохранён: %v", err)
	}
	if c.Ready() {
		t.Fatal("ядро не должно быть готово после сбоя старта")
	}
	if len(*order) != 2 || (*order)[0] != "start:ok" || (*order)[1] != "stop:ok" {
		t.Fatalf("откат выполнен неверно: %v", *order)
	}
	// Сбой компонента оставляет ядро в DEGRADED, а не в неопределённом состоянии.
	snap := st.Snapshot()
	if snap.State != state.StateDegraded {
		t.Fatalf("состояние при сбое старта = %q, ожидалось DEGRADED", snap.State)
	}
	if !strings.Contains(snap.Reason, "bad") {
		t.Fatalf("причина не содержит компонент: %q", snap.Reason)
	}
	if err := c.Stop(context.Background()); err != nil {
		t.Fatalf("остановка после сбоя: %v", err)
	}
}

func TestInvalidConfigKeepsCoreRunning(t *testing.T) {
	st := state.NewStore()
	log := logging.New(io.Discard, logging.LevelError)
	order := &[]string{}
	cfg := config.Default()
	cfg.Intent = config.IntentEnabled
	c := New(cfg, log, st, WithConfigState(state.ConfigInvalid, "неподдерживаемая schema_version: 9"))
	if err := c.Register(&fakeComp{name: "a", order: order}); err != nil {
		t.Fatal(err)
	}

	err := c.Start(context.Background())
	if !vmerrors.Is(err, vmerrors.CodeValidation) {
		t.Fatalf("ожидался VM_ERR_VALIDATION, получено %v", err)
	}
	if c.Ready() {
		t.Fatal("при невалидной конфигурации готовность не публикуется")
	}
	snap := st.Snapshot()
	if snap.State != state.StateDegraded || snap.Config != state.ConfigInvalid {
		t.Fatalf("снимок = %+v, ожидались DEGRADED/config=Invalid", snap)
	}
	if snap.ConfigError == "" {
		t.Fatal("причина невалидности не попала в снимок")
	}
	if len(*order) != 0 {
		t.Fatalf("компоненты не должны запускаться при невалидной конфигурации: %v", *order)
	}
	if _, err := c.Submit("enable"); !vmerrors.Is(err, vmerrors.CodeInvalidState) {
		t.Fatalf("операции должны отклоняться: %v", err)
	}
	// Демон остаётся доступным для остановки и диагностики.
	if err := c.Stop(context.Background()); err != nil {
		t.Fatalf("остановка: %v", err)
	}
}

func TestPanicOnStartDegradesAndRollsBack(t *testing.T) {
	st := state.NewStore()
	log := logging.New(io.Discard, logging.LevelError)
	order := &[]string{}
	c := New(config.Default(), log, st)
	if err := c.Register(&fakeComp{name: "ok", order: order}); err != nil {
		t.Fatal(err)
	}
	bad := &panicComp{name: "bad", order: order, panicStart: true}
	if err := c.Register(bad); err != nil {
		t.Fatal(err)
	}

	err := c.Start(context.Background())
	if !vmerrors.Is(err, vmerrors.CodeInternal) {
		t.Fatalf("паника должна стать VM_ERR_INTERNAL, получено %v", err)
	}
	if !strings.Contains(err.Error(), "паника") {
		t.Fatalf("сообщение не описывает панику: %v", err)
	}
	if c.Ready() {
		t.Fatal("после паники готовность не публикуется")
	}
	snap := st.Snapshot()
	if snap.State != state.StateDegraded {
		t.Fatalf("состояние = %q, ожидалось DEGRADED", snap.State)
	}
	if !strings.Contains(snap.Reason, "bad") {
		t.Fatalf("причина не содержит компонент: %q", snap.Reason)
	}
	if len(*order) != 2 || (*order)[1] != "stop:ok" {
		t.Fatalf("запущенные компоненты не остановлены: %v", *order)
	}
	if err := c.Stop(context.Background()); err != nil {
		t.Fatalf("остановка после паники: %v", err)
	}
}

func TestPanicOnStopDegradesButStopsOthers(t *testing.T) {
	st := state.NewStore()
	log := logging.New(io.Discard, logging.LevelError)
	order := &[]string{}
	c := New(config.Default(), log, st)
	bad := &panicComp{name: "bad", order: order, panicStop: true}
	if err := c.Register(bad); err != nil {
		t.Fatal(err)
	}
	if err := c.Register(&fakeComp{name: "ok", order: order}); err != nil {
		t.Fatal(err)
	}
	if err := c.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := c.Stop(context.Background()); !vmerrors.Is(err, vmerrors.CodeInternal) {
		t.Fatalf("ожидался VM_ERR_INTERNAL, получено %v", err)
	}
	if !bad.stopReached {
		t.Fatal("Stop компонента не вызывался")
	}
	// Остановка остальных компонентов не прервана: порядок обратный (ok → bad),
	// поэтому ok остановлен до паники, а его запись присутствует.
	if len(*order) < 3 || (*order)[2] != "stop:ok" {
		t.Fatalf("остановка остальных компонентов прервана: %v", *order)
	}
	if got := st.Snapshot().State; got != state.StateDegraded {
		t.Fatalf("состояние = %q, ожидалось DEGRADED после паники при остановке", got)
	}
}

func TestRegisterAfterStart(t *testing.T) {
	c, _, order := newTestCore(t, config.IntentDisabled)
	if err := c.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Stop(context.Background()) }()
	if err := c.Register(&fakeComp{name: "late", order: order}); !vmerrors.Is(err, vmerrors.CodeInvalidState) {
		t.Fatalf("ожидался VM_ERR_INVALID_STATE, получено %v", err)
	}
}

func TestRegisterDuplicate(t *testing.T) {
	c, _, order := newTestCore(t, config.IntentDisabled)
	if err := c.Register(&fakeComp{name: "a", order: order}); !vmerrors.Is(err, vmerrors.CodeConflict) {
		t.Fatalf("ожидался VM_ERR_CONFLICT, получено %v", err)
	}
}

func TestSubmitAndComplete(t *testing.T) {
	c, _, _ := newTestCore(t, config.IntentDisabled)
	if _, err := c.Submit("enable"); !vmerrors.Is(err, vmerrors.CodeInvalidState) {
		t.Fatalf("до старта ожидался VM_ERR_INVALID_STATE, получено %v", err)
	}
	if err := c.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Stop(context.Background()) }()

	id, err := c.Submit("enable")
	if err != nil {
		t.Fatalf("операция не принята: %v", err)
	}
	if _, err := c.Submit("disable"); !vmerrors.Is(err, vmerrors.CodeBusy) {
		t.Fatalf("ожидался VM_ERR_BUSY, получено %v", err)
	}
	if err := c.Complete("op-999"); !vmerrors.Is(err, vmerrors.CodeNotFound) {
		t.Fatalf("ожидался VM_ERR_NOT_FOUND, получено %v", err)
	}
	if err := c.Complete(id); err != nil {
		t.Fatalf("завершение операции: %v", err)
	}
	if c.ActiveOperation() != "" {
		t.Fatal("активная операция не снята")
	}
}

func TestStopIsIdempotent(t *testing.T) {
	c, _, _ := newTestCore(t, config.IntentDisabled)
	if err := c.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := c.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := c.Stop(context.Background()); err != nil {
		t.Fatalf("повторная остановка должна быть безопасной: %v", err)
	}
}

func TestConfigStatus(t *testing.T) {
	c, _, _ := newTestCore(t, config.IntentDisabled)
	cs, reason := c.ConfigStatus()
	if cs != state.ConfigSaved || reason != "" {
		t.Fatalf("состояние конфигурации = %q/%q", cs, reason)
	}
}
