// Package profiles — профили серверов.
// Этап 1: интерфейс и заглушка, хранение профилей не реализуется.
package profiles

import (
	"context"

	"github.com/vvidomina/vpn-manager-openwrt/internal/vmerrors"
)

// Profile — профиль сервера. Секреты в журналы и отчёты не попадают.
type Profile struct {
	ID     string
	Name   string
	Tag    string
	Active bool
}

// Registry — реестр профилей.
type Registry interface {
	List(ctx context.Context) ([]Profile, error)
	Activate(ctx context.Context, id string) error
}

// Stub — заглушка этапа 1.
type Stub struct{}

// NewStub создаёт заглушку.
func NewStub() *Stub { return &Stub{} }

// Name реализует core.Component.
func (*Stub) Name() string { return "profiles" }

// Start реализует core.Component.
func (*Stub) Start(ctx context.Context) error { return nil }

// Stop реализует core.Component.
func (*Stub) Stop(ctx context.Context) error { return nil }

func (*Stub) List(ctx context.Context) ([]Profile, error) {
	return nil, vmerrors.Unsupported("profiles.list", "Profiles (этап 4)")
}

func (*Stub) Activate(ctx context.Context, id string) error {
	return vmerrors.Unsupported("profiles.activate", "переключение профиля (этап 3)")
}
