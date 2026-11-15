package vaartix

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
)

// EmailsService sends and reads emails.
type EmailsService struct{ c *Client }

// Send sends one email. Pass RequestOptions with an IdempotencyKey to make retries safe.
func (s *EmailsService) Send(ctx context.Context, email *SendEmailRequest, opts ...*RequestOptions) (*SendEmailResponse, error) {
	out := &SendEmailResponse{}
	return out, s.c.do(ctx, http.MethodPost, "/emails", email, out, first(opts))
}

// Get returns one email with its events. Needs a full-access key.
func (s *EmailsService) Get(ctx context.Context, id string) (*Email, error) {
	out := &Email{}
	return out, s.c.do(ctx, http.MethodGet, "/emails/"+esc(id), nil, out, nil)
}

// List returns recent emails, newest first. Needs a full-access key.
func (s *EmailsService) List(ctx context.Context, req *ListEmailsRequest) (*ListEmailsResponse, error) {
	q := url.Values{}
	if req != nil && req.Limit > 0 {
		q.Set("limit", strconv.Itoa(req.Limit))
	}
	if req != nil && req.Before != "" {
		q.Set("before", req.Before)
	}
	path := "/emails"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	out := &ListEmailsResponse{}
	return out, s.c.do(ctx, http.MethodGet, path, nil, out, nil)
}

// Cancel cancels a scheduled email before it is sent. Needs a full-access key.
func (s *EmailsService) Cancel(ctx context.Context, id string) (*EmailSummary, error) {
	out := &EmailSummary{}
	return out, s.c.do(ctx, http.MethodPost, "/emails/"+esc(id)+"/cancel", nil, out, nil)
}

// BatchService sends many emails in one request.
type BatchService struct{ c *Client }

// Send sends up to 100 emails (no attachments or scheduling). All or nothing.
func (s *BatchService) Send(ctx context.Context, emails []*SendEmailRequest) ([]SendEmailResponse, error) {
	var out struct {
		Data []SendEmailResponse `json:"data"`
	}
	if err := s.c.do(ctx, http.MethodPost, "/emails/batch", emails, &out, nil); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// DomainsService manages sending domains. Changes need a full-access key from an owner or admin.
type DomainsService struct{ c *Client }

// Update changes a domain's default open and click tracking.
func (s *DomainsService) Update(ctx context.Context, id string, req *UpdateDomainRequest) (*Domain, error) {
	var raw json.RawMessage
	if err := s.c.do(ctx, http.MethodPatch, "/domains/"+esc(id), req, &raw, nil); err != nil {
		return nil, err
	}
	d := &Domain{Raw: raw}
	if err := json.Unmarshal(raw, d); err != nil {
		return nil, err
	}
	return d, nil
}

// WebhooksService manages webhook endpoints. Needs a full-access key.
type WebhooksService struct{ c *Client }

// Create creates an endpoint. The response includes the signing secret, shown only this once.
func (s *WebhooksService) Create(ctx context.Context, req *CreateWebhookRequest) (*Webhook, error) {
	out := &Webhook{}
	return out, s.c.do(ctx, http.MethodPost, "/webhooks", req, out, nil)
}

// List returns all endpoints.
func (s *WebhooksService) List(ctx context.Context) ([]Webhook, error) {
	var out struct {
		Data []Webhook `json:"data"`
	}
	if err := s.c.do(ctx, http.MethodGet, "/webhooks", nil, &out, nil); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// Get returns one endpoint.
func (s *WebhooksService) Get(ctx context.Context, id string) (*Webhook, error) {
	out := &Webhook{}
	return out, s.c.do(ctx, http.MethodGet, "/webhooks/"+esc(id), nil, out, nil)
}

// Update changes the URL or events, or turns the endpoint off ("disabled") and on ("enabled").
func (s *WebhooksService) Update(ctx context.Context, id string, req *UpdateWebhookRequest) (*Webhook, error) {
	out := &Webhook{}
	return out, s.c.do(ctx, http.MethodPatch, "/webhooks/"+esc(id), req, out, nil)
}

// Remove deletes an endpoint.
func (s *WebhooksService) Remove(ctx context.Context, id string) error {
	return s.c.do(ctx, http.MethodDelete, "/webhooks/"+esc(id), nil, nil, nil)
}

// RotateSecret replaces the signing secret. The old one stops working immediately.
func (s *WebhooksService) RotateSecret(ctx context.Context, id string) (*Webhook, error) {
	out := &Webhook{}
	return out, s.c.do(ctx, http.MethodPost, "/webhooks/"+esc(id)+"/rotate-secret", nil, out, nil)
}

// Deliveries returns recent deliveries to an endpoint, newest first. limit 0 uses the default.
func (s *WebhooksService) Deliveries(ctx context.Context, id string, limit int) ([]WebhookDelivery, error) {
	path := "/webhooks/" + esc(id) + "/deliveries"
	if limit > 0 {
		path += "?limit=" + strconv.Itoa(limit)
	}
	var out struct {
		Data []WebhookDelivery `json:"data"`
	}
	if err := s.c.do(ctx, http.MethodGet, path, nil, &out, nil); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// RetryDelivery sends a delivery again, with the same webhook-id.
func (s *WebhooksService) RetryDelivery(ctx context.Context, id, deliveryID string) (*WebhookDelivery, error) {
	out := &WebhookDelivery{}
	return out, s.c.do(ctx, http.MethodPost, "/webhooks/"+esc(id)+"/deliveries/"+esc(deliveryID)+"/retry", nil, out, nil)
}

func first(o []*RequestOptions) *RequestOptions {
	if len(o) > 0 {
		return o[0]
	}
	return nil
}
