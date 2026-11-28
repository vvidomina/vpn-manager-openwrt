// Package core — ядро демона: порядок инициализации, жизненный цикл
// компонентов, очередь операций изменения и публикация состояния.
// Ядро не выполняет системных изменений: этим занимаются модули (этап 3).
package core

import (
	"context"
	"sync"
	"time"

	"github.com/vvidomina/vpn-manager-openwrt/internal/config"
	"github.com/vvidomina/vpn-manager-openwrt/internal/logging"
	"github.com/vvidomina/vpn-manager-openwrt/internal/state"
	"github.com/vvidomina/vpn-manager-openwrt/internal/vmerrors"
)

// Component — внутренний компонент демона.
type Component interface {
	Name() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// Core — координатор жизненного цикла.
type Core struct {
	cfg   config.Config
	log   logging.Logger
	state *state.Store
	now   func() time.Time
	stopT time.Duration

	mu         sync.RWMutex
	components []Component
	started    []Component
	running    bool
	ready      bool
	activeOp   string
	opSeq      int
}

// Option — настройка ядра.
type Option func(*Core)

// WithClock подменяет часы (тесты).
func WithClock(now func() time.Time) Option { return func(c *Core) { c.now = now } }

// WithStopTimeout задаёт таймаут остановки компонентов.
func WithStopTimeout(d time.Duration) Option { return func(c *Core) { c.stopT = d } }

// New создаёт ядро.
func New(cfg config.Config, log logging.Logger, st *state.Store, opts ...Option) *Core {
	c := &Core{cfg: cfg, log: log, state: st, now: time.Now, stopT: 5 * time.Second}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Register добавляет компонент. Регистрация после старта запрещена.
func (c *Core) Register(comp Component) error {
	if comp == nil {
		return vmerrors.New(vmerrors.CodeInvalidArgument, "core.register", "пустой компонент")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.running {
		return vmerrors.New(vmerrors.CodeInvalidState, "core.register", "регистрация после старта запрещена")
	}
	for _, existing := range c.components {
		if existing.Name() == comp.Name() {
			return vmerrors.New(vmerrors.CodeConflict, "core.register", "компонент уже зарегистрирован: "+comp.Name())
		}
	}
	c.components = append(c.components, comp)
	return nil
}

// Start инициализирует компоненты в порядке регистрации и публикует состояние.
func (c *Core) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return vmerrors.New(vmerrors.CodeInvalidState, "core.start", "ядро уже запущено")
	}
	c.running = true
	c.started = nil
	comps := append([]Component(nil), c.components...)
	c.mu.Unlock()

	for _, comp := range comps {
		if err := comp.Start(ctx); err != nil {
			c.log.Error("компонент не запущен", "component", comp.Name(), "error", err)
			c.closeStarted(ctx)
			c.mu.Lock()
			c.running = false
			c.ready = false
			c.mu.Unlock()
			if code := vmerrors.CodeOf(err); code != "" {
				return err
			}
			return vmerrors.Wrap(vmerrors.CodeInternal, "core.start."+comp.Name(), err)
		}
		c.mu.Lock()
		c.started = append(c.started, comp)
		c.mu.Unlock()
		c.log.Debug("компонент запущен", "component", comp.Name())
	}

	if err := c.state.Set(state.Derive(c.cfg.Intent == config.IntentEnabled, c.now())); err != nil {
		c.closeStarted(ctx)
		c.mu.Lock()
		c.running = false
		c.mu.Unlock()
		return err
	}

	c.mu.Lock()
	c.ready = true
	c.mu.Unlock()
	c.log.Info("ядро готово", "components", len(comps), "intent", string(c.cfg.Intent))
	return nil
}

// Ready сообщает, завершён ли старт каркаса.
func (c *Core) Ready() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ready
}

// Stop останавливает компоненты в обратном порядке. Идемпотентно.
func (c *Core) Stop(ctx context.Context) error {
	c.mu.Lock()
	if !c.running && len(c.started) == 0 {
		c.mu.Unlock()
		return nil
	}
	c.running = false
	c.ready = false
	c.mu.Unlock()

	err := c.closeStarted(ctx)
	c.log.Info("ядро остановлено")

	if setErr := c.state.Set(state.Snapshot{
		State:  state.StateUnknown,
		Reason: "демон остановлен",
	}); setErr != nil && err == nil {
		err = setErr
	}
	return err
}

// closeStarted закрывает запущенные компоненты в обратном порядке,
// ограничивая каждую остановку таймаутом.
func (c *Core) closeStarted(ctx context.Context) error {
	c.mu.Lock()
	started := append([]Component(nil), c.started...)
	c.started = nil
	c.mu.Unlock()

	var firstErr error
	for i := len(started) - 1; i >= 0; i-- {
		comp := started[i]
		stopCtx, cancel := context.WithTimeout(ctx, c.stopT)
		err := comp.Stop(stopCtx)
		cancel()
		if err != nil {
			c.log.Error("компонент не остановлен", "component", comp.Name(), "error", err)
			if firstErr == nil && vmerrors.CodeOf(err) != "" {
				firstErr = err
			} else if firstErr == nil {
				firstErr = vmerrors.Wrap(vmerrors.CodeInternal, "core.stop."+comp.Name(), err)
			}
		}
	}
	return firstErr
}

// Status возвращает копию текущего снимка состояния.
func (c *Core) Status() state.Snapshot { return c.state.Snapshot() }

// Submit ставит операцию в очередь. Параллельная операция запрещена.
// На этапе 1 операция только регистрируется: сетевых действий нет.
func (c *Core) Submit(op string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.running {
		return "", vmerrors.New(vmerrors.CodeInvalidState, "core.submit", "ядро не запущено")
	}
	if c.activeOp != "" {
		return "", vmerrors.New(vmerrors.CodeBusy, "core.submit", "уже выполняется операция "+c.activeOp)
	}
	c.opSeq++
	id := "op-" + itoa(c.opSeq)
	c.activeOp = id
	c.log.Info("операция принята", "op_id", id, "operation", op)
	return id, nil
}

// Complete снимает признак активной операции.
func (c *Core) Complete(opID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.activeOp == "" || c.activeOp != opID {
		return vmerrors.New(vmerrors.CodeNotFound, "core.complete", "неизвестная операция: "+opID)
	}
	c.activeOp = ""
	return nil
}

// ActiveOperation возвращает идентификатор активной операции ("" — нет).
func (c *Core) ActiveOperation() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.activeOp
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
