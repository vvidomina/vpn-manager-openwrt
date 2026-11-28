// Package api — граница взаимодействия с LuCI.
// Транспорт (ubus) подключается на следующем этапе; здесь определены формат
// запроса и ответа, набор методов №12 §2 и отображение ошибок в VM_ERR_*.
// Собственный HTTP API и Unix-сокет не создаются.
package api

import (
	"context"
	"encoding/json"
	"time"

	"github.com/vvidomina/vpn-manager-openwrt/internal/state"
	"github.com/vvidomina/vpn-manager-openwrt/internal/vmerrors"
)

// Имена методов (№12 §2).
const (
	MethodStatus    = "vpnmanager.status"
	MethodVersion   = "vpnmanager.version"
	MethodOperation = "vpnmanager.operation"
)

// Core — минимальная зависимость API от ядра.
type Core interface {
	Status() state.Snapshot
	Ready() bool
	ActiveOperation() string
	Submit(op string) (string, error)
	Complete(opID string) error
}

// Request — запрос LuCI → демон.
type Request struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

// Failure — ошибка в ответе.
type Failure struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Response — ответ демона → LuCI.
type Response struct {
	OK     bool     `json:"ok"`
	Result any      `json:"result,omitempty"`
	Error  *Failure `json:"error,omitempty"`
}

// StatusResult — результат vpnmanager.status.
type StatusResult struct {
	State       string         `json:"state"`
	Reason      string         `json:"reason,omitempty"`
	Config      string         `json:"config,omitempty"`
	ConfigError string         `json:"config_error,omitempty"`
	Ready       bool           `json:"ready"`
	ObservedAt  time.Time      `json:"observed_at"`
	Signals     []state.Signal `json:"signals"`
}

// HealthResult — внутренний health-ping демона для границы API.
// Это не метод ubus: в реестре №12 §6 такого метода нет, а регистрировать
// метод вне реестра запрещено (№44 §3.5). Проверка только читает состояние.
type HealthResult struct {
	OK         bool      `json:"ok"`
	State      string    `json:"state"`
	Config     string    `json:"config,omitempty"`
	Ready      bool      `json:"ready"`
	Reason     string    `json:"reason,omitempty"`
	ObservedAt time.Time `json:"observed_at"`
}

// VersionResult — результат vpnmanager.version.
type VersionResult struct {
	Version string `json:"version"`
}

// OperationParams — параметры vpnmanager.operation.
type OperationParams struct {
	ID string `json:"id,omitempty"`
}

// OperationResult — результат vpnmanager.operation.
type OperationResult struct {
	Active string `json:"active,omitempty"`
}

// Service — реализация методов API.
type Service struct {
	core    Core
	version string
}

// New создаёт сервис API.
func New(core Core, version string) *Service {
	return &Service{core: core, version: version}
}

// Call выполняет запрос. Ошибка всегда возвращается в теле ответа.
func (s *Service) Call(ctx context.Context, req Request) Response {
	if err := ctx.Err(); err != nil {
		return fail(vmerrors.Wrap(vmerrors.CodeTimeout, "api.call", err))
	}
	switch req.Method {
	case MethodStatus:
		return s.status()
	case MethodVersion:
		return Response{OK: true, Result: VersionResult{Version: s.version}}
	case MethodOperation:
		return s.operation(req.Params)
	default:
		return fail(vmerrors.New(vmerrors.CodeUnsupported, "api.call", "неизвестный метод: "+req.Method))
	}
}

func (s *Service) status() Response {
	snap := s.core.Status()
	return Response{OK: true, Result: StatusResult{
		State:       string(snap.State),
		Reason:      snap.Reason,
		Config:      string(snap.Config),
		ConfigError: snap.ConfigError,
		Ready:       s.core.Ready(),
		ObservedAt:  snap.ObservedAt,
		Signals:     snap.Signals,
	}}
}

// Health выполняет внутренний health-ping. Успех означает «ядро готово и
// нет запрещающих состояний»; недоступность не маскируется успешным ответом.
func (s *Service) Health(ctx context.Context) (HealthResult, error) {
	if err := ctx.Err(); err != nil {
		return HealthResult{}, vmerrors.Wrap(vmerrors.CodeTimeout, "api.health", err)
	}
	snap := s.core.Status()
	ready := s.core.Ready()
	res := HealthResult{
		State:      string(snap.State),
		Config:     string(snap.Config),
		Ready:      ready,
		Reason:     snap.Reason,
		ObservedAt: snap.ObservedAt,
	}
	res.OK = ready && snap.Config != state.ConfigInvalid &&
		snap.State != state.StateUnknown &&
		snap.State != state.StateDegraded &&
		snap.State != state.StateEmergencyDirect
	return res, nil
}

func (s *Service) operation(params json.RawMessage) Response {
	var p OperationParams
	if len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return fail(vmerrors.Wrap(vmerrors.CodeInvalidArgument, "api.operation", err))
		}
	}
	active := s.core.ActiveOperation()
	if p.ID == "" {
		return Response{OK: true, Result: OperationResult{Active: active}}
	}
	if active != p.ID {
		return fail(vmerrors.New(vmerrors.CodeNotFound, "api.operation", "неизвестная операция: "+p.ID))
	}
	return Response{OK: true, Result: OperationResult{Active: active}}
}

func fail(err error) Response {
	code := vmerrors.CodeOf(err)
	if code == "" {
		code = vmerrors.CodeInternal
	}
	return Response{Error: &Failure{Code: string(code), Message: err.Error()}}
}
