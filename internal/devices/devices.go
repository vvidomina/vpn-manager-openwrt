// Package devices — учёт устройств и их исключений.
// Этап 1: интерфейс и заглушка, чтение системных источников не выполняется.
package devices

import (
	"context"

	"github.com/vvidomina/vpn-manager-openwrt/internal/vmerrors"
)

// Device — устройство сети.
type Device struct {
	MAC      string
	IP       string
	Hostname string
	Excluded bool
}

// Registry — реестр устройств.
type Registry interface {
	List(ctx context.Context) ([]Device, error)
	SetExcluded(ctx context.Context, mac string, excluded bool) error
}

// Stub — заглушка этапа 1.
type Stub struct{}

// NewStub создаёт заглушку.
func NewStub() *Stub { return &Stub{} }

// Name реализует core.Component.
func (*Stub) Name() string { return "devices" }

// Start реализует core.Component.
func (*Stub) Start(ctx context.Context) error { return nil }

// Stop реализует core.Component.
func (*Stub) Stop(ctx context.Context) error { return nil }

func (*Stub) List(ctx context.Context) ([]Device, error) {
	return nil, vmerrors.Unsupported("devices.list", "Devices Registry (этап 4)")
}

func (*Stub) SetExcluded(ctx context.Context, mac string, excluded bool) error {
	return vmerrors.Unsupported("devices.exclude", "исключения устройств (этап 4)")
}
