package vaartix

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type captured struct {
	method, path, auth, ua, idem, ctype string
	body                                map[string]any
	list                                []any
}

func server(t *testing.T, status int, resp string) (*Client, *captured) {
	t.Helper()
	c := &captured{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.method, c.path = r.Method, r.URL.RequestURI()
		c.auth, c.ua, c.idem, c.ctype = r.Header.Get("Authorization"), r.Header.Get("User-Agent"), r.Header.Get("Idempotency-Key"), r.Header.Get("Content-Type")
		c.body, c.list = nil, nil
		b, _ := io.ReadAll(r.Body)
		if len(b) > 0 && b[0] == '[' {
			_ = json.Unmarshal(b, &c.list)
		} else if len(b) > 0 {
			_ = json.Unmarshal(b, &c.body)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(resp))
	}))
	t.Cleanup(srv.Close)
	client, err := NewClient("vx_test", WithBaseURL(srv.URL+"/"))
	if err != nil {
		t.Fatal(err)
	}
	return client, c
}

func TestNewClientNeedsKey(t *testing.T) {
	t.Setenv("VAARTIX_API_KEY", "")
	if _, err := NewClient(""); err == nil {
		t.Fatal("expected an error without a key")
	}
	t.Setenv("VAARTIX_API_KEY", "vx_env")
	if c, err := NewClient(""); err != nil || c.apiKey != "vx_env" {
		t.Fatal("expected the key from the environment")
	}
}

func TestSendEmail(t *testing.T) {
	client, c := server(t, 200, `{"id":"em_123"}`)
	res, err := client.Emails.Send(context.Background(), &SendEmailRequest{
		From: "Acme <orders@acme.in>", To: []string{"priya@example.com"}, Subject: "Hi", HTML: "<p>Hi</p>",
		ReplyTo: []string{"help@acme.in"}, ScheduledAt: "2026-11-15T10:00:00+05:30",
		Tracking: &TrackingSettings{Opens: Bool(true)},
	}, &RequestOptions{IdempotencyKey: "order-1"})
	if err != nil || res.ID != "em_123" {
		t.Fatalf("got %v %v", res, err)
	}
	if c.method != "POST" || c.path != "/emails" || c.auth != "Bearer vx_test" || c.idem != "order-1" || c.ctype != "application/json" {
		t.Fatalf("bad request %+v", c)
	}
	if c.ua != "vaartix-go/"+Version {
		t.Fatalf("user agent %q", c.ua)
	}
	if c.body["replyTo"] == nil || c.body["scheduledAt"] == nil || c.body["text"] != nil {
		t.Fatalf("bad body %v", c.body)
	}
	tr := c.body["tracking"].(map[string]any)
	if tr["opens"] != true || tr["clicks"] != nil {
		t.Fatalf("tracking %v", tr)
	}
}

func TestErrors(t *testing.T) {
	client, _ := server(t, 422, `{"error":{"type":"validation_error","message":"to: not a valid email address."}}`)
	_, err := client.Emails.Send(context.Background(), &SendEmailRequest{From: "a@acme.in", To: []string{"x"}, Subject: "s", Text: "t"})
	var ve *Error
	if !errors.As(err, &ve) || ve.Type != "validation_error" || ve.StatusCode != 422 {
		t.Fatalf("got %v", err)
	}
	client2, _ := server(t, 500, `oops`)
	_, err = client2.Emails.Get(context.Background(), "em_1")
	if !errors.As(err, &ve) || ve.Type != "api_error" || ve.StatusCode != 500 {
		t.Fatalf("got %v", err)
	}
	dead, _ := NewClient("vx_test", WithBaseURL("http://127.0.0.1:1"))
	_, err = dead.Emails.Get(context.Background(), "em_1")
	if !errors.As(err, &ve) || ve.Type != "network_error" || ve.StatusCode != 0 {
		t.Fatalf("got %v", err)
	}
}

