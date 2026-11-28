package profiles

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
	if err := stub.Activate(context.Background(), "p1"); !vmerrors.Is(err, vmerrors.CodeUnsupported) {
		t.Fatalf("Activate: ожидался VM_ERR_UNSUPPORTED, получено %v", err)
	}
}
