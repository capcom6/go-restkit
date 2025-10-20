package restkit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Config struct {
	Client  *http.Client // Optional HTTP Client, defaults to `http.DefaultClient`
	BaseURL string       // Optional base URL
}

type Client struct {
	client  *http.Client
	baseURL *url.URL
}

func (c *Client) Do(ctx context.Context, method, path string, headers http.Header, payload, response any) error {
	var reqBody io.Reader
	if payload != nil {
		jsonBytes, err := json.Marshal(payload)
		if err != nil {
			return newInternalError("Do", fmt.Errorf("failed to marshal payload: %w", err))
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

	return c.DoRAW(ctx, method, path, headers, reqBody, response)
}

func (c *Client) DoRAW(
	ctx context.Context,
	method, path string,
	headers http.Header,
	payload io.Reader,
	response any,
) error {
	if method == "" {
		return ErrEmptyMethod
	}

	// Parse the path (this preserves query parameters)
	pathURL, err := url.Parse(path)
	if err != nil {
		return newInternalError("DoRAW", fmt.Errorf("failed to parse path: %w", err))
	}

	// Resolve the path against the base URL to get a properly encoded full URL
	fullURL := c.baseURL.ResolveReference(pathURL).String()

	req, err := http.NewRequestWithContext(ctx, method, fullURL, payload)
	if err != nil {
		return newInternalError("DoRAW", fmt.Errorf("failed to create request: %w", err))
	}

	req.Header = headers

	resp, err := c.client.Do(req)
	if err != nil {
		return newInfrastructureError(fullURL, err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode >= http.StatusBadRequest {
		const maxErrBody = 1 << 20 // 1 MiB
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrBody))

		return c.formatError(resp.StatusCode, body, fullURL)
	}

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	if response == nil {
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return newInternalError("DoRAW", fmt.Errorf("failed to decode response: %w", err))
	}

	return nil
}

func (c *Client) formatError(statusCode int, body []byte, reqURL string) error {
	return &APIError{
		StatusCode: statusCode,
		URL:        reqURL,
		Body:       body,
	}
}

func NewClient(config Config) (*Client, error) {
	if config.Client == nil {
		config.Client = http.DefaultClient
	}

	// Parse the base URL
	baseURL, err := url.Parse(config.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}

	if config.BaseURL != "" && baseURL.Scheme == "" {
		return nil, fmt.Errorf("%w: base URL must be absolute (got %q)", ErrInvalidConfig, config.BaseURL)
	}

	return &Client{
		client:  config.Client,
		baseURL: baseURL,
	}, nil
}
