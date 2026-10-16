package security

import "errors"

var (
	// ErrInvalid means the request is structurally unacceptable.
	ErrInvalid = errors.New("invalid")
	// ErrLocked means login is temporarily blocked.
	ErrLocked = errors.New("locked")
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

func locked(message string) error {
	return &Error{Kind: ErrLocked, Message: message}
}
