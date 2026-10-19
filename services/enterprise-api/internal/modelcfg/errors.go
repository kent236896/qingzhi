package modelcfg

import "errors"

var (
	ErrNotFound  = errors.New("not found")
	ErrConflict  = errors.New("conflict")
	ErrInvalid   = errors.New("invalid")
	ErrForbidden = errors.New("forbidden")
	ErrNotReady  = errors.New("not ready")
	ErrTimeout   = errors.New("timeout")
)

// Error is a client-safe failure. Data carries structured conflict details.
type Error struct {
	Kind    error
	Message string
	Data    any
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.Kind }

func invalid(message string) error { return &Error{Kind: ErrInvalid, Message: message} }
func notFound(message string) error {
	return &Error{Kind: ErrNotFound, Message: message}
}
func forbidden(message string) error {
	return &Error{Kind: ErrForbidden, Message: message}
}
func notReady(message string) error { return &Error{Kind: ErrNotReady, Message: message} }
func timeout(message string) error  { return &Error{Kind: ErrTimeout, Message: message} }

func conflictData(message string, data any) error {
	return &Error{Kind: ErrConflict, Message: message, Data: data}
}

// RemoteError is a non-success WeKnora response. Error text never includes the body.
type RemoteError struct {
	Status  int
	Code    int
	Message string
	Details jsonRaw
}

type jsonRaw = []byte

func (e *RemoteError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	return "模型服务请求失败"
}
