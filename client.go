package restkit

import (
	"bytes"
	"context"
	"encoding/json"
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
	config Config
}

func (c *Client) Do(ctx context.Context, method, path string, headers map[string]string, payload, response any) error {
	var reqBody io.Reader
	if payload != nil {
		jsonBytes, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}
		reqBody = bytes.NewReader(jsonBytes)
	}

	if headers == nil {
		headers = make(map[string]string)
	}
	if _, ok := headers["Accept"]; !ok {
		headers["Accept"] = "application/json"
	}
	if reqBody != nil {
		headers["Content-Type"] = "application/json"
	}

	return c.DoRAW(ctx, method, path, headers, reqBody, response)
}

func (c *Client) DoRAW(
	ctx context.Context,
	method, path string,
	headers map[string]string,
	payload io.Reader,
	response any,
) error {
	base := strings.TrimRight(c.config.BaseURL, "/")
	fullURL, err := url.JoinPath(base, path)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, payload)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.config.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)

		return c.formatError(resp.StatusCode, body)
	}

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	if response == nil {
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

func (c *Client) formatError(statusCode int, body []byte) error {
	switch statusCode {
	case http.StatusBadRequest:
		return fmt.Errorf("%w: %s", ErrBadRequest, string(body))
	case http.StatusConflict:
		return fmt.Errorf("%w: %s", ErrConflict, string(body))
	}

	if statusCode >= http.StatusInternalServerError {
		return fmt.Errorf("%w: unexpected status code %d with body %s", ErrServer, statusCode, string(body))
	}

	// All other client errors (400-499)
	return fmt.Errorf("%w: unexpected status code %d with body %s", ErrClient, statusCode, string(body))
}

func NewClient(config Config) *Client {
	if config.Client == nil {
		config.Client = http.DefaultClient
	}

	return &Client{config: config}
}
