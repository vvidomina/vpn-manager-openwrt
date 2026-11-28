// Package state — состояния VPN Manager, признаки S1–S6, Snapshot и Report.
// Модель состояний — документ №10 §5 и №20 §3; правила свежести — №20 §7.
// Состояние вычисляется по признакам и никогда не является записанным флагом.
package state

import (
	"sync"
	"time"

	"github.com/vvidomina/vpn-manager-openwrt/internal/vmerrors"
)

// State — состояние системы (№10 §5.1). В этапе 1 вычисляются только
// DISABLED_BY_USER, STARTING, DEGRADED и UNKNOWN: признаки S2–S6 неизвестны,
// поэтому ACTIVE недостижим, а переходы сетевых состояний не реализуются.
type State string

const (
	StateDisabledByUser  State = "DISABLED_BY_USER"
	StateStarting        State = "STARTING"
	StateActive          State = "ACTIVE"
	StateDegraded        State = "DEGRADED"
	StateEmergencyDirect State = "EMERGENCY_DIRECT"
	StateWANDown         State = "WAN_DOWN"
	StateFallbackDirect  State = "FALLBACK_DIRECT"
	StateFallbackBlock   State = "FALLBACK_BLOCK"
	StateApplying        State = "APPLYING"
	StateRollingBack     State = "ROLLING_BACK"
	StateStopping        State = "STOPPING"
	StateMaintenance     State = "MAINTENANCE"
	StateUnknown         State = "UNKNOWN"
)

// Имена признаков (№10 §5.2). S1 — намерение, S5 — наблюдаемость outbound,
// которая в обязательный минимум не входит.
const (
	SignalIntent    = "S1"
	SignalProcess   = "S2"
	SignalConfig    = "S3"
	SignalPolicy    = "S4"
	SignalOutbounds = "S5"
	SignalTraffic   = "S6"
)

// requiredSignals — признаки, без которых состояние ACTIVE запрещено.
var requiredSignals = []string{SignalProcess, SignalConfig, SignalPolicy, SignalTraffic}

// Signal — один признак состояния с временем наблюдения.
type Signal struct {
	Name  string    `json:"name"`
	Value string    `json:"value"`
	At    time.Time `json:"at"`
	Known bool      `json:"known"`
}

// Snapshot — неизменяемый снимок состояния для API и диагностики.
type Snapshot struct {
	State      State     `json:"state"`
	Reason     string    `json:"reason,omitempty"`
	Signals    []Signal  `json:"signals"`
	ObservedAt time.Time `json:"observed_at"`
}

// Report — диагностический отчёт (read-only).
type Report struct {
	Version       string   `json:"version"`
	SchemaVersion int      `json:"schema_version"`
	State         Snapshot `json:"state"`
	Warnings      []string `json:"warnings"`
}

// Validate проверяет инвариант: ACTIVE невозможен без известных S2, S3, S4, S6.
func (s Snapshot) Validate() error {
	if s.State != StateActive {
		return nil
	}
	for _, name := range requiredSignals {
		sig, ok := s.signal(name)
		if !ok || !sig.Known {
			return vmerrors.New(vmerrors.CodeInvalidState, "state.validate",
				"состояние ACTIVE требует известного признака "+name)
		}
	}
	return nil
}

func (s Snapshot) signal(name string) (Signal, bool) {
	for _, sig := range s.Signals {
		if sig.Name == name {
			return sig, true
		}
	}
	return Signal{}, false
}

// Clone делает глубокую копию снимка.
func (s Snapshot) Clone() Snapshot {
	out := s
	out.Signals = append([]Signal(nil), s.Signals...)
	return out
}

// Store — потокобезопасное хранилище текущего снимка.
type Store struct {
	mu   sync.RWMutex
	snap Snapshot
	now  func() time.Time
}

// NewStore создаёт хранилище с начальным состоянием UNKNOWN.
func NewStore() *Store {
	return &Store{
		snap: Snapshot{State: StateUnknown, Reason: "не инициализировано"},
		now:  time.Now,
	}
}

// NewStoreWithClock — хранилище с подменяемыми часами (для тестов).
func NewStoreWithClock(now func() time.Time) *Store {
	s := NewStore()
	s.now = now
	return s
}

// Set сохраняет снимок, подставляя время наблюдения, если оно не задано,
// и отклоняя нарушение инварианта ACTIVE.
func (s *Store) Set(snap Snapshot) error {
	snap = snap.Clone()
	if snap.ObservedAt.IsZero() {
		snap.ObservedAt = s.now()
	}
	if err := snap.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snap = snap
	return nil
}

// Snapshot возвращает копию текущего снимка.
func (s *Store) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snap.Clone()
}

// Derive вычисляет стартовое состояние по намерению пользователя.
// В этапе 1 известен только S1; остальные признаки помечаются unknown,
// поэтому состояние выше STARTING не поднимается.
func Derive(intentEnabled bool, now time.Time) Snapshot {
	snap := Snapshot{
		ObservedAt: now,
		Signals: []Signal{
			{Name: SignalIntent, Value: intentValue(intentEnabled), At: now, Known: true},
			{Name: SignalProcess, At: now, Known: false},
			{Name: SignalConfig, At: now, Known: false},
			{Name: SignalPolicy, At: now, Known: false},
			{Name: SignalOutbounds, At: now, Known: false},
			{Name: SignalTraffic, At: now, Known: false},
		},
	}
	if intentEnabled {
		snap.State = StateStarting
		snap.Reason = "признаки S2–S6 в этапе 1 не наблюдаются (not_implemented)"
	} else {
		snap.State = StateDisabledByUser
		snap.Reason = "intent=disabled"
	}
	return snap
}

func intentValue(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}
