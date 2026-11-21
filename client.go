package restkit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Config struct {
	Client  *http.Client // Optional HTTP Client, defaults to `http.DefaultClient`
	BaseURL string       // Optional base URL
}

type Client struct {
	client  *http.Client
	baseURL *url.URL
}

// Do performs an HTTP request with JSON serialization of payload and response.
// Returns an error classified as InternalError, InfrastructureError, or APIError.
func (c *Client) Do(ctx context.Context, method, path string, headers http.Header, payload, response any) error {
	_, err := c.DoWithHeaders(ctx, method, path, headers, payload, response)
	return err
}

// DoWithHeaders is like Do but also returns the HTTP response headers.
// Headers are returned on both success and API error (4xx/5xx) responses,
// allowing access to headers such as Retry-After, X-RateLimit-Remaining, etc.
func (c *Client) DoWithHeaders(
	ctx context.Context,
	method, path string,
	headers http.Header,
	payload, response any,
) (http.Header, error) {
	var reqBody io.Reader
	if payload != nil {
		jsonBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, newInternalError("DoWithHeaders", fmt.Errorf("failed to marshal payload: %w", err))
		}
		reqBody = bytes.NewReader(jsonBytes)
	}

	if headers == nil {
		headers = http.Header{}
	} else {
		headers = headers.Clone()
	}
	if headers.Get("Accept") == "" {
		headers.Set("Accept", "application/json")
	}
	if reqBody != nil && headers.Get("Content-Type") == "" {
		headers.Set("Content-Type", "application/json")
	}

	return c.do(ctx, method, path, headers, reqBody, response)
}

// DoRAW performs a raw HTTP request. Unlike Do, the payload is sent as-is
// without JSON serialization. Returns an error classified as InternalError,
// InfrastructureError, or APIError.
func (c *Client) DoRAW(
	ctx context.Context,
	method, path string,
	headers http.Header,
	payload io.Reader,
	response any,
) error {
	_, err := c.DoRAWWithHeaders(ctx, method, path, headers, payload, response)
	return err
}

// DoRAWWithHeaders is like DoRAW but also returns the HTTP response headers.
// Headers are returned on both success and API error (4xx/5xx) responses.
func (c *Client) DoRAWWithHeaders(
	ctx context.Context,
	method, path string,
	headers http.Header,
	payload io.Reader,
	response any,
) (http.Header, error) {
	return c.do(ctx, method, path, headers, payload, response)
}

func (c *Client) do(
	ctx context.Context,
	method, path string,
	headers http.Header,
	payload io.Reader,
	response any,
) (http.Header, error) {
	if method == "" {
		return nil, ErrEmptyMethod
	}

	pathURL, err := url.Parse(path)
	if err != nil {
		return nil, newInternalError("do", fmt.Errorf("failed to parse path: %w", err))
	}
	if pathURL.User != nil || !sameOrigin(c.baseURL, pathURL) {
		return nil, newInternalError("do", fmt.Errorf("%w: %q", ErrCrossOriginPath, path))
	}

	fullURL := joinBaseURL(c.baseURL, pathURL, path)

	req, err := http.NewRequestWithContext(ctx, method, fullURL, payload)
	if err != nil {
		return nil, newInternalError("do", fmt.Errorf("failed to create request: %w", err))
	}

	req.Header = headers

	resp, err := c.client.Do(req)
	if err != nil {
		if errors.Is(err, ErrCrossOriginPath) {
			return nil, newInternalError("do", err)
		}

		return nil, newInfrastructureError(fullURL, err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode >= http.StatusMultipleChoices {
		const maxErrBody = 1 << 20 // 1 MiB
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrBody))

		return resp.Header, c.formatError(resp.StatusCode, body, fullURL, resp.Header)
	}

	if resp.StatusCode == http.StatusNoContent {
		return resp.Header, nil
	}

	if response == nil {
		return resp.Header, nil
	}

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&response); err != nil {
		if errors.Is(err, io.EOF) && decoder.InputOffset() == 0 {
			return resp.Header, nil
		}

		return resp.Header, newInternalError("do", fmt.Errorf("failed to decode response: %w", err))
	}

	return resp.Header, nil
}

func joinBaseURL(base *url.URL, ref *url.URL, path string) string {
	if ref.Host != "" {
		if ref.Scheme == "" {
			return base.Scheme + ":" + path
		}

		return path
	}

	root := *base
	root.RawQuery = ""
	root.Fragment = ""
	root.RawFragment = ""

	full := strings.TrimRight(root.String(), "/")
	if escaped := ref.EscapedPath(); escaped != "" {
		full += "/" + strings.TrimLeft(escaped, "/")
	}

	query := base.RawQuery
	if ref.RawQuery != "" || ref.ForceQuery {
		query = ref.RawQuery
	}
	if query != "" {
		full += "?" + query
	}

	fragment := base.Fragment
	if ref.Fragment != "" {
		fragment = ref.Fragment
	}
	if fragment != "" {
		full += "#" + fragment
	}

	return full
}

func sameOrigin(base, ref *url.URL) bool {
	if ref.Host == "" {
		return true
	}
	if base.Host == "" {
		return ref.Scheme != ""
	}
	if ref.Scheme != "" && ref.Scheme != base.Scheme {
		return false
	}

	return ref.Host == base.Host
}

type originGuard struct {
	base *url.URL
	next http.RoundTripper
}

func (g originGuard) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.User != nil || !sameOrigin(g.base, req.URL) {
		return nil, fmt.Errorf("%w: %s", ErrCrossOriginPath, req.URL.Redacted())
	}
	next := g.next
	if next == nil {
		next = http.DefaultTransport
	}

	resp, err := next.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("round trip: %w", err)
	}

	return resp, nil
}

func (c *Client) formatError(statusCode int, body []byte, reqURL string, headers http.Header) error {
	return &APIError{
		StatusCode: statusCode,
		URL:        reqURL,
		Body:       body,
		Headers:    headers,
	}
}

func NewClient(config Config) (*Client, error) {
	// Parse the base URL
	baseURL, err := url.Parse(config.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}

	if config.BaseURL != "" && baseURL.Scheme == "" {
		return nil, fmt.Errorf("%w: base URL must be absolute (got %q)", ErrInvalidConfig, config.BaseURL)
	}

	source := config.Client
	if source == nil {
		source = http.DefaultClient
	}

	client := &http.Client{
		Transport:     originGuard{base: baseURL, next: source.Transport},
		CheckRedirect: source.CheckRedirect,
		Jar:           source.Jar,
		Timeout:       source.Timeout,
	}

	return &Client{
		client:  client,
		baseURL: baseURL,
	}, nil
}
