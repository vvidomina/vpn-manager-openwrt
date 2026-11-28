// Package policy — политика перехвата трафика (nftables).
// Этап 1: только интерфейс и заглушка. Правила не создаются, firewall не
// изменяется, состояния политики не имитируются.
package policy

import (
	"context"

	"github.com/vvidomina/vpn-manager-openwrt/internal/vmerrors"
)

// Intent — требуемая политика перехвата (№18 §5).
type Intent struct {
	Mode       string // "capture" | "direct" | "block"
	UDPPorts   []PortRange
	ExcludeLAN bool
}

// PortRange — диапазон UDP-портов.
type PortRange struct {
	From int
	To   int
}

// Observed — фактически наблюдаемая политика, полученная из ядра.
type Observed struct {
	Installed bool
	Rules     int
	Table     string
}

// Manager — применение и проверка политики.
type Manager interface {
	Apply(ctx context.Context, in Intent) (Observed, error)
	Verify(ctx context.Context, want Intent) (Observed, error)
	Remove(ctx context.Context) error
}

// Stub — заглушка этапа 1.
type Stub struct{}

// NewStub создаёт заглушку.
func NewStub() *Stub { return &Stub{} }

// Name реализует core.Component.
func (*Stub) Name() string { return "policy" }

// Start реализует core.Component.
func (*Stub) Start(ctx context.Context) error { return nil }

// Stop реализует core.Component.
func (*Stub) Stop(ctx context.Context) error { return nil }

func (*Stub) Apply(ctx context.Context, in Intent) (Observed, error) {
	return Observed{}, vmerrors.Unsupported("policy.apply", "Policy Manager и nftables (этап 3)")
}

func (*Stub) Verify(ctx context.Context, want Intent) (Observed, error) {
	return Observed{}, vmerrors.Unsupported("policy.verify", "проверка политики на роутере (этап 3)")
}

func (*Stub) Remove(ctx context.Context) error {
	return vmerrors.Unsupported("policy.remove", "удаление правил (этап 3)")
}
