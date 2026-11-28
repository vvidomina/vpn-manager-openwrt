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

func newService() *Service {
	return New(&fakeCore{
		snap:  state.Derive(false, time.Date(2026, 11, 28, 12, 0, 0, 0, time.UTC)),
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
	if len(res.Signals) == 0 {
		t.Fatal("признаки состояния не переданы")
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
