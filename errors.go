package zouwu

import "net/http"

// define error
var (
	ErrNotFound            = NewHTTPError(http.StatusNotFound)
	ErrUnauthorized        = NewHTTPError(http.StatusUnauthorized)
	ErrForbidden           = NewHTTPError(http.StatusForbidden)
	ErrMethodNotAllowed    = NewHTTPError(http.StatusMethodNotAllowed)
	ErrTooManyRequests     = NewHTTPError(http.StatusTooManyRequests)
	ErrBadRequest          = NewHTTPError(http.StatusBadRequest)
	ErrBadGateway          = NewHTTPError(http.StatusBadGateway)
	ErrInternalServerError = NewHTTPError(http.StatusInternalServerError)
	ErrRequestTimeout      = NewHTTPError(http.StatusRequestTimeout)
	ErrServiceUnavailable  = NewHTTPError(http.StatusServiceUnavailable)
)

// Error error
type Error struct {
	Code    int
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

// NewHTTPError creates a new HTTPError instance.
func NewHTTPError(code int, message ...string) *Error {
	he := &Error{Code: code, Message: http.StatusText(code)}
	if len(message) > 0 {
		he.Message = message[0]
	}
	return he
}
