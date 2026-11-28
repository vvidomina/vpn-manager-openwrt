package devices

import (
	"context"
	"testing"

	"github.com/vvidomina/vpn-manager-openwrt/internal/vmerrors"
)

func TestModeValues(t *testing.T) {
	if ModeDefault != "default" || ModeAlwaysDirect != "always_direct" {
		t.Fatalf("режимы не соответствуют №11 §7: %q, %q", ModeDefault, ModeAlwaysDirect)
	}
}

func TestStubImplementsContract(t *testing.T) {
	var d Devices = NewStub()
	ctx := context.Background()

	if list, err := d.List(ctx, Filter{OnlineOnly: true}); !vmerrors.Is(err, vmerrors.CodeUnsupported) {
		t.Fatalf("List: ожидался VM_ERR_UNSUPPORTED, получено %v", err)
	} else if len(list) != 0 {
		t.Fatalf("List вернул устройства: %+v", list)
	}
	if err := d.SetMode(ctx, "00:11:22:33:44:55", ModeAlwaysDirect); !vmerrors.Is(err, vmerrors.CodeUnsupported) {
		t.Fatalf("SetMode: ожидался VM_ERR_UNSUPPORTED, получено %v", err)
	}
	if err := d.Merge(ctx, []string{"00:11:22:33:44:55", "00:11:22:33:44:66"}); !vmerrors.Is(err, vmerrors.CodeUnsupported) {
		t.Fatalf("Merge: ожидался VM_ERR_UNSUPPORTED, получено %v", err)
	}
	if res, err := d.Resolve(ctx); !vmerrors.Is(err, vmerrors.CodeUnsupported) {
		t.Fatalf("Resolve: ожидался VM_ERR_UNSUPPORTED, получено %v", err)
	} else if len(res) != 0 {
		t.Fatalf("Resolve вернул назначения: %+v", res)
	}
}

func TestStubLifecycleIsNoop(t *testing.T) {
	stub := NewStub()
	if stub.Name() != "devices" {
		t.Fatalf("имя компонента = %q", stub.Name())
	}
	if err := stub.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := stub.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}
