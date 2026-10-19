package knowledge

import "errors"

var (
	ErrNotFound  = errors.New("not found")
	ErrConflict  = errors.New("conflict")
	ErrInvalid   = errors.New("invalid")
	ErrForbidden = errors.New("forbidden")
	ErrNotReady  = errors.New("not ready")
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

func (e *Error) Unwrap() error { return e.Kind }

func invalid(message string) error  { return &Error{Kind: ErrInvalid, Message: message} }
func notFound(message string) error { return &Error{Kind: ErrNotFound, Message: message} }
func conflict(message string) error { return &Error{Kind: ErrConflict, Message: message} }
func forbidden(message string) error {
	return &Error{Kind: ErrForbidden, Message: message}
}
func notReady(message string) error { return &Error{Kind: ErrNotReady, Message: message} }