func TestEmailsReadAndList(t *testing.T) {
	client, c := server(t, 200, `{"id":"em_1","status":"delivered","to":["a@b.in"],"tracking":{"opens":true},"events":[{"type":"delivered","recipient":"a@b.in","data":{},"occurredAt":"x"}]}`)
	e, err := client.Emails.Get(context.Background(), "em_1/x")
	if err != nil || e.Status != "delivered" || *e.Tracking.Opens != true || len(e.Events) != 1 || c.path != "/emails/em_1%2Fx" {
		t.Fatalf("got %+v %v %s", e, err, c.path)
	}
	client2, c2 := server(t, 200, `{"data":[{"id":"em_2"}],"hasMore":true}`)
	l, err := client2.Emails.List(context.Background(), &ListEmailsRequest{Limit: 5, Before: "em_9"})
	if err != nil || !l.HasMore || l.Data[0].ID != "em_2" || c2.path != "/emails?before=em_9&limit=5" {
		t.Fatalf("got %+v %v %s", l, err, c2.path)
	}
	client3, c3 := server(t, 200, `{"id":"em_3","status":"canceled"}`)
	s, err := client3.Emails.Cancel(context.Background(), "em_3")
	if err != nil || s.Status != "canceled" || c3.path != "/emails/em_3/cancel" || c3.method != "POST" {
		t.Fatal("cancel")
	}
}

func TestBatchAndDomains(t *testing.T) {
	client, c := server(t, 200, `{"data":[{"id":"em_a"},{"id":"em_b"}]}`)
	out, err := client.Batch.Send(context.Background(), []*SendEmailRequest{
		{From: "a@acme.in", To: []string{"x@y.in"}, Subject: "1", Text: "1"},
		{From: "a@acme.in", To: []string{"z@y.in"}, Subject: "2", Text: "2"},
	})
	if err != nil || len(out) != 2 || c.path != "/emails/batch" || len(c.list) != 2 {
		t.Fatalf("got %v %v", out, err)
	}
	client2, c2 := server(t, 200, `{"id":"dom_1","name":"acme.in","openTracking":true,"clickTracking":false}`)
	d, err := client2.Domains.Update(context.Background(), "dom_1", &UpdateDomainRequest{OpenTracking: Bool(true)})
	if err != nil || !d.OpenTracking || c2.method != "PATCH" || c2.body["clickTracking"] != nil || len(d.Raw) == 0 {
		t.Fatalf("got %+v %v %v", d, err, c2.body)
	}
}

func TestWebhooks(t *testing.T) {
	ctx := context.Background()
	client, c := server(t, 200, `{"id":"wh_1","url":"https://acme.in/h","events":["email.delivered"],"status":"enabled","secret":"whsec_x"}`)
	w, err := client.Webhooks.Create(ctx, &CreateWebhookRequest{URL: "https://acme.in/h", Events: []string{EventDelivered}})
	if err != nil || w.Secret != "whsec_x" || c.path != "/webhooks" {
		t.Fatal("create")
	}
	if _, err := client.Webhooks.Update(ctx, "wh_1", &UpdateWebhookRequest{Status: "disabled"}); err != nil || c.method != "PATCH" || c.body["status"] != "disabled" || c.body["url"] != nil {
		t.Fatal("update")
	}
	if _, err := client.Webhooks.RotateSecret(ctx, "wh_1"); err != nil || c.path != "/webhooks/wh_1/rotate-secret" {
		t.Fatal("rotate")
	}
	if _, err := client.Webhooks.RetryDelivery(ctx, "wh_1", "whd_2"); err != nil || c.path != "/webhooks/wh_1/deliveries/whd_2/retry" {
		t.Fatal("retry")
	}
	client2, c2 := server(t, 200, `{"data":[{"id":"whd_1","status":"succeeded","attempts":1,"payload":{"type":"email.sent"}}]}`)
	ds, err := client2.Webhooks.Deliveries(ctx, "wh_1", 10)
	if err != nil || len(ds) != 1 || c2.path != "/webhooks/wh_1/deliveries?limit=10" {
		t.Fatal("deliveries")
	}
	client3, c3 := server(t, 204, ``)
	if err := client3.Webhooks.Remove(ctx, "wh_1"); err != nil || c3.method != "DELETE" {
		t.Fatal("remove")
	}
}
