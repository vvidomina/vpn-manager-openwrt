// Package txn — транзакции изменений с откатом.
// Контракт — документ №11 §11, фазы — №10 §7.1. Этап 1: модель фаз и
// интерфейс; ни одна транзакция не начинается, поэтому частичных изменений
// системы возникнуть не может.
package txn

import (
	"context"
	"time"

	"github.com/vvidomina/vpn-manager-openwrt/internal/vmerrors"
)

// Phase — фаза транзакции P0–P8 (№10 §7.1). Фаза — это шаг последовательности,
// а не состояние системы: состояние операции описывает OpState.
type Phase string

const (
	PhasePrepare           Phase = "P0_PREPARE"
	PhaseValidate          Phase = "P1_VALIDATE"
	PhasePreflightOutbound Phase = "P2_PREFLIGHT_OUTBOUND"
	PhaseStagePolicy       Phase = "P3_STAGE_POLICY"
	PhaseStartEngine       Phase = "P4_START_ENGINE"
	PhaseApplyPolicy       Phase = "P5_APPLY_POLICY"
	PhaseTrial             Phase = "P6_TRIAL"
	PhaseCommit            Phase = "P7_COMMIT"
	PhaseRollback          Phase = "P8_ROLLBACK"
)

// applyPhases — последовательность фаз применения в порядке №10 §7.1.
var applyPhases = []Phase{
	PhasePrepare,
	PhaseValidate,
	PhasePreflightOutbound,
	PhaseStagePolicy,
	PhaseStartEngine,
	PhaseApplyPolicy,
	PhaseTrial,
	PhaseCommit,
}

// Phases возвращает копию последовательности фаз применения (P0–P7).
// P8 ROLLBACK выполняется только при сбое и в последовательность не входит.
func Phases() []Phase { return append([]Phase(nil), applyPhases...) }

// Valid сообщает, принадлежит ли фаза модели P0–P8.
func (p Phase) Valid() bool {
	if p == PhaseRollback {
		return true
	}
	for _, known := range applyPhases {
		if p == known {
			return true
		}
	}
	return false
}

// OpState — состояние операции (№11 §11).
type OpState string

const (
	StatePreparing   OpState = "preparing"
	StateValidating  OpState = "validating"
	StateStaging     OpState = "staging"
	StateStarting    OpState = "starting"
	StateApplying    OpState = "applying"
	StateTrial       OpState = "trial"
	StateCommitted   OpState = "committed"
	StateRollingBack OpState = "rolling_back"
	StateRolledBack  OpState = "rolled_back"
	StateFailed      OpState = "failed"
)

// Request — запрос apply/switch/restore с ревизией конфигурации.
type Request struct {
	OpID           string
	Reason         string
	ConfigRevision string
}

// Operation — операция транзакции и её текущая фаза.
type Operation struct {
	ID        string
	State     OpState
	Phase     Phase
	Reason    string
	StartedAt time.Time
}

// RecoveryReport — результат восстановления после сбоя или перезагрузки.
type RecoveryReport struct {
	Recovered bool
	Discarded []string
	State     string
}

// Transactions — контракт Transaction Manager (№11 §11). Вызывается только Core.
type Transactions interface {
	Begin(ctx context.Context, req Request) (Operation, error)
	Status(ctx context.Context, opID string) (Operation, error)
	Rollback(ctx context.Context, opID string, reason string) error
	Recover(ctx context.Context) (RecoveryReport, error)
}

// Stub — заглушка этапа 1: любая операция возвращает VM_ERR_UNSUPPORTED.
type Stub struct{}

// NewStub создаёт заглушку.
func NewStub() *Stub { return &Stub{} }

// Name реализует core.Component.
func (*Stub) Name() string { return "txn" }

// Start реализует core.Component; журнал не создаётся.
func (*Stub) Start(ctx context.Context) error { return nil }

// Stop реализует core.Component.
func (*Stub) Stop(ctx context.Context) error { return nil }

// Begin реализует Transactions; транзакция не начинается.
func (*Stub) Begin(ctx context.Context, req Request) (Operation, error) {
	return Operation{}, vmerrors.Unsupported("txn.begin", "транзакции изменений (этап 3)")
}

// Status реализует Transactions; статуса операции нет.
func (*Stub) Status(ctx context.Context, opID string) (Operation, error) {
	return Operation{}, vmerrors.Unsupported("txn.status", "статус транзакции (этап 3)")
}

// Rollback реализует Transactions; откат не выполняется.
func (*Stub) Rollback(ctx context.Context, opID string, reason string) error {
	return vmerrors.Unsupported("txn.rollback", "откат транзакции (этап 3)")
}

// Recover реализует Transactions; восстановление не выполняется.
func (*Stub) Recover(ctx context.Context) (RecoveryReport, error) {
	return RecoveryReport{}, vmerrors.Unsupported("txn.recover", "восстановление после сбоя (этап 3)")
}
