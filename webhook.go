package vaartix

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ErrWebhookVerification is wrapped by every error VerifyWebhook returns.
var ErrWebhookVerification = errors.New("vaartix: webhook verification failed")

// DefaultWebhookTolerance is how old a webhook's timestamp may be before it is rejected.
const DefaultWebhookTolerance = 5 * time.Minute

// VerifyWebhook checks that a request really came from Vaartix and returns its parsed body.
// payload must be the raw request body, exactly as received. secret is the endpoint's whsec_ secret.
// It follows the Standard Webhooks spec, so any Standard Webhooks library works too.
func VerifyWebhook(payload []byte, headers http.Header, secret string) (*WebhookPayload, error) {
	return verifyAt(payload, headers, secret, DefaultWebhookTolerance, time.Now())
}

// VerifyWebhookWithTolerance is VerifyWebhook with a custom timestamp tolerance.
func VerifyWebhookWithTolerance(payload []byte, headers http.Header, secret string, tolerance time.Duration) (*WebhookPayload, error) {
	return verifyAt(payload, headers, secret, tolerance, time.Now())
}

func fail(msg string) error { return errors.Join(ErrWebhookVerification, errors.New(msg)) }

func verifyAt(payload []byte, headers http.Header, secret string, tolerance time.Duration, now time.Time) (*WebhookPayload, error) {
	id, ts, sigs := headers.Get("webhook-id"), headers.Get("webhook-timestamp"), headers.Get("webhook-signature")
	if id == "" || ts == "" || sigs == "" {
		return nil, fail("missing webhook-id, webhook-timestamp or webhook-signature header")
	}
	sent, err := strconv.ParseInt(ts, 10, 64)
	if err != nil || math.Abs(float64(now.Unix()-sent)) > tolerance.Seconds() {
		return nil, fail("the webhook timestamp is too old or in the future")
	}
	if !strings.HasPrefix(secret, "whsec_") {
		return nil, fail("the secret should start with whsec_")
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(secret, "whsec_"))
	if err != nil {
		return nil, fail("the secret is not valid base64")
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(id + "." + ts + "."))
	mac.Write(payload)
	expected := mac.Sum(nil)
	ok := false
	// The header can hold several space-separated signatures, e.g. while a secret is rotated.
	for _, entry := range strings.Split(sigs, " ") {
		version, sig, found := strings.Cut(entry, ",")
		if !found || version != "v1" {
			continue
		}
		given, err := base64.StdEncoding.DecodeString(sig)
		if err == nil && hmac.Equal(given, expected) {
			ok = true
			break
		}
	}
	if !ok {
		return nil, fail("the webhook signature does not match")
	}
	out := &WebhookPayload{}
	if err := json.Unmarshal(payload, out); err != nil {
		return nil, fail("the body is not valid JSON")
	}
	return out, nil
}
