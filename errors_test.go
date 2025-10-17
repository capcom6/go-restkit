package restkit_test

import (
	"errors"
	"fmt"
	"testing"

	liberr "github.com/capcom6/go-restkit"
)

var (
	errWrapped     = errors.New("original error")
	errNotAPI      = errors.New("not API error")
	errNotInfra    = errors.New("not infrastructure error")
	errNotInternal = errors.New("not internal error")
)

func TestInternalError(t *testing.T) {
	t.Parallel()

	internalErr := &liberr.InternalError{
		Err: errWrapped,
		Op:  "test operation",
	}

	if internalErr.Error() != "rest: test operation: original error" {
		t.Errorf("Expected error message 'rest: test operation: original error', got '%s'", internalErr.Error())
	}

	if !errors.Is(internalErr, errWrapped) {
		t.Error("Expected InternalError to wrap the original error")
	}

	if !errors.Is(internalErr.Unwrap(), errWrapped) {
		t.Error("Unwrap() should return the original error")
	}
}

func TestInfrastructureError(t *testing.T) {
	t.Parallel()

	infraErr := &liberr.InfrastructureError{
		Err: errWrapped,
		URL: "http://example.com",
	}

	if infraErr.Error() != "rest: infrastructure error contacting http://example.com: original error" {
		t.Errorf("Expected error message 'rest: infrastructure error contacting http://example.com: original error', got '%s'", infraErr.Error())
	}

	if !errors.Is(infraErr, errWrapped) {
		t.Error("Expected InfrastructureError to wrap the original error")
	}

	if !errors.Is(infraErr.Unwrap(), errWrapped) {
		t.Error("Unwrap() should return the original error")
	}
}

func TestAPIError(t *testing.T) {
	t.Parallel()

	body := []byte(`{"error": "not found", "code": "404"}`)
	apiErr := &liberr.APIError{
		StatusCode: 404,
		URL:        "http://example.com/api",
		Body:       body,
	}

	expectedMsg := "rest: API error 404 from http://example.com/api: " + string(body)
	if apiErr.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, apiErr.Error())
	}

	if string(apiErr.RawBody()) != string(body) {
		t.Error("RawBody() should return the original body")
	}

	// Test ParseError
	var parsed struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}
	if err := apiErr.ParseError(&parsed); err != nil {
		t.Errorf("ParseError() failed: %v", err)
	}

	if parsed.Error != "not found" || parsed.Code != "404" {
		t.Errorf("Parsed error data incorrect: %+v", parsed)
	}

	// Test ParseError with empty body
	emptyAPIErr := &liberr.APIError{
		StatusCode: 400,
		URL:        "http://example.com/api",
		Body:       []byte{},
	}

	if emptyAPIErr.ParseError(&parsed) == nil {
		t.Error("ParseError() should fail with empty body")
	}
}

func TestAsAPIError(t *testing.T) {
	t.Parallel()

	// Test with APIError
	apiErr := &liberr.APIError{
		StatusCode: 400,
		URL:        "http://example.com",
		Body:       []byte("error"),
	}

	extracted, ok := liberr.AsAPIError(apiErr)
	if !ok || extracted != apiErr {
		t.Error("AsAPIError should extract APIError from itself")
	}

	// Test with wrapped APIError
	wrapped := fmt.Errorf("wrapper: %w", apiErr)
	extracted, ok = liberr.AsAPIError(wrapped)
	if !ok || extracted != apiErr {
		t.Error("AsAPIError should extract APIError from wrapped error")
	}

	// Test with non-APIError
	_, ok = liberr.AsAPIError(errNotAPI)
	if ok {
		t.Error("AsAPIError should return false for non-APIError")
	}
}

func TestIsInternalError(t *testing.T) {
	t.Parallel()

	internalErr := &liberr.InternalError{
		Err: errWrapped,
		Op:  "test",
	}

	if !liberr.IsInternalError(internalErr) {
		t.Error("IsInternalError should return true for InternalError")
	}

	if liberr.IsInternalError(errNotInternal) {
		t.Error("IsInternalError should return false for non-InternalError")
	}
}

func TestIsInfrastructureError(t *testing.T) {
	t.Parallel()

	infraErr := &liberr.InfrastructureError{
		Err: errWrapped,
		URL: "http://example.com",
	}

	if !liberr.IsInfrastructureError(infraErr) {
		t.Error("IsInfrastructureError should return true for InfrastructureError")
	}

	if liberr.IsInfrastructureError(errNotInfra) {
		t.Error("IsInfrastructureError should return false for non-InfrastructureError")
	}
}

func TestIsAPIError(t *testing.T) {
	t.Parallel()

	apiErr := &liberr.APIError{
		StatusCode: 400,
		URL:        "http://example.com",
		Body:       []byte("error"),
	}

	if !liberr.IsAPIError(apiErr) {
		t.Error("IsAPIError should return true for APIError")
	}

	if liberr.IsAPIError(errNotAPI) {
		t.Error("IsAPIError should return false for non-APIError")
	}
}
