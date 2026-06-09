package apperrors

import "errors"

var (
	ErrConflict      = errors.New("conflict")
	ErrStaleRevision = errors.New("stale revision")
)

const staleRevisionMessage = "record was changed or deleted by another user; refresh and try again"

type typedError struct {
	message string
	cause   error
}

func (e *typedError) Error() string {
	return e.message
}

func (e *typedError) Unwrap() error {
	return e.cause
}

func Conflict(message string) error {
	return &typedError{
		message: message,
		cause:   ErrConflict,
	}
}

func StaleRevision() error {
	return &typedError{
		message: staleRevisionMessage,
		cause:   errors.Join(ErrConflict, ErrStaleRevision),
	}
}

func IsConflict(err error) bool {
	return errors.Is(err, ErrConflict)
}

func IsStaleRevision(err error) bool {
	return errors.Is(err, ErrStaleRevision)
}

func IsNotFoundOrStale(err error) bool {
	return err != nil && (IsStaleRevision(err) || err.Error() == staleRevisionMessage)
}
