package policy

import (
	"context"
	"testing"

	"github.com/vvidomina/vpn-manager-openwrt/internal/vmerrors"
)

func TestStubImplementsContract(t *testing.T) {
	var p Policy = NewStub()
	ctx := context.Background()

	plan, err := p.Stage(ctx, Desired{Mode: ModeCapture, UDPPorts: []PortRange{{From: 53, To: 53}}})
	if !vmerrors.Is(err, vmerrors.CodeUnsupported) {
		t.Fatalf("Stage: ожидался VM_ERR_UNSUPPORTED, получено %v", err)
	}
	if plan.Rules != 0 || plan.Table != "" {
		t.Fatalf("Stage не должен подготавливать правила: %+v", plan)
	}

	applied, err := p.Apply(ctx, plan)
	if !vmerrors.Is(err, vmerrors.CodeUnsupported) {
		t.Fatalf("Apply: ожидался VM_ERR_UNSUPPORTED, получено %v", err)
	}
	if applied.Rules != 0 || !applied.AppliedAt.IsZero() {
		t.Fatalf("Apply не должен сообщать о применённой политике: %+v", applied)
	}

	if err := p.Rollback(ctx, applied); !vmerrors.Is(err, vmerrors.CodeUnsupported) {
		t.Fatalf("Rollback: ожидался VM_ERR_UNSUPPORTED, получено %v", err)
	}
	if err := p.Remove(ctx); !vmerrors.Is(err, vmerrors.CodeUnsupported) {
		t.Fatalf("Remove: ожидался VM_ERR_UNSUPPORTED, получено %v", err)
	}

	st, err := p.Status(ctx)
	if !vmerrors.Is(err, vmerrors.CodeUnsupported) {
		t.Fatalf("Status: ожидался VM_ERR_UNSUPPORTED, получено %v", err)
	}
	if st.Present || st.Mismatch {
		t.Fatalf("Status не должен подтверждать наличие правил: %+v", st)
	}
}

func TestStubLifecycleIsNoop(t *testing.T) {
	stub := NewStub()
	if stub.Name() != "policy" {
		t.Fatalf("имя компонента = %q", stub.Name())
	}
	if err := stub.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := stub.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}
