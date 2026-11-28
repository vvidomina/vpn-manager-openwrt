package devices

import (
	"context"
	"testing"

	"github.com/vvidomina/vpn-manager-openwrt/internal/vmerrors"
)

func TestStubReturnsUnsupported(t *testing.T) {
	stub := NewStub()
	if _, err := stub.List(context.Background()); !vmerrors.Is(err, vmerrors.CodeUnsupported) {
		t.Fatalf("List: ожидался VM_ERR_UNSUPPORTED, получено %v", err)
	}
	if err := stub.SetExcluded(context.Background(), "00:00:00:00:00:00", true); !vmerrors.Is(err, vmerrors.CodeUnsupported) {
		t.Fatalf("SetExcluded: ожидался VM_ERR_UNSUPPORTED, получено %v", err)
	}
	if err := stub.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := stub.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}
