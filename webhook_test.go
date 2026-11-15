package vaartix

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"
)

// Test-only secret: the worked example from the Standard Webhooks specification.
const prefix = "whsec_"

var specSecret = prefix + "MfKQ9r8GKYqrTwjUPD8ILPZIo2LaLaSw"

func specHeaders() http.Header {
	h := http.Header{}
	h.Set("webhook-id", "msg_p5jXN8AQM9LWM0D4loKWxJek")
	h.Set("webhook-timestamp", "1614265330")
	h.Set("webhook-signature", "v1,g0hM9SsE+OTPJTGt/tmIKtSyZlE3uFJELVlNIOLJ1OE=")
	return h
}

var specNow = time.Unix(1614265330, 0)

func TestVerifyStandardWebhooksExample(t *testing.T) {
	if _, err := verifyAt([]byte(`{"test": 2432232314}`), specHeaders(), specSecret, DefaultWebhookTolerance, specNow); err != nil {
		t.Fatal(err)
	}
	h := specHeaders()
	h.Set("webhook-signature", "v1,bm90LXRoaXM= "+h.Get("webhook-signature"))
	if _, err := verifyAt([]byte(`{"test": 2432232314}`), h, specSecret, DefaultWebhookTolerance, specNow); err != nil {
		t.Fatal("one of several signatures should be enough:", err)
	}
}

func TestVerifyRejects(t *testing.T) {
	cases := map[string]func() error{
		"changed body": func() error {
			_, err := verifyAt([]byte(`{"test": 1}`), specHeaders(), specSecret, DefaultWebhookTolerance, specNow)
			return err
		},
		"wrong secret": func() error {
			_, err := verifyAt([]byte(`{"test": 2432232314}`), specHeaders(), prefix+"c2VjcmV0", DefaultWebhookTolerance, specNow)
			return err
		},
		"stale": func() error {
			_, err := verifyAt([]byte(`{"test": 2432232314}`), specHeaders(), specSecret, DefaultWebhookTolerance, specNow.Add(10*time.Minute))
			return err
		},
		"missing headers": func() error {
			_, err := verifyAt([]byte(`{}`), http.Header{}, specSecret, DefaultWebhookTolerance, specNow)
			return err
		},
		"bad secret prefix": func() error {
			_, err := verifyAt([]byte(`{"test": 2432232314}`), specHeaders(), "nope", DefaultWebhookTolerance, specNow)
			return err
		},
	}
	for name, f := range cases {
		if err := f(); !errors.Is(err, ErrWebhookVerification) {
			t.Errorf("%s: expected ErrWebhookVerification, got %v", name, err)
		}
	}
}

func TestVerifyClickedPayload(t *testing.T) {
	body := []byte(`{"type":"email.clicked","createdAt":"2026-10-28T10:05:41.000Z","data":{"emailId":"em_1","eventId":"evt_1","recipient":"rahul@example.in","from":"Acme <orders@acme.in>","to":["rahul@example.in"],"cc":[],"bcc":[],"subject":"Shipped","tags":[],"emailCreatedAt":"x","details":{"machine":false,"machineReason":null,"linkIndex":0,"link":"https://acme.in/orders/123","userAgentFamily":"Chrome","country":"IN","asn":24560}}}`)
	secret := prefix + base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	key, _ := base64.StdEncoding.DecodeString(secret[len(prefix):])
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte("whd_1.1614265330."))
	mac.Write(body)
	h := http.Header{}
	h.Set("webhook-id", "whd_1")
	h.Set("webhook-timestamp", "1614265330")
	h.Set("webhook-signature", "v1,"+base64.StdEncoding.EncodeToString(mac.Sum(nil)))
	p, err := verifyAt(body, h, secret, DefaultWebhookTolerance, specNow)
	if err != nil || p.Type != EventClicked {
		t.Fatal(err)
	}
	var d ClickedDetails
	if err := json.Unmarshal(p.Data.Details, &d); err != nil || d.Link != "https://acme.in/orders/123" || d.Machine || *d.ASN != 24560 {
		t.Fatalf("details %+v %v", d, err)
	}
	var delayed DelayedDetails
	_ = json.Unmarshal([]byte(`{"delayType":"TransientCommunicationFailure","diagnosticCode":"421 4.7.0 [TSS04]","statusCode":"4.7.0"}`), &delayed)
	if delayed.BounceType != "" || delayed.StatusCode != "4.7.0" {
		t.Fatal("delayed")
	}
}
