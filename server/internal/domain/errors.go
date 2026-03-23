package domain

import "errors"

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrInvalidInput  = errors.New("invalid input")
	ErrRateLimited   = errors.New("rate limited")
	ErrNoProvider    = errors.New("no provider found")
)

// PermanentError signals that retrying will not help.
type PermanentError struct{ Err error }

func (e *PermanentError) Error() string { return e.Err.Error() }
func (e *PermanentError) Unwrap() error { return e.Err }

func Permanent(err error) error {
	if err == nil || IsPermanent(err) {
		return err
	}
	return &PermanentError{Err: err}
}

func IsPermanent(err error) bool {
	var pe *PermanentError
	return errors.As(err, &pe)
}
