package restkit_test

import (
	"context"
	"errors"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	rest "github.com/capcom6/go-restkit"
)

func setupTestServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Id", "test-req-123")

		switch r.URL.Path {
		case "/":
			if r.Method != http.MethodGet {
				t.Errorf("Expected method GET, got %s", r.Method)
			}
		case "/headers":
			w.Header().Set("X-Custom", "custom-value")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": "456"}`))
			return
		case "/204":
			w.WriteHeader(http.StatusNoContent)
			return
		case "/400":
			w.Header().Set("X-Error-Hint", "check-format")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"message": "bad request"}`))
			return
		case "/404":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("not found"))
			return
		case "/409":
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte("conflict"))
			return
		case "/500":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("internal server error"))
			return
		case "/corrupt":
			w.WriteHeader(http.StatusOK)
			w.Header().Add("Content-Type", "application/json")
			_, _ = w.Write([]byte("{not a json"))
			return
		}

		if r.URL.Path == "/body" {
			if r.Method != http.MethodPost {
				t.Errorf("Expected method POST, got %s", r.Method)
			}
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
			}

			_, _ = io.ReadAll(r.Body)
			defer r.Body.Close()
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "123", "state": "Pending"}`))
	}))
}

func TestClient_Do(t *testing.T) {
	httpServer := setupTestServer(t)
	defer httpServer.Close()

	type fields struct {
		config rest.Config
	}
	type args struct {
		ctx      context.Context
		method   string
		path     string
		headers  http.Header
		payload  any
		response any
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantErr        bool
		wantStatusCode int
	}{
		{
			name: "Empty method",
			fields: fields{
				config: rest.Config{
					BaseURL: httpServer.URL,
				},
			},
			args: args{
				ctx:    context.Background(),
				method: "",
				path:   "/",
			},
			wantErr: true,
		},
		{
			name: "With body",
			fields: fields{
				config: rest.Config{
					BaseURL: httpServer.URL,
				},
			},
			args: args{
				ctx:    context.Background(),
				method: http.MethodPost,
				path:   "/body",
				payload: map[string]string{
					"foo": "bar",
				},
				response: new(map[string]any),
			},
			wantErr: false,
		},
		{
			name: "HTTP 400 error",
			fields: fields{
				config: rest.Config{
					BaseURL: httpServer.URL,
				},
			},
			args: args{
				ctx:    context.Background(),
				method: http.MethodGet,
				path:   "/400",
			},
			wantErr:        true,
			wantStatusCode: 400,
		},
		{
			name: "HTTP 404 error",
			fields: fields{
				config: rest.Config{
					BaseURL: httpServer.URL,
				},
			},
			args: args{
				ctx:    context.Background(),
				method: http.MethodGet,
				path:   "/404",
			},
			wantErr:        true,
			wantStatusCode: 404,
		},
		{
			name: "HTTP 409 error",
			fields: fields{
				config: rest.Config{
					BaseURL: httpServer.URL,
				},
			},
			args: args{
				ctx:    context.Background(),
				method: http.MethodGet,
				path:   "/409",
			},
			wantErr:        true,
			wantStatusCode: 409,
		},
		{
			name: "HTTP 500 error",
			fields: fields{
				config: rest.Config{
					BaseURL: httpServer.URL,
				},
			},
			args: args{
				ctx:    context.Background(),
				method: http.MethodGet,
				path:   "/500",
			},
			wantErr:        true,
			wantStatusCode: 500,
		},
		{
			name: "No Content response",
			fields: fields{
				config: rest.Config{
					BaseURL: httpServer.URL,
				},
			},
			args: args{
				ctx:    context.Background(),
				method: http.MethodGet,
				path:   "/204",
			},
			wantErr: false,
		},
		{
			name: "Corrupt response",
			fields: fields{
				config: rest.Config{
					BaseURL: httpServer.URL,
				},
			},
			args: args{
				ctx:      context.Background(),
				method:   http.MethodGet,
				path:     "/corrupt",
				response: new(map[string]any),
			},
			wantErr: true,
		},
		{
			name: "Corrupt request",
			fields: fields{
				config: rest.Config{
					BaseURL: httpServer.URL,
				},
			},
			args: args{
				ctx:     context.Background(),
				method:  http.MethodPost,
				path:    "/",
				payload: math.NaN(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := rest.NewClient(tt.fields.config)
			err := c.Do(tt.args.ctx, tt.args.method, tt.args.path, tt.args.headers, tt.args.payload, tt.args.response)

			if (err != nil) != tt.wantErr {
				t.Errorf("Client.Do() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Skip error type checking if we don't expect an error
			if !tt.wantErr {
				return
			}

			// Check if it's an API error with the expected status code
			if apiErr, ok := rest.AsAPIError(err); ok {
				if apiErr.StatusCode != tt.wantStatusCode {
					t.Errorf("Expected API error with status code %d, got %d", tt.wantStatusCode, apiErr.StatusCode)
				}
			}
		})
	}
}

func TestInternalErrorClassification(t *testing.T) {
	httpServer := setupTestServer(t)
	defer httpServer.Close()

	client, _ := rest.NewClient(rest.Config{})
	err := client.Do(context.Background(), "", "/invalid", nil, math.NaN(), nil)
	if !rest.IsInternalError(err) {
		t.Error("Expected internal error for invalid payload")
	}
}

func TestInfrastructureErrorClassification(t *testing.T) {
	client, _ := rest.NewClient(rest.Config{BaseURL: "http://localhost:1"})
	err := client.Do(context.Background(), "GET", "/", nil, nil, nil)
	if !rest.IsInfrastructureError(err) {
		t.Error("Expected infrastructure error for unreachable host")
	}
}

func TestClient_DoWithHeaders_Success(t *testing.T) {
	httpServer := setupTestServer(t)
	defer httpServer.Close()

	client, _ := rest.NewClient(rest.Config{BaseURL: httpServer.URL})

	var result map[string]any
	headers, err := client.DoWithHeaders(context.Background(), http.MethodGet, "/headers", nil, nil, &result)

	if err != nil {
		t.Fatalf("DoWithHeaders() unexpected error: %v", err)
	}
	if headers == nil {
		t.Fatal("DoWithHeaders() returned nil headers")
	}
	if headers.Get("X-Custom") != "custom-value" {
		t.Errorf("Expected X-Custom header 'custom-value', got %q", headers.Get("X-Custom"))
	}
	if result["id"] != "456" {
		t.Errorf("Expected response id '456', got %v", result["id"])
	}
}

func TestClient_DoRAWWithHeaders_Success(t *testing.T) {
	httpServer := setupTestServer(t)
	defer httpServer.Close()

	client, _ := rest.NewClient(rest.Config{BaseURL: httpServer.URL})

	var result map[string]any
	headers, err := client.DoRAWWithHeaders(context.Background(), http.MethodGet, "/headers", nil, nil, &result)

	if err != nil {
		t.Fatalf("DoRAWWithHeaders() unexpected error: %v", err)
	}
	if headers == nil {
		t.Fatal("DoRAWWithHeaders() returned nil headers")
	}
	if headers.Get("X-Request-Id") != "test-req-123" {
		t.Errorf("Expected X-Request-Id header 'test-req-123', got %q", headers.Get("X-Request-Id"))
	}
}

func TestClient_DoWithHeaders_NoContent(t *testing.T) {
	httpServer := setupTestServer(t)
	defer httpServer.Close()

	client, _ := rest.NewClient(rest.Config{BaseURL: httpServer.URL})

	headers, err := client.DoWithHeaders(context.Background(), http.MethodGet, "/204", nil, nil, nil)

	if err != nil {
		t.Fatalf("DoWithHeaders() unexpected error: %v", err)
	}
	if headers == nil {
		t.Fatal("DoWithHeaders() returned nil headers for 204")
	}
	if headers.Get("X-Request-Id") != "test-req-123" {
		t.Errorf("Expected X-Request-Id header on 204, got %q", headers.Get("X-Request-Id"))
	}
}

func TestClient_DoWithHeaders_APIError_WithHeaders(t *testing.T) {
	httpServer := setupTestServer(t)
	defer httpServer.Close()

	client, _ := rest.NewClient(rest.Config{BaseURL: httpServer.URL})

	headers, err := client.DoWithHeaders(context.Background(), http.MethodGet, "/400", nil, nil, nil)

	if err == nil {
		t.Fatal("DoWithHeaders() expected API error")
	}

	if headers == nil {
		t.Fatal("DoWithHeaders() returned nil headers on API error")
	}
	if headers.Get("X-Error-Hint") != "check-format" {
		t.Errorf("Expected X-Error-Hint header 'check-format', got %q", headers.Get("X-Error-Hint"))
	}

	var apiErr *rest.APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("Expected error to be *APIError")
	}
	if apiErr.Headers.Get("X-Error-Hint") != "check-format" {
		t.Errorf("Expected APIError.Headers X-Error-Hint 'check-format', got %q", apiErr.Headers.Get("X-Error-Hint"))
	}
}

func TestClient_Do_BackwardCompat(t *testing.T) {
	httpServer := setupTestServer(t)
	defer httpServer.Close()

	client, _ := rest.NewClient(rest.Config{BaseURL: httpServer.URL})

	var result map[string]any
	err := client.Do(context.Background(), http.MethodGet, "/", nil, nil, &result)

	if err != nil {
		t.Fatalf("Do() unexpected error: %v", err)
	}
	if result["id"] != "123" {
		t.Errorf("Expected id '123', got %v", result["id"])
	}
}

func TestAPIErrorBodyParsing(t *testing.T) {
	httpServer := setupTestServer(t)
	defer httpServer.Close()

	client, _ := rest.NewClient(rest.Config{BaseURL: httpServer.URL})
	err := client.Do(context.Background(), "GET", "/400", nil, nil, nil)

	apiErr, ok := rest.AsAPIError(err)
	if !ok {
		t.Fatal("Expected API error")
	}

	var customErr struct {
		Message string `json:"message"`
	}
	if err := apiErr.ParseError(&customErr); err != nil {
		t.Error("Failed to parse error body")
	}
	if customErr.Message != "bad request" {
		t.Errorf("Unexpected error message: %s", customErr.Message)
	}
}
