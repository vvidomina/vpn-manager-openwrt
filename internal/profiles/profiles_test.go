package profiles

import (
	"context"
	"testing"

	"github.com/vvidomina/vpn-manager-openwrt/internal/vmerrors"
)

func TestStubImplementsContract(t *testing.T) {
	var p Profiles = NewStub()
	ctx := context.Background()
	profile := Profile{ID: "p1", Name: "default", RouteOrder: []string{"direct", "proxy"}}

	if err := p.Save(ctx, profile); !vmerrors.Is(err, vmerrors.CodeUnsupported) {
		t.Fatalf("Save: ожидался VM_ERR_UNSUPPORTED, получено %v", err)
	}
	if err := p.Assign(ctx, "srv-1", "p1"); !vmerrors.Is(err, vmerrors.CodeUnsupported) {
		t.Fatalf("Assign: ожидался VM_ERR_UNSUPPORTED, получено %v", err)
	}
	if set, err := p.Compile(ctx, profile); !vmerrors.Is(err, vmerrors.CodeUnsupported) {
		t.Fatalf("Compile: ожидался VM_ERR_UNSUPPORTED, получено %v", err)
	} else if len(set.Rules) != 0 || set.ProfileID != "" {
		t.Fatalf("Compile вернул правила: %+v", set)
	}
	if err := p.SetPending(ctx, "p1"); !vmerrors.Is(err, vmerrors.CodeUnsupported) {
		t.Fatalf("SetPending: ожидался VM_ERR_UNSUPPORTED, получено %v", err)
	}
	if list, err := p.List(ctx); !vmerrors.Is(err, vmerrors.CodeUnsupported) {
		t.Fatalf("List: ожидался VM_ERR_UNSUPPORTED, получено %v", err)
	} else if len(list) != 0 {
		t.Fatalf("List вернул профили: %+v", list)
	}
}

func TestStubLifecycleIsNoop(t *testing.T) {
	stub := NewStub()
	if stub.Name() != "profiles" {
		t.Fatalf("имя компонента = %q", stub.Name())
	}
	if err := stub.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := stub.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}
