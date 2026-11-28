package core

import (
	"context"
	"io"
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
	if got := st.Snapshot().State; got != state.StateUnknown {
		t.Fatalf("состояние при сбое старта = %q, ожидалось UNKNOWN", got)
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
