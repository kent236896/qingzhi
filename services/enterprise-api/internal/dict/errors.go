package dict

import "errors"

var (
	// ErrNotFound means the requested record does not exist.
	ErrNotFound = errors.New("not found")
	// ErrConflict means a uniqueness or dependency rule failed.
	ErrConflict = errors.New("conflict")
	// ErrInvalid means the request is structurally unacceptable.
	ErrInvalid = errors.New("invalid")
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

func conflict(message string) error {
	return &Error{Kind: ErrConflict, Message: message}
}

func notFound(message string) error {
	return &Error{Kind: ErrNotFound, Message: message}
}
