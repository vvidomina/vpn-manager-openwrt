// Package vmerrors содержит утверждённые коды ошибок VM_ERR_* (документ №12 §1)
// и типизированную ошибку с контекстом. Ошибка подсистемы не смешивается с
// состоянием VPN: состояние вычисляется отдельно (internal/state).
package vmerrors

import "fmt"

// Code — стабильный идентификатор ошибки.
type Code string

// Утверждённые коды (№12 §1, №11 §1).
const (
	CodeInvalidArgument Code = "VM_ERR_INVALID_ARGUMENT"
	CodeNotFound        Code = "VM_ERR_NOT_FOUND"
	CodeConflict        Code = "VM_ERR_CONFLICT"
	CodeBusy            Code = "VM_ERR_BUSY"
	CodeForbidden       Code = "VM_ERR_FORBIDDEN"
	CodeUnsupported     Code = "VM_ERR_UNSUPPORTED"
	CodeTimeout         Code = "VM_ERR_TIMEOUT"
	CodeInternal        Code = "VM_ERR_INTERNAL"
	CodeStorage         Code = "VM_ERR_STORAGE"
	CodeValidation      Code = "VM_ERR_VALIDATION"
	CodeInvalidState    Code = "VM_ERR_INVALID_STATE"
	CodeRolledBack      Code = "VM_ERR_ROLLED_BACK"
)

// Error — ошибка с кодом, операцией и исходной причиной.
type Error struct {
	Code Code
	Op   string
	Msg  string
	Err  error
}

func (e *Error) Error() string {
	switch {
	case e.Op != "" && e.Msg != "" && e.Err != nil:
		return fmt.Sprintf("%s: %s: %s: %v", e.Code, e.Op, e.Msg, e.Err)
	case e.Op != "" && e.Msg != "":
		return fmt.Sprintf("%s: %s: %s", e.Code, e.Op, e.Msg)
	case e.Msg != "" && e.Err != nil:
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Msg, e.Err)
	case e.Msg != "":
		return fmt.Sprintf("%s: %s", e.Code, e.Msg)
	case e.Err != nil:
		return fmt.Sprintf("%s: %v", e.Code, e.Err)
	default:
		return string(e.Code)
	}
}

func (e *Error) Unwrap() error { return e.Err }

// New создаёт ошибку без исходной причины.
func New(code Code, op, msg string) *Error {
	return &Error{Code: code, Op: op, Msg: msg}
}

// Wrap оборачивает причину, сохраняя код.
func Wrap(code Code, op string, err error) *Error {
	return &Error{Code: code, Op: op, Err: err}
}

// CodeOf возвращает код ошибки или пустую строку, если код не определён.
func CodeOf(err error) Code {
	for err != nil {
		if e, ok := err.(*Error); ok {
			return e.Code
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return ""
		}
		err = u.Unwrap()
	}
	return ""
}

// Is сообщает, совпадает ли код ошибки с ожидаемым.
func Is(err error, code Code) bool { return CodeOf(err) == code }

// Unsupported — заглушка этапа 1: функциональность не реализована.
func Unsupported(op, feature string) *Error {
	return New(CodeUnsupported, op, "не реализовано на этапе 1: "+feature)
}
