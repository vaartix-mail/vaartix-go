# Vaartix Go SDK

The official Go client for [Vaartix](https://vaartix.com), the transactional email API built in India.
Data stored in India (AWS Mumbai), pricing in ₹.

```sh
go get github.com/vaartix-mail/vaartix-go
```

Go 1.22 or newer. No dependencies outside the standard library.

## Send an email

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/vaartix-mail/vaartix-go"
)

func main() {
	client, err := vaartix.NewClient("") // reads VAARTIX_API_KEY
	if err != nil {
		log.Fatal(err)
	}
	sent, err := client.Emails.Send(context.Background(), &vaartix.SendEmailRequest{
		From:    "Acme <orders@acme.in>", // an address on a domain verified in Vaartix
		To:      []string{"priya@example.com"},
		Subject: "Your order is confirmed",
		HTML:    "<p>Thanks for your order!</p>",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(sent.ID) // em_...
}
```

Testing? Send to `delivered@test.vaartix.com` (or `bounced@` / `complained@`). Nothing real is sent.

## Errors

API errors are `*vaartix.Error`, with the API's `Type`, `Message` and `StatusCode`:

```go
var apiErr *vaartix.Error
if errors.As(err, &apiErr) && apiErr.Type == "validation_error" {
	// fix the request
}
```

`Type` is `network_error` (and `StatusCode` is 0) when no response was received.

## More sending options

```go
// Safe retries: the same key within 24 hours returns the first result.
client.Emails.Send(ctx, email, &vaartix.RequestOptions{IdempotencyKey: "receipt-10482"})

// Send later
email.ScheduledAt = "2026-11-15T10:00:00+05:30"

// Up to 100 emails in one request
client.Batch.Send(ctx, []*vaartix.SendEmailRequest{a, b})

// Open and click tracking for one email (overrides the domain's defaults)
email.Tracking = &vaartix.TrackingSettings{Opens: vaartix.Bool(true), Clicks: vaartix.Bool(false)}
```

Keep click tracking off for login codes, password resets and magic links.

## Read emails

Needs a full-access key.

```go
email, err := client.Emails.Get(ctx, "em_...")            // content, status and events
page, err := client.Emails.List(ctx, &vaartix.ListEmailsRequest{Limit: 50})
_, err = client.Emails.Cancel(ctx, "em_...")               // a scheduled email
```

## Webhooks

Check that a webhook request really came from Vaartix before you trust it. Use the **raw** body.

```go
http.HandleFunc("/webhooks/vaartix", func(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	event, err := vaartix.VerifyWebhook(body, r.Header, os.Getenv("VAARTIX_WEBHOOK_SECRET"))
	if err != nil {
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return
	}
	switch event.Type {
	case vaartix.EventBounced:
		var d vaartix.BounceDetails
		json.Unmarshal(event.Data.Details, &d) // d.DiagnosticCode: the receiving server's reply
	case vaartix.EventClicked:
		var d vaartix.ClickedDetails
		json.Unmarshal(event.Data.Details, &d) // d.Machine is true for link scanners
	}
	w.WriteHeader(http.StatusOK)
})
```

Signatures follow the [Standard Webhooks](https://www.standardwebhooks.com) spec. Manage endpoints with
`client.Webhooks.Create`, `List`, `Get`, `Update`, `Remove`, `RotateSecret`, `Deliveries` and `RetryDelivery`.

## Domains

```go
client.Domains.Update(ctx, domainID, &vaartix.UpdateDomainRequest{OpenTracking: vaartix.Bool(true)})
```

## Docs

- Docs: https://vaartix.com/docs
- API reference: https://vaartix.com/docs/api
- Status: https://status.vaartix.com

## Contributing and security

Issues and pull requests are welcome. Report security problems privately, see [SECURITY.md](SECURITY.md).

## License

MIT
