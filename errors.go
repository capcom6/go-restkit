package restkit

import (
	"encoding/json"
	"errors"
	"fmt"
)

var (
	ErrInvalidConfig  = errors.New("rest: invalid config")
	ErrEmptyMethod    = errors.New("rest: empty method")
	ErrEmptyErrorBody = errors.New("rest: empty error body")
	ErrUnmarshalJSON  = errors.New("rest: failed to unmarshal body")
)

// ErrorWithBody provides access to raw error response bodies.
// ParseError expects target to be a pointer to a struct; it returns a json.Unmarshal error otherwise.
type ErrorWithBody interface {
	RawBody() []byte             // RawBody returns the raw error response body
	ParseError(target any) error // ParseError attempts to parse the error body
}

// InternalError represents errors in request construction
type InternalError struct {
	Err error  // Underlying error
	Op  string // Operation where error occurred
}

func (e *InternalError) Error() string {
	return fmt.Sprintf("rest: %s: %v", e.Op, e.Err)
}

func (e *InternalError) Unwrap() error { return e.Err }

// newInternalError creates a new InternalError
func newInternalError(op string, err error) *InternalError {
	return &InternalError{Err: err, Op: op}
}

// InfrastructureError represents network-level failures
type InfrastructureError struct {
	Err error
	URL string
}

func (e *InfrastructureError) Error() string {
	return fmt.Sprintf("rest: infrastructure error contacting %s: %v", e.URL, e.Err)
}

func (e *InfrastructureError) Unwrap() error { return e.Err }

// newInfrastructureError creates a new InfrastructureError
func newInfrastructureError(url string, err error) *InfrastructureError {
	return &InfrastructureError{Err: err, URL: url}
}

// APIError represents server responses with error status codes
type APIError struct {
	StatusCode int    // HTTP status code
	URL        string // URL of the request
	Body       []byte // Raw error response body
}

func (e *APIError) Error() string {
	return fmt.Sprintf("rest: API error %d from %s: %s",
		e.StatusCode, e.URL, string(e.Body))
}

// RawBody returns the raw error response body
func (e *APIError) RawBody() []byte {
	return e.Body
}

// ParseError attempts to parse the error body into the provided struct
func (e *APIError) ParseError(target any) error {
	if len(e.Body) == 0 {
		return ErrEmptyErrorBody
	}
	if err := json.Unmarshal(e.Body, target); err != nil {
		return fmt.Errorf("%w: %w", ErrUnmarshalJSON, err)
	}
	return nil
}

// AsAPIError attempts to extract an APIError from an error chain
func AsAPIError(err error) (*APIError, bool) {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr, true
	}
	return nil, false
}

// IsInternalError checks if error is an internal library error
func IsInternalError(err error) bool {
	var target *InternalError
	return errors.As(err, &target)
}

// IsInfrastructureError checks if error is a network infrastructure error
func IsInfrastructureError(err error) bool {
	var target *InfrastructureError
	return errors.As(err, &target)
}

// IsAPIError checks if an error is an API error with a response body
func IsAPIError(err error) bool {
	var target ErrorWithBody
	return errors.As(err, &target)
}

// IsClientError reports whether err is a client (4xx) error.
// This function now works with the new error hierarchy while maintaining backward compatibility.
func IsClientError(err error) bool {
	// Check new API error types first
	apiErr, ok := AsAPIError(err)

	return ok && apiErr.StatusCode >= 400 && apiErr.StatusCode < 500
}

// IsServerError reports whether err is a server (5xx) error.
// This function now works with the new error hierarchy while maintaining backward compatibility.
func IsServerError(err error) bool {
	// Check new API error types first
	apiErr, ok := AsAPIError(err)

	return ok && apiErr.StatusCode >= 500 && apiErr.StatusCode < 600
}

// Ensure APIError implements ErrorWithBody.
var _ ErrorWithBody = (*APIError)(nil)
