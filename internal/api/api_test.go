package api

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/vvidomina/vpn-manager-openwrt/internal/state"
	"github.com/vvidomina/vpn-manager-openwrt/internal/vmerrors"
)

type fakeCore struct {
	snap   state.Snapshot
	ready  bool
	active string
}

func (f *fakeCore) Status() state.Snapshot           { return f.snap }
func (f *fakeCore) Ready() bool                      { return f.ready }
func (f *fakeCore) ActiveOperation() string          { return f.active }
func (f *fakeCore) Submit(op string) (string, error) { return "op-1", nil }
func (f *fakeCore) Complete(id string) error         { return nil }

func fixedTime() time.Time { return time.Date(2026, 11, 28, 12, 0, 0, 0, time.UTC) }

func newService() *Service {
	return New(&fakeCore{
		snap:  state.Derive(false, fixedTime()),
		ready: true,
	}, "0.1.0-stage1")
}

func TestStatus(t *testing.T) {
	resp := newService().Call(context.Background(), Request{Method: MethodStatus})
	if !resp.OK || resp.Error != nil {
		t.Fatalf("ожидался успешный ответ, получено %+v", resp)
	}
	res, ok := resp.Result.(StatusResult)
	if !ok {
		t.Fatalf("неожиданный тип результата: %T", resp.Result)
	}
	if res.State != string(state.StateDisabledByUser) || !res.Ready {
		t.Fatalf("неверное состояние: %+v", res)
	}
	if res.Config != string(state.ConfigSaved) {
		t.Fatalf("ось конфигурации не передана: %q", res.Config)
	}
	if len(res.Signals) == 0 {
		t.Fatal("признаки состояния не переданы")
	}
}

func TestStatusWithInvalidConfig(t *testing.T) {
	snap := state.NewSnapshot(state.ConfigInvalid, "неподдерживаемая schema_version: 9", true, fixedTime())
	svc := New(&fakeCore{snap: snap, ready: false}, "v")
	resp := svc.Call(context.Background(), Request{Method: MethodStatus})
	if !resp.OK {
		t.Fatalf("статус должен отдаваться и при невалидной конфигурации: %+v", resp)
	}
	res := resp.Result.(StatusResult)
	if res.Config != string(state.ConfigInvalid) || res.ConfigError == "" {
		t.Fatalf("причина невалидности не передана: %+v", res)
	}
	if res.Ready {
		t.Fatal("готовность не должна публиковаться при невалидной конфигурации")
	}
}

func TestHealth(t *testing.T) {
	res, err := newService().Health(context.Background())
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if !res.OK || res.State != string(state.StateDisabledByUser) || !res.Ready {
		t.Fatalf("health-ping: %+v", res)
	}
}

func TestHealthNotReady(t *testing.T) {
	svc := New(&fakeCore{snap: state.Derive(false, fixedTime()), ready: false}, "v")
	res, err := svc.Health(context.Background())
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if res.OK {
		t.Fatal("health-ping не должен сообщать об успехе при неготовом ядре")
	}
}

func TestHealthInvalidConfigAndDegraded(t *testing.T) {
	invalid := state.NewSnapshot(state.ConfigInvalid, "нет schema_version", true, fixedTime())
	svc := New(&fakeCore{snap: invalid, ready: true}, "v")
	res, _ := svc.Health(context.Background())
	if res.OK {
		t.Fatal("health-ping не должен сообщать об успехе при config=Invalid")
	}
	if res.Config != string(state.ConfigInvalid) {
		t.Fatalf("config не передан: %+v", res)
	}

	degraded := state.Degraded(state.ConfigSaved, "", "паника компонента", false, fixedTime())
	svc = New(&fakeCore{snap: degraded, ready: true}, "v")
	res, _ = svc.Health(context.Background())
	if res.OK {
		t.Fatal("health-ping не должен сообщать об успехе в DEGRADED")
	}
}

func TestHealthCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := newService().Health(ctx); !vmerrors.Is(err, vmerrors.CodeTimeout) {
		t.Fatalf("ожидался VM_ERR_TIMEOUT, получено %v", err)
	}
}

func TestVersion(t *testing.T) {
	resp := newService().Call(context.Background(), Request{Method: MethodVersion})
	if !resp.OK {
		t.Fatalf("ожидался успешный ответ: %+v", resp)
	}
	if v, ok := resp.Result.(VersionResult); !ok || v.Version != "0.1.0-stage1" {
		t.Fatalf("неверная версия: %+v", resp.Result)
	}
}

func TestUnknownMethod(t *testing.T) {
	resp := newService().Call(context.Background(), Request{Method: "vpnmanager.nftables"})
	if resp.OK || resp.Error == nil {
		t.Fatalf("ожидалась ошибка: %+v", resp)
	}
	if resp.Error.Code != string(vmerrors.CodeUnsupported) {
		t.Fatalf("код = %q, ожидался VM_ERR_UNSUPPORTED", resp.Error.Code)
	}
}

func TestOperation(t *testing.T) {
	resp := newService().Call(context.Background(), Request{Method: MethodOperation})
	if !resp.OK {
		t.Fatalf("ожидался успешный ответ: %+v", resp)
	}
	if op, ok := resp.Result.(OperationResult); !ok || op.Active != "" {
		t.Fatalf("неверный результат: %+v", resp.Result)
	}

	params, _ := json.Marshal(OperationParams{ID: "op-1"})
	resp = newService().Call(context.Background(), Request{Method: MethodOperation, Params: params})
	if resp.OK || resp.Error == nil || resp.Error.Code != string(vmerrors.CodeNotFound) {
		t.Fatalf("ожидался VM_ERR_NOT_FOUND: %+v", resp)
	}
}

func TestOperationInvalidParams(t *testing.T) {
	resp := newService().Call(context.Background(), Request{Method: MethodOperation, Params: json.RawMessage(`"строка"`)})
	if resp.Error == nil || resp.Error.Code != string(vmerrors.CodeInvalidArgument) {
		t.Fatalf("ожидался VM_ERR_INVALID_ARGUMENT: %+v", resp)
	}
}

func TestCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	resp := newService().Call(ctx, Request{Method: MethodStatus})
	if resp.Error == nil || resp.Error.Code != string(vmerrors.CodeTimeout) {
		t.Fatalf("ожидался VM_ERR_TIMEOUT: %+v", resp)
	}
}
