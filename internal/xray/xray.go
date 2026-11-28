// Package xray — граница управления процессом Xray.
// Этап 1: интерфейс и заглушка. Генерация конфигурации, запуск, перезапуск и
// наблюдение не реализуются; реальные вызовы Xray запрещены границами этапа.
// Заглушка намеренно не является core.Component: у Manager своя пара
// Start/Stop с параметрами, а запускать на этапе 1 нечего.
package xray

import (
	"context"

	"github.com/vvidomina/vpn-manager-openwrt/internal/vmerrors"
)

// BuildResult — результат генерации конфигурации.
type BuildResult struct {
	ConfigJSON []byte
	Tag        string
}

// Status — наблюдаемое состояние процесса Xray (не предположение).
type Status struct {
	Running   bool
	PID       int
	Version   string
	StartedAt string
}

// Manager — управление Xray (№18 §7, №10 §3).
type Manager interface {
	Build(ctx context.Context, cfg any) (BuildResult, error)
	Validate(ctx context.Context, configJSON []byte) error
	Start(ctx context.Context, configJSON []byte) (Status, error)
	Stop(ctx context.Context) error
	Status(ctx context.Context) (Status, error)
}

// Stub — заглушка этапа 1: любая операция возвращает VM_ERR_UNSUPPORTED.
type Stub struct{}

// NewStub создаёт заглушку.
func NewStub() *Stub { return &Stub{} }

func (*Stub) Build(ctx context.Context, cfg any) (BuildResult, error) {
	return BuildResult{}, vmerrors.Unsupported("xray.build", "Xray Manager (этап 3)")
}

func (*Stub) Validate(ctx context.Context, configJSON []byte) error {
	return vmerrors.Unsupported("xray.validate", "Xray Manager (этап 3)")
}

func (*Stub) Start(ctx context.Context, configJSON []byte) (Status, error) {
	return Status{}, vmerrors.Unsupported("xray.start", "Xray Manager (этап 3)")
}

func (*Stub) Stop(ctx context.Context) error {
	return vmerrors.Unsupported("xray.stop", "Xray Manager (этап 3)")
}

func (*Stub) Status(ctx context.Context) (Status, error) {
	return Status{}, vmerrors.Unsupported("xray.status", "наблюдение Xray (этап 3)")
}
