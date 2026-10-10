package services

import "errors"

// UserError is a fixed, human-readable message that is safe to show in a
// browser (GYST-VN-015). Handlers show a UserError's text as is and answer any
// other error with a generic message plus a request reference, so a raw SQL or
// network error can never reach a response body.
//
// Only wrap text that is written in this repository and carries no data from
// the database, the environment or an upstream provider.
type UserError struct{ Message string }

func (e *UserError) Error() string { return e.Message }

// userError builds a UserError from fixed text.
func userError(message string) error { return &UserError{Message: message} }

// UserMessage returns the browser-safe text of err when it is, or wraps, a
// UserError.
func UserMessage(err error) (string, bool) {
	var user *UserError
	if errors.As(err, &user) {
		return user.Message, true
	}
	return "", false
}
