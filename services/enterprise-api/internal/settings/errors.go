package settings

import "errors"

var ErrInvalid = errors.New("invalid settings")

// Error is a settings failure whose message can be shown to the user.
type Error struct {
	Kind    error
	Message string
}

func (e *Error) Error() string { return e.Message }

func (e *Error) Unwrap() error { return e.Kind }

func invalid(message string) error {
	return &Error{Kind: ErrInvalid, Message: message}
}
