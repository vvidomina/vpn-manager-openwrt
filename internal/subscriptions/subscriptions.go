// Package subscriptions — подписки на серверы.
// Этап 1: интерфейс и заглушка. Сеть не используется, подписки не загружаются.
package subscriptions

import (
	"context"

	"github.com/vvidomina/vpn-manager-openwrt/internal/vmerrors"
)

// Subscription — подписка.
type Subscription struct {
	ID        string
	URL       string
	Count     int
	UpdatedAt string
}

// Provider — работа с подписками.
type Provider interface {
	List(ctx context.Context) ([]Subscription, error)
	Refresh(ctx context.Context, id string) (Subscription, error)
}

// Stub — заглушка этапа 1.
type Stub struct{}

// NewStub создаёт заглушку.
func NewStub() *Stub { return &Stub{} }

// Name реализует core.Component.
func (*Stub) Name() string { return "subscriptions" }

// Start реализует core.Component.
func (*Stub) Start(ctx context.Context) error { return nil }

// Stop реализует core.Component.
func (*Stub) Stop(ctx context.Context) error { return nil }

func (*Stub) List(ctx context.Context) ([]Subscription, error) {
	return nil, vmerrors.Unsupported("subscriptions.list", "Subscriptions (этап 4)")
}

func (*Stub) Refresh(ctx context.Context, id string) (Subscription, error) {
	return Subscription{}, vmerrors.Unsupported("subscriptions.refresh", "загрузка подписок (этап 4)")
}
