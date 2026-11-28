// Package profiles — профили маршрутизации и их назначения серверам.
// Контракт — документ №11 §9. Этап 1: интерфейс и заглушка; профили не
// хранятся и не компилируются, конфигурация Xray не генерируется.
package profiles

import (
	"context"

	"github.com/vvidomina/vpn-manager-openwrt/internal/vmerrors"
)

// PortRange — диапазон UDP-портов профиля.
type PortRange struct {
	From int
	To   int
}

// Profile — профиль маршрутизации (routeOrder, списки, DomainStrategy,
// GlobalProxy, UDP). Секреты в профиле не хранятся.
type Profile struct {
	ID             string
	Name           string
	RouteOrder     []string
	DomainStrategy string
	GlobalProxy    bool
	UDPAllowlist   []PortRange
	Restricted     bool // профиль подписки с ограничениями (D-07/D-13)
}

// Ruleset — детерминированный результат компиляции профиля. Формат правил
// определяет Xray Manager (этап 3), а не этот пакет.
type Ruleset struct {
	ProfileID string
	Rules     []string
}

// Profiles — контракт Profile Manager (№11 §9).
type Profiles interface {
	Save(ctx context.Context, p Profile) error
	Assign(ctx context.Context, serverID, profileID string) error
	Compile(ctx context.Context, p Profile) (Ruleset, error)
	SetPending(ctx context.Context, profileID string) error
	List(ctx context.Context) ([]Profile, error)
}

// Stub — заглушка этапа 1: любая операция возвращает VM_ERR_UNSUPPORTED.
type Stub struct{}

// NewStub создаёт заглушку.
func NewStub() *Stub { return &Stub{} }

// Name реализует core.Component.
func (*Stub) Name() string { return "profiles" }

// Start реализует core.Component.
func (*Stub) Start(ctx context.Context) error { return nil }

// Stop реализует core.Component.
func (*Stub) Stop(ctx context.Context) error { return nil }

// Save реализует Profiles; профиль не сохраняется.
func (*Stub) Save(ctx context.Context, p Profile) error {
	return vmerrors.Unsupported("profiles.save", "хранение профилей (этап 2/3)")
}

// Assign реализует Profiles; назначение не выполняется.
func (*Stub) Assign(ctx context.Context, serverID, profileID string) error {
	return vmerrors.Unsupported("profiles.assign", "назначение профиля (этап 2/3)")
}

// Compile реализует Profiles; правила не компилируются.
func (*Stub) Compile(ctx context.Context, p Profile) (Ruleset, error) {
	return Ruleset{}, vmerrors.Unsupported("profiles.compile", "компиляция профиля (этап 3)")
}

// SetPending реализует Profiles; признак Pending reconnect не ставится.
func (*Stub) SetPending(ctx context.Context, profileID string) error {
	return vmerrors.Unsupported("profiles.set_pending", "Pending reconnect (этап 3)")
}

// List реализует Profiles; список не формируется.
func (*Stub) List(ctx context.Context) ([]Profile, error) {
	return nil, vmerrors.Unsupported("profiles.list", "список профилей (этап 2/3)")
}
