// Package txn — транзакции изменений с откатом (№11 §5).
// Этап 1: интерфейс и заглушка. Ни одна транзакция не начинается, поэтому
// частичных изменений системы возникнуть не может.
package txn

import (
	"context"

	"github.com/vvidomina/vpn-manager-openwrt/internal/vmerrors"
)

// Step — шаг транзакции: Apply выполняется, Rollback отменяет.
type Step interface {
	Name() string
	Apply(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// Manager — управление транзакциями.
type Manager interface {
	Run(ctx context.Context, steps []Step) error
}

// Stub — заглушка этапа 1.
type Stub struct{}

// NewStub создаёт заглушку.
func NewStub() *Stub { return &Stub{} }

// Name реализует core.Component.
func (*Stub) Name() string { return "txn" }

// Start реализует core.Component.
func (*Stub) Start(ctx context.Context) error { return nil }

// Stop реализует core.Component.
func (*Stub) Stop(ctx context.Context) error { return nil }

func (*Stub) Run(ctx context.Context, steps []Step) error {
	return vmerrors.Unsupported("txn.run", "транзакции изменений (этап 3)")
}
