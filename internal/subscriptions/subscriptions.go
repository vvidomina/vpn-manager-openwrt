// Package subscriptions — подписки на серверы.
// Контракт — документ №11 §8. Этап 1: интерфейс и заглушка; сеть не
// используется, подписки не загружаются и не разбираются.
package subscriptions

import (
	"context"
	"time"

	"github.com/vvidomina/vpn-manager-openwrt/internal/vmerrors"
)

// Subscription — сохранённая подписка. URL/токен не попадают в журналы.
type Subscription struct {
	ID        string
	Name      string
	URL       string
	Servers   int
	UpdatedAt time.Time
}

// Source — источник подписки: URL или локальный файл.
type Source struct {
	URL  string
	File string
}

// Report — отчёт импорта: сколько принято, сколько отклонено и почему.
type Report struct {
	SubscriptionID string
	Accepted       int
	Rejected       int
	Reasons        []string
}

// Subscriptions — контракт Subscription Manager (№11 §8).
type Subscriptions interface {
	Add(ctx context.Context, src Source) (Subscription, error)
	Update(ctx context.Context, id string) (Report, error)
	Remove(ctx context.Context, id string) error
	Reorder(ctx context.Context, ids []string) error
	List(ctx context.Context) ([]Subscription, error)
}

// Stub — заглушка этапа 1: любая операция возвращает VM_ERR_UNSUPPORTED.
type Stub struct{}

// NewStub создаёт заглушку.
func NewStub() *Stub { return &Stub{} }

// Name реализует core.Component.
func (*Stub) Name() string { return "subscriptions" }

// Start реализует core.Component; сетевых запросов нет.
func (*Stub) Start(ctx context.Context) error { return nil }

// Stop реализует core.Component.
func (*Stub) Stop(ctx context.Context) error { return nil }

// Add реализует Subscriptions; подписка не создаётся.
func (*Stub) Add(ctx context.Context, src Source) (Subscription, error) {
	return Subscription{}, vmerrors.Unsupported("subscriptions.add", "подписки (этап 2/3)")
}

// Update реализует Subscriptions; загрузки нет.
func (*Stub) Update(ctx context.Context, id string) (Report, error) {
	return Report{}, vmerrors.Unsupported("subscriptions.update", "обновление подписки (этап 2/3)")
}

// Remove реализует Subscriptions; удаления нет.
func (*Stub) Remove(ctx context.Context, id string) error {
	return vmerrors.Unsupported("subscriptions.remove", "удаление подписки (этап 2/3)")
}

// Reorder реализует Subscriptions; порядок не сохраняется.
func (*Stub) Reorder(ctx context.Context, ids []string) error {
	return vmerrors.Unsupported("subscriptions.reorder", "порядок подписок (этап 2/3)")
}

// List реализует Subscriptions; список не формируется.
func (*Stub) List(ctx context.Context) ([]Subscription, error) {
	return nil, vmerrors.Unsupported("subscriptions.list", "список подписок (этап 2/3)")
}
