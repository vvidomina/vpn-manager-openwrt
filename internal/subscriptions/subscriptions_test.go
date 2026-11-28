package subscriptions

import (
	"context"
	"testing"

	"github.com/vvidomina/vpn-manager-openwrt/internal/vmerrors"
)

func TestStubImplementsContract(t *testing.T) {
	var s Subscriptions = NewStub()
	ctx := context.Background()

	sub, err := s.Add(ctx, Source{URL: "https://example.invalid/sub"})
	if !vmerrors.Is(err, vmerrors.CodeUnsupported) {
		t.Fatalf("Add: ожидался VM_ERR_UNSUPPORTED, получено %v", err)
	}
	if sub.ID != "" || sub.Servers != 0 {
		t.Fatalf("Add вернул подписку: %+v", sub)
	}

	rep, err := s.Update(ctx, "sub-1")
	if !vmerrors.Is(err, vmerrors.CodeUnsupported) {
		t.Fatalf("Update: ожидался VM_ERR_UNSUPPORTED, получено %v", err)
	}
	if rep.Accepted != 0 || rep.Rejected != 0 {
		t.Fatalf("Update вернул отчёт импорта: %+v", rep)
	}

	if err := s.Remove(ctx, "sub-1"); !vmerrors.Is(err, vmerrors.CodeUnsupported) {
		t.Fatalf("Remove: ожидался VM_ERR_UNSUPPORTED, получено %v", err)
	}
	if err := s.Reorder(ctx, []string{"sub-2", "sub-1"}); !vmerrors.Is(err, vmerrors.CodeUnsupported) {
		t.Fatalf("Reorder: ожидался VM_ERR_UNSUPPORTED, получено %v", err)
	}
	if list, err := s.List(ctx); !vmerrors.Is(err, vmerrors.CodeUnsupported) {
		t.Fatalf("List: ожидался VM_ERR_UNSUPPORTED, получено %v", err)
	} else if len(list) != 0 {
		t.Fatalf("List вернул подписки: %+v", list)
	}
}

func TestStubLifecycleIsNoop(t *testing.T) {
	stub := NewStub()
	if stub.Name() != "subscriptions" {
		t.Fatalf("имя компонента = %q", stub.Name())
	}
	if err := stub.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := stub.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}
