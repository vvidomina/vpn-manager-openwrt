package policy

import (
	"context"
	"testing"

	"github.com/vvidomina/vpn-manager-openwrt/internal/vmerrors"
)

func TestStubDoesNotApplyRules(t *testing.T) {
	var m Manager = NewStub()
	ctx := context.Background()

	if got, err := m.Apply(ctx, Intent{Mode: "capture"}); !vmerrors.Is(err, vmerrors.CodeUnsupported) {
		t.Fatalf("Apply: ожидался VM_ERR_UNSUPPORTED, получено %v", err)
	} else if got.Installed || got.Rules != 0 {
		t.Fatalf("заглушка сообщила об установленных правилах: %+v", got)
	}
	if got, err := m.Verify(ctx, Intent{Mode: "capture"}); !vmerrors.Is(err, vmerrors.CodeUnsupported) {
		t.Fatalf("Verify: ожидался VM_ERR_UNSUPPORTED, получено %v", err)
	} else if got.Installed {
		t.Fatal("заглушка не должна подтверждать политику")
	}
	if err := m.Remove(ctx); !vmerrors.Is(err, vmerrors.CodeUnsupported) {
		t.Fatalf("Remove: ожидался VM_ERR_UNSUPPORTED, получено %v", err)
	}
}
