package subscriptions

import (
	"context"
	"testing"

	"github.com/vvidomina/vpn-manager-openwrt/internal/vmerrors"
)

func TestStubDoesNotUseNetwork(t *testing.T) {
	stub := NewStub()
	if _, err := stub.List(context.Background()); !vmerrors.Is(err, vmerrors.CodeUnsupported) {
		t.Fatalf("List: ожидался VM_ERR_UNSUPPORTED, получено %v", err)
	}
	if _, err := stub.Refresh(context.Background(), "sub-1"); !vmerrors.Is(err, vmerrors.CodeUnsupported) {
		t.Fatalf("Refresh: ожидался VM_ERR_UNSUPPORTED, получено %v", err)
	}
}
