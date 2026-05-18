package apperrors

import "errors"

var (
	ErrBadRequest   = errors.New("bad request")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
)

type Error struct {
	Kind    error
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Kind
}

func BadRequest(message string) error {
	return &Error{Kind: ErrBadRequest, Message: message}
}

func Unauthorized(message string) error {
	return &Error{Kind: ErrUnauthorized, Message: message}
}

func Forbidden(message string) error {
	return &Error{Kind: ErrForbidden, Message: message}
}

func NotFound(message string) error {
	return &Error{Kind: ErrNotFound, Message: message}
}

func Conflict(message string) error {
	return &Error{Kind: ErrConflict, Message: message}
}
