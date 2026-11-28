package xray

import (
	"context"
	"testing"

	"github.com/vvidomina/vpn-manager-openwrt/internal/vmerrors"
)

func TestStubNeverReportsRunning(t *testing.T) {
	var m Manager = NewStub()
	ctx := context.Background()

	if _, err := m.Build(ctx, nil); !vmerrors.Is(err, vmerrors.CodeUnsupported) {
		t.Fatalf("Build: ожидался VM_ERR_UNSUPPORTED, получено %v", err)
	}
	if err := m.Validate(ctx, []byte(`{}`)); !vmerrors.Is(err, vmerrors.CodeUnsupported) {
		t.Fatalf("Validate: ожидался VM_ERR_UNSUPPORTED, получено %v", err)
	}
	if _, err := m.Start(ctx, []byte(`{}`)); !vmerrors.Is(err, vmerrors.CodeUnsupported) {
		t.Fatalf("Start: ожидался VM_ERR_UNSUPPORTED, получено %v", err)
	}
	if err := m.Stop(ctx); !vmerrors.Is(err, vmerrors.CodeUnsupported) {
		t.Fatalf("Stop: ожидался VM_ERR_UNSUPPORTED, получено %v", err)
	}
	status, err := m.Status(ctx)
	if !vmerrors.Is(err, vmerrors.CodeUnsupported) {
		t.Fatalf("Status: ожидался VM_ERR_UNSUPPORTED, получено %v", err)
	}
	if status.Running {
		t.Fatal("заглушка не должна сообщать о запущенном Xray")
	}
}
