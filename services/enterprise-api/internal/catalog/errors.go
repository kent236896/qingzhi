package catalog

import "errors"

var (
	ErrInvalid  = errors.New("invalid catalog input")
	ErrNotFound = errors.New("catalog item not found")
	ErrConflict = errors.New("catalog conflict")
	ErrForbidden = errors.New("catalog forbidden")
	ErrNotReady = errors.New("catalog not ready")
)

type Error struct {
	Kind    error
	Message string
}

func (e *Error) Error() string { return e.Message }
func (e *Error) Unwrap() error { return e.Kind }

func invalid(msg string) error  { return &Error{Kind: ErrInvalid, Message: msg} }
func notFound(msg string) error { return &Error{Kind: ErrNotFound, Message: msg} }
func conflict(msg string) error { return &Error{Kind: ErrConflict, Message: msg} }
func forbidden(msg string) error {
	return &Error{Kind: ErrForbidden, Message: msg}
}
func notReady(msg string) error { return &Error{Kind: ErrNotReady, Message: msg} }
