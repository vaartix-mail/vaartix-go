// Package vaartix is the official Go client for Vaartix, the transactional email API built in India.
//
//	client, err := vaartix.NewClient("") // reads VAARTIX_API_KEY
//	sent, err := client.Emails.Send(ctx, &vaartix.SendEmailRequest{
//		From:    "Acme <orders@acme.in>",
//		To:      []string{"priya@example.com"},
//		Subject: "Your order is confirmed",
//		HTML:    "<p>Thanks for your order!</p>",
//	})
package vaartix

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Version is this SDK's version, sent in the User-Agent header.
const Version = "0.1.0"

// DefaultBaseURL is the Vaartix API.
const DefaultBaseURL = "https://api.vaartix.com"

// Client talks to the Vaartix API. Create one with NewClient and reuse it; it is safe for concurrent use.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client

	Emails   *EmailsService
	Batch    *BatchService
	Domains  *DomainsService
	Webhooks *WebhooksService
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL points the client at another API URL, for example a local mock.
func WithBaseURL(u string) Option {
	return func(c *Client) { c.baseURL = strings.TrimRight(u, "/") }
}

// WithHTTPClient uses your own *http.Client (timeouts, proxies, transport).
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.httpClient = h }
}

// NewClient creates a client. An empty apiKey reads the VAARTIX_API_KEY environment variable.
func NewClient(apiKey string, opts ...Option) (*Client, error) {
	if apiKey == "" {
		apiKey = os.Getenv("VAARTIX_API_KEY")
	}
	if apiKey == "" {
		return nil, errors.New("vaartix: missing API key; pass it to NewClient or set VAARTIX_API_KEY")
	}
	c := &Client{apiKey: apiKey, baseURL: DefaultBaseURL, httpClient: &http.Client{Timeout: 30 * time.Second}}
	for _, o := range opts {
		o(c)
	}
	c.Emails = &EmailsService{c}
	c.Batch = &BatchService{c}
	c.Domains = &DomainsService{c}
	c.Webhooks = &WebhooksService{c}
	return c, nil
}

// Error is returned when the API answers with an error, or the request could not be made.
type Error struct {
	// Type is the API's error type, for example "validation_error", or "network_error".
	Type    string `json:"type"`
	Message string `json:"message"`
	// StatusCode is the HTTP status, or 0 when no response was received.
	StatusCode int `json:"-"`
}

func (e *Error) Error() string {
	if e.StatusCode == 0 {
		return fmt.Sprintf("vaartix: %s: %s", e.Type, e.Message)
	}
	return fmt.Sprintf("vaartix: %s (%d): %s", e.Type, e.StatusCode, e.Message)
}

// RequestOptions are per-request settings.
type RequestOptions struct {
	// IdempotencyKey: retrying a request with the same key within 24 hours returns the first result.
	IdempotencyKey string
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any, opts *RequestOptions) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("vaartix: encode request: %w", err)
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("vaartix: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("User-Agent", "vaartix-go/"+Version)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if opts != nil && opts.IdempotencyKey != "" {
		req.Header.Set("Idempotency-Key", opts.IdempotencyKey)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &Error{Type: "network_error", Message: err.Error()}
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return &Error{Type: "network_error", Message: err.Error(), StatusCode: resp.StatusCode}
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		var payload struct {
			Error *Error `json:"error"`
		}
		if json.Unmarshal(data, &payload) == nil && payload.Error != nil && payload.Error.Message != "" {
			payload.Error.StatusCode = resp.StatusCode
			if payload.Error.Type == "" {
				payload.Error.Type = "api_error"
			}
			return payload.Error
		}
		return &Error{Type: "api_error", Message: fmt.Sprintf("request failed with status %d", resp.StatusCode), StatusCode: resp.StatusCode}
	}
	if out == nil || len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("vaartix: decode response: %w", err)
	}
	return nil
}

func esc(s string) string { return url.PathEscape(s) }
