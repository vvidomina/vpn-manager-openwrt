package txn

import (
	"context"
	"testing"

	"github.com/vvidomina/vpn-manager-openwrt/internal/vmerrors"
)

func TestPhasesFollowDocumentOrder(t *testing.T) {
	want := []Phase{
		PhasePrepare,
		PhaseValidate,
		PhasePreflightOutbound,
		PhaseStagePolicy,
		PhaseStartEngine,
		PhaseApplyPolicy,
		PhaseTrial,
		PhaseCommit,
	}
	got := Phases()
	if len(got) != len(want) {
		t.Fatalf("фаз = %d, ожидалось %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("фаза %d = %q, ожидалась %q", i, got[i], want[i])
		}
	}
	if PhaseRollback != "P8_ROLLBACK" {
		t.Fatalf("фаза отката = %q, ожидалась P8_ROLLBACK", PhaseRollback)
	}
}

func TestPhasesIsCopy(t *testing.T) {
	got := Phases()
	got[0] = "изменено"
	if Phases()[0] != PhasePrepare {
		t.Fatal("Phases вернула ссылку на внутренний срез")
	}
}

func TestPhaseValid(t *testing.T) {
	for _, p := range append(Phases(), PhaseRollback) {
		if !p.Valid() {
			t.Fatalf("фаза %q должна быть допустимой", p)
		}
	}
	for _, p := range []Phase{"", "P9_UNKNOWN", "PREPARE"} {
		if p.Valid() {
			t.Fatalf("фаза %q не должна считаться допустимой", p)
		}
	}
}

func TestStubImplementsContract(t *testing.T) {
	var m Transactions = NewStub()
	ctx := context.Background()

	op, err := m.Begin(ctx, Request{OpID: "op-1", ConfigRevision: "rev-1"})
	if !vmerrors.Is(err, vmerrors.CodeUnsupported) {
		t.Fatalf("Begin: ожидался VM_ERR_UNSUPPORTED, получено %v", err)
	}
	if op.State != "" || op.Phase != "" {
		t.Fatalf("Begin вернул операцию: %+v", op)
	}

	if _, err := m.Status(ctx, "op-1"); !vmerrors.Is(err, vmerrors.CodeUnsupported) {
		t.Fatalf("Status: ожидался VM_ERR_UNSUPPORTED, получено %v", err)
	}
	if err := m.Rollback(ctx, "op-1", "сбой trial"); !vmerrors.Is(err, vmerrors.CodeUnsupported) {
		t.Fatalf("Rollback: ожидался VM_ERR_UNSUPPORTED, получено %v", err)
	}
	if rep, err := m.Recover(ctx); !vmerrors.Is(err, vmerrors.CodeUnsupported) {
		t.Fatalf("Recover: ожидался VM_ERR_UNSUPPORTED, получено %v", err)
	} else if rep.Recovered || len(rep.Discarded) != 0 {
		t.Fatalf("Recover вернул результат восстановления: %+v", rep)
	}
}

func TestStubLifecycleIsNoop(t *testing.T) {
	stub := NewStub()
	if stub.Name() != "txn" {
		t.Fatalf("имя компонента = %q", stub.Name())
	}
	if err := stub.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := stub.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}
