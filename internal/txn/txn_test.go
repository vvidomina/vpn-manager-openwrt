package txn

import (
	"context"
	"testing"

	"github.com/vvidomina/vpn-manager-openwrt/internal/vmerrors"
)

type countingStep struct {
	name    string
	applied int
}

func (s *countingStep) Name() string                       { return s.name }
func (s *countingStep) Apply(ctx context.Context) error    { s.applied++; return nil }
func (s *countingStep) Rollback(ctx context.Context) error { return nil }

func TestStubRunsNoSteps(t *testing.T) {
	step := &countingStep{name: "nftables"}
	err := NewStub().Run(context.Background(), []Step{step})
	if !vmerrors.Is(err, vmerrors.CodeUnsupported) {
		t.Fatalf("ожидался VM_ERR_UNSUPPORTED, получено %v", err)
	}
	if step.applied != 0 {
		t.Fatalf("шаг выполнен %d раз, ожидалось 0", step.applied)
	}
}
