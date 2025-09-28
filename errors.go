package restkit

import (
	"errors"
	"fmt"
)

var (
	// ErrAPIError is the root category for errors originating from the API.
	ErrAPIError = errors.New("api error")
	// ErrClient represents 4xx-class client-side errors.
	ErrClient = fmt.Errorf("%w: client error", ErrAPIError)
	// ErrServer represents 5xx-class server-side errors.
	ErrServer = fmt.Errorf("%w: server error", ErrAPIError)
)

var (
	// ErrBadRequest indicates a 400 Bad Request.
	ErrBadRequest = fmt.Errorf("%w: validation failed", ErrClient)
	// ErrConflict indicates a 409 Conflict.
	ErrConflict = fmt.Errorf("%w: conflict", ErrClient)
)

// IsAPIError reports whether err is in the API error category.
func IsAPIError(err error) bool {
	return errors.Is(err, ErrAPIError)
}

// IsClientError reports whether err is a client (4xx) error.
func IsClientError(err error) bool {
	return errors.Is(err, ErrClient)
}

// IsServerError reports whether err is a server (5xx) error.
func IsServerError(err error) bool {
	return errors.Is(err, ErrServer)
}

// IsConflict reports whether err indicates a 409 Conflict.
func IsConflict(err error) bool {
	return errors.Is(err, ErrConflict)
}

// IsBadRequest reports whether err indicates a 400 Bad Request.
func IsBadRequest(err error) bool {
	return errors.Is(err, ErrBadRequest)
}
