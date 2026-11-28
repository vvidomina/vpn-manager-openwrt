// Package policy — политика перехвата трафика (nftables, fwmark, ip rule).
// Контракт — документ №11 §3. Этап 1: только интерфейс и заглушка. Правила не
// создаются, firewall не изменяется, состояния политики не имитируются.
package policy

import (
	"context"
	"time"

	"github.com/vvidomina/vpn-manager-openwrt/internal/vmerrors"
)

// Mode — режим политики для клиентского трафика.
type Mode string

const (
	ModeCapture Mode = "capture" // трафик устройств направляется в VPN
	ModeDirect  Mode = "direct"  // перехват отсутствует
	ModeBlock   Mode = "block"   // fallback Xray blackhole, без DIRECT
)

// PortRange — диапазон UDP-портов allowlist (№18 §5).
type PortRange struct {
	From int
	To   int
}

// Desired — требуемая политика от Transaction Manager (вход Stage).
// Пользовательские строки здесь не интерпретируются как команды.
type Desired struct {
	Mode         Mode
	UDPPorts     []PortRange
	AlwaysDirect []string // MAC-адреса устройств с приоритетом Always DIRECT
}

// Plan — подготовленная, но не применённая политика (`nft -c` без изменений).
type Plan struct {
	Desired Desired
	Table   string
	Rules   int
}

// Applied — применённая политика; используется для отката.
type Applied struct {
	Plan      Plan
	Table     string
	Rules     int
	AppliedAt time.Time
}

// Status — наблюдаемое состояние политики: наличие, счётчики, расхождения.
type Status struct {
	Present  bool
	Table    string
	Rules    int
	Mismatch bool
}

// Policy — контракт Policy Manager (№11 §3). Вызывается только Transaction
// Manager; Diagnostics читает Status и ничего не меняет.
type Policy interface {
	Stage(ctx context.Context, d Desired) (Plan, error)
	Apply(ctx context.Context, plan Plan) (Applied, error)
	Rollback(ctx context.Context, prev Applied) error
	Remove(ctx context.Context) error
	Status(ctx context.Context) (Status, error)
}

// Stub — заглушка этапа 1: любая операция возвращает VM_ERR_UNSUPPORTED.
type Stub struct{}

// NewStub создаёт заглушку.
func NewStub() *Stub { return &Stub{} }

// Name реализует core.Component.
func (*Stub) Name() string { return "policy" }

// Start реализует core.Component; системных действий нет.
func (*Stub) Start(ctx context.Context) error { return nil }

// Stop реализует core.Component; системных действий нет.
func (*Stub) Stop(ctx context.Context) error { return nil }

// Stage реализует Policy; правила не подготавливаются.
func (*Stub) Stage(ctx context.Context, d Desired) (Plan, error) {
	return Plan{}, vmerrors.Unsupported("policy.stage", "Policy Manager и nftables (этап 3)")
}

// Apply реализует Policy; правила не применяются.
func (*Stub) Apply(ctx context.Context, plan Plan) (Applied, error) {
	return Applied{}, vmerrors.Unsupported("policy.apply", "Policy Manager и nftables (этап 3)")
}

// Rollback реализует Policy; откатывать нечего.
func (*Stub) Rollback(ctx context.Context, prev Applied) error {
	return vmerrors.Unsupported("policy.rollback", "откат политики (этап 3)")
}

// Remove реализует Policy; правила не удаляются.
func (*Stub) Remove(ctx context.Context) error {
	return vmerrors.Unsupported("policy.remove", "удаление правил (этап 3)")
}

// Status реализует Policy; наблюдаемого состояния политики нет.
func (*Stub) Status(ctx context.Context) (Status, error) {
	return Status{}, vmerrors.Unsupported("policy.status", "наблюдение политики (этап 3)")
}
