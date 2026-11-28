// Package devices — учёт устройств и режим Always DIRECT.
// Контракт — документ №11 §7. Этап 1: интерфейс и заглушка; системные
// источники (leases, neigh, iwinfo, bridge fdb) не читаются.
package devices

import (
	"context"
	"time"

	"github.com/vvidomina/vpn-manager-openwrt/internal/vmerrors"
)

// Mode — режим устройства относительно VPN-политики (№11 §7).
// Always DIRECT имеет приоритет над VPN-политикой и UDP allowlist; трафик
// такого устройства не блокируется.
type Mode string

const (
	ModeDefault      Mode = "default"
	ModeAlwaysDirect Mode = "always_direct"
)

// Device — устройство сети. MAC — единственный устойчивый идентификатор.
type Device struct {
	MAC      string
	IP       string
	Hostname string
	Mode     Mode
	Online   bool
	LastSeen time.Time
}

// Filter — фильтр списка устройств.
type Filter struct {
	OnlineOnly bool
	MAC        string
}

// Assignment — назначение режима, которое получает Policy Manager.
type Assignment struct {
	MAC  string
	Mode Mode
}

// Devices — контракт Device Manager (№11 §7). Правила не применяет.
type Devices interface {
	List(ctx context.Context, f Filter) ([]Device, error)
	SetMode(ctx context.Context, mac string, mode Mode) error
	Merge(ctx context.Context, macs []string) error
	Resolve(ctx context.Context) ([]Assignment, error)
}

// Stub — заглушка этапа 1: любая операция возвращает VM_ERR_UNSUPPORTED.
type Stub struct{}

// NewStub создаёт заглушку.
func NewStub() *Stub { return &Stub{} }

// Name реализует core.Component.
func (*Stub) Name() string { return "devices" }

// Start реализует core.Component; источники не читаются.
func (*Stub) Start(ctx context.Context) error { return nil }

// Stop реализует core.Component.
func (*Stub) Stop(ctx context.Context) error { return nil }

// List реализует Devices; список устройств не формируется.
func (*Stub) List(ctx context.Context, f Filter) ([]Device, error) {
	return nil, vmerrors.Unsupported("devices.list", "Device Manager (этап 2/3)")
}

// SetMode реализует Devices; режим не меняется.
func (*Stub) SetMode(ctx context.Context, mac string, mode Mode) error {
	return vmerrors.Unsupported("devices.set_mode", "назначение режима устройства (этап 2/3)")
}

// Merge реализует Devices; записи не объединяются.
func (*Stub) Merge(ctx context.Context, macs []string) error {
	return vmerrors.Unsupported("devices.merge", "объединение записей устройств (этап 2/3)")
}

// Resolve реализует Devices; назначения не вычисляются.
func (*Stub) Resolve(ctx context.Context) ([]Assignment, error) {
	return nil, vmerrors.Unsupported("devices.resolve", "назначения устройств (этап 2/3)")
}
