package vaartix

import "encoding/json"

// Tag is a name/value label on an email, returned in events and webhooks.
type Tag struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Attachment is a file sent with an email.
type Attachment struct {
	Filename string `json:"filename"`
	// Content is the base64-encoded file content.
	Content string `json:"content"`
	// ContentType is detected from the filename if empty.
	ContentType string `json:"contentType,omitempty"`
	// ContentID is for inline images: reference it in your HTML as cid:<ContentID>.
	ContentID string `json:"contentId,omitempty"`
}

// TrackingSettings turns open and click tracking on or off for one email, overriding the domain's
// defaults. A nil field uses the domain's setting.
type TrackingSettings struct {
	Opens  *bool `json:"opens,omitempty"`
	Clicks *bool `json:"clicks,omitempty"`
}

// Bool returns a pointer to b, for TrackingSettings and UpdateDomainRequest.
func Bool(b bool) *bool { return &b }

// SendEmailRequest is one email. HTML or Text (or both) is required.
type SendEmailRequest struct {
	From        string            `json:"from"`
	To          []string          `json:"to"`
	Subject     string            `json:"subject"`
	HTML        string            `json:"html,omitempty"`
	Text        string            `json:"text,omitempty"`
	CC          []string          `json:"cc,omitempty"`
	BCC         []string          `json:"bcc,omitempty"`
	ReplyTo     []string          `json:"replyTo,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Tags        []Tag             `json:"tags,omitempty"`
	Attachments []Attachment      `json:"attachments,omitempty"`
	// ScheduledAt is an ISO 8601 date-time to send later, e.g. "2026-11-01T09:00:00+05:30".
	ScheduledAt string            `json:"scheduledAt,omitempty"`
	Tracking    *TrackingSettings `json:"tracking,omitempty"`
}

// SendEmailResponse holds the new email's ID.
type SendEmailResponse struct {
	ID string `json:"id"`
}

// EmailEvent is one step in an email's timeline.
type EmailEvent struct {
	Type       string          `json:"type"`
	Recipient  *string         `json:"recipient"`
	Data       json.RawMessage `json:"data"`
	OccurredAt string          `json:"occurredAt"`
}

// AttachmentInfo describes an attachment on a sent email.
type AttachmentInfo struct {
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
}

// EmailSummary is an email as listed.
type EmailSummary struct {
	ID          string           `json:"id"`
	From        string           `json:"from"`
	To          []string         `json:"to"`
	CC          []string         `json:"cc"`
	BCC         []string         `json:"bcc"`
	ReplyTo     []string         `json:"replyTo"`
	Subject     string           `json:"subject"`
	Status      string           `json:"status"`
	Tags        []Tag            `json:"tags"`
	Attachments []AttachmentInfo `json:"attachments"`
	ScheduledAt *string          `json:"scheduledAt"`
	SentAt      *string          `json:"sentAt"`
	LastEventAt *string          `json:"lastEventAt"`
	Error       *string          `json:"error"`
	CreatedAt   string           `json:"createdAt"`
}

// Email is one email with its content and events.
type Email struct {
	EmailSummary
	HTML    *string           `json:"html"`
	Text    *string           `json:"text"`
	Headers map[string]string `json:"headers"`
	// Tracking is the email's own setting, or nil when it uses the domain's defaults.
	Tracking *TrackingSettings `json:"tracking"`
	Events   []EmailEvent      `json:"events"`
}

// ListEmailsRequest pages through emails, newest first.
type ListEmailsRequest struct {
	// Limit is 1 to 100 (default 20).
	Limit int
	// Before returns emails older than this email ID.
	Before string
}

// ListEmailsResponse is one page of emails.
type ListEmailsResponse struct {
	Data    []EmailSummary `json:"data"`
	HasMore bool           `json:"hasMore"`
}

// Domain is a sending domain. Only the tracking fields are typed; Raw holds the full JSON.
type Domain struct {
	ID            string          `json:"id"`
	OpenTracking  bool            `json:"openTracking"`
	ClickTracking bool            `json:"clickTracking"`
	Raw           json.RawMessage `json:"-"`
}

// UpdateDomainRequest changes a domain's tracking defaults. Nil fields keep their value.
type UpdateDomainRequest struct {
	OpenTracking  *bool `json:"openTracking,omitempty"`
	ClickTracking *bool `json:"clickTracking,omitempty"`
}

// Webhook is a webhook endpoint.
type Webhook struct {
	ID        string   `json:"id"`
	URL       string   `json:"url"`
	Events    []string `json:"events"`
	Status    string   `json:"status"`
	CreatedAt string   `json:"createdAt"`
	// Secret (whsec_...) is only returned when the endpoint is created or its secret rotated.
	Secret string `json:"secret,omitempty"`
}

// CreateWebhookRequest creates an endpoint for a public https:// URL.
type CreateWebhookRequest struct {
	URL    string   `json:"url"`
	Events []string `json:"events"`
}

// UpdateWebhookRequest changes an endpoint. Empty fields keep their value; Status is "enabled" or "disabled".
type UpdateWebhookRequest struct {
	URL    string   `json:"url,omitempty"`
	Events []string `json:"events,omitempty"`
	Status string   `json:"status,omitempty"`
}

// WebhookDelivery is one request Vaartix made to your endpoint.
type WebhookDelivery struct {
	ID             string          `json:"id"`
	Type           string          `json:"type"`
	EventID        *string         `json:"eventId"`
	Status         string          `json:"status"`
	Attempts       int             `json:"attempts"`
	ResponseStatus *int            `json:"responseStatus"`
	ResponseBody   *string         `json:"responseBody"`
	NextAttemptAt  *string         `json:"nextAttemptAt"`
	DeliveredAt    *string         `json:"deliveredAt"`
	CreatedAt      string          `json:"createdAt"`
	Payload        json.RawMessage `json:"payload"`
}

// Webhook event types.
const (
	EventSent             = "email.sent"
	EventDelivered        = "email.delivered"
	EventDeliveryDelayed  = "email.delivery_delayed"
	EventBounced          = "email.bounced"
	EventComplained       = "email.complained"
	EventRejected         = "email.rejected"
	EventRenderingFailure = "email.rendering_failure"
	EventSuppressed       = "email.suppressed"
	EventOpened           = "email.opened"
	EventClicked          = "email.clicked"
)

// WebhookPayload is the JSON body Vaartix posts to your endpoint. Decode Data.Details with the
// matching details type (OpenedDetails, ClickedDetails, DelayedDetails, BounceDetails) for the event.
type WebhookPayload struct {
	Type string `json:"type"`
	// CreatedAt is when the event happened. Deliveries can arrive out of order; sort by this.
	CreatedAt string `json:"createdAt"`
	Data      struct {
		EmailID string `json:"emailId"`
		EventID string `json:"eventId"`
		// Recipient is nil when the email had more than one recipient.
		Recipient      *string         `json:"recipient"`
		From           string          `json:"from"`
		To             []string        `json:"to"`
		CC             []string        `json:"cc"`
		BCC            []string        `json:"bcc"`
		Subject        string          `json:"subject"`
		Tags           []Tag           `json:"tags"`
		EmailCreatedAt string          `json:"emailCreatedAt"`
		Details        json.RawMessage `json:"details"`
	} `json:"data"`
}

// OpenedDetails is Details of an email.opened event.
type OpenedDetails struct {
	Machine         bool    `json:"machine"`
	MachineReason   *string `json:"machineReason"`
	UserAgentFamily string  `json:"userAgentFamily"`
	Country         *string `json:"country"`
	ASN             *int    `json:"asn"`
}

// ClickedDetails is Details of an email.clicked event.
type ClickedDetails struct {
	OpenedDetails
	LinkIndex int    `json:"linkIndex"`
	Link      string `json:"link"`
}

// DelayedDetails is Details of an email.delivery_delayed event. A soft bounce also arrives as
// email.delivery_delayed with BounceType set: decode into BounceDetails when it is.
type DelayedDetails struct {
	DelayType      string `json:"delayType,omitempty"`
	ExpirationTime string `json:"expirationTime,omitempty"`
	// DiagnosticCode is the receiving server's exact reply, e.g. Yahoo's "421 4.7.0 [TSS04] ..." line.
	DiagnosticCode string `json:"diagnosticCode,omitempty"`
	StatusCode     string `json:"statusCode,omitempty"`
	BounceType     string `json:"bounceType,omitempty"`
}

// BounceDetails is Details of an email.bounced event.
type BounceDetails struct {
	BounceType    string `json:"bounceType"`
	BounceSubType string `json:"bounceSubType,omitempty"`
	// Deprecated: use DiagnosticCode.
	Diagnostics    []string `json:"diagnostics"`
	DiagnosticCode string   `json:"diagnosticCode,omitempty"`
	StatusCode     string   `json:"statusCode,omitempty"`
}
