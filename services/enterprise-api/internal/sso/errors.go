package sso

import "errors"

var (
	// ErrInvalid means the request is structurally unacceptable.
	ErrInvalid = errors.New("invalid")
	// ErrNotFound means the resource does not exist.
	ErrNotFound = errors.New("not found")
	// ErrConflict means a uniqueness constraint would be violated.
	ErrConflict = errors.New("conflict")
	// ErrNotReady means encryption or required config is unavailable.
	ErrNotReady = errors.New("not ready")
)

// Error is a client-safe domain failure.
type Error struct {
	Kind    error
	Message string
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Kind
}

func invalid(message string) error {
	return &Error{Kind: ErrInvalid, Message: message}
}

func notFound(message string) error {
	return &Error{Kind: ErrNotFound, Message: message}
}

func conflict(message string) error {
	return &Error{Kind: ErrConflict, Message: message}
}

func notReady(message string) error {
	return &Error{Kind: ErrNotReady, Message: message}
}
