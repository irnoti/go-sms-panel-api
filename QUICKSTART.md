# Quickstart (Go)

Goal: send your first SMS with the irnoti Go SDK in about 5 minutes, without spending money.

Full function list: [README → function reference](README.md#what-can-this-sdk-do-function-reference). Full API reference (the docs panel): [irnoti.com/docs](https://irnoti.com/docs).

## 1. Install

```bash
go get github.com/irnoti/go-sms-panel-api
```

Requires Go 1.21+. Standard library only.

## 2. Get an access key

In the irnoti panel create an **access key** (docs panel → *Authentication / Access keys*).

- `irn_test_…` — sandbox key: nothing is sent, nothing is charged. **Start with this.**
- `irn_live_…` — real key: sends real SMS and charges your wallet.

Keep it in an environment variable, never in source control or client-side code:

```bash
export IRNOTI_API_KEY=irn_test_xxxxxxxx
```

## 3. Send a test SMS (sandbox, free)

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	irnoti "github.com/irnoti/go-sms-panel-api"
)

func main() {
	ctx := context.Background()
	c := irnoti.New(os.Getenv("IRNOTI_API_KEY"))

	// How much would it cost? (no send, no charge)
	q, err := c.SMS.Quote(ctx, irnoti.SendParams{LineID: "sandbox", To: []string{"09121234567"}, Text: "Hello"})
	fmt.Println(q, err)

	// Send. IdempotencyKey makes retries safe: the same key never sends twice.
	res, err := c.SMS.Send(ctx, irnoti.SendParams{
		LineID: "sandbox", To: []string{"09121234567"}, Text: "Hello", IdempotencyKey: "my-first-send",
	})
	var apiErr *irnoti.APIError
	if errors.As(err, &apiErr) {
		fmt.Println(apiErr.Status, apiErr.Code, apiErr.Message)
		return
	}
	fmt.Println(res) // BatchID, Count, TotalPrice, Replayed
}
```

`totalPrice` is an integer amount in **Rial, returned as a string** — don't convert it to a float.

## 4. Go live

1. Switch to an `irn_live_…` key.
2. Find your sender number: list your lines and use its `id` instead of `"sandbox"`.
3. Check money: the wallet `balance` function.
4. Always `quote` first for big sends, and always pass an idempotency key to `send`.

```go
lines, _ := c.Lines.List(ctx)
c.SMS.Send(ctx, irnoti.SendParams{LineID: lines[0].ID, To: []string{"09121234567"}, Text: "Hello", IdempotencyKey: "order-1001"})
```

## 5. Common tasks — which function, which docs section

| I want to… | Call | Docs section |
|---|---|---|
| Send one/many SMS | `c.SMS.Send(ctx, irnoti.SendParams{LineID, Text, To})` | SMS |
| Send to a contact group | `c.SMS.Send(ctx, irnoti.SendParams{LineID, Text, GroupIDs})` | SMS + Phonebook |
| Schedule / drip-send | `SendParams{ScheduledAt: &t, Gradual: &irnoti.Gradual{ChunkSize: 500, IntervalMinutes: 10}}` | SMS |
| Send an OTP with an approved template | `c.SMS.Pattern(ctx, "template_code", "0912…", map[string]string{"code": "4821"}, "")` | SMS → Pattern |
| Check delivery | `c.Messages.Get(ctx, id)` or a webhook | SMS → Messages / Webhooks |
| Manage contacts | `c.Phonebook.CreateGroup`, `AddContacts`, `ImportCSV` | Phonebook |
| Check balance | `c.Wallet.Balance(ctx)` | Wallet |
| Phone verification flow | `c.Verify.Create(ctx, irnoti.Object{"to": "0912…", "ttlSec": 300})` | Verify |
| Telegram bot relay | `c.Broker.Create(ctx, label, token)` | Telegram broker |

(Every function: [README reference table](README.md#what-can-this-sdk-do-function-reference).)

## 6. Handle errors

```go
_, err := c.SMS.Send(ctx, irnoti.SendParams{LineID: line, To: []string{"0912…"}, Text: "Hi"})
var apiErr *irnoti.APIError
if errors.As(err, &apiErr) {
	switch {
	case apiErr.IsInsufficientBalance(): // top up the wallet
	case apiErr.IsRateLimited(): // slow down
	case apiErr.IsInvalidRecipients(): fmt.Println(apiErr.Details)
	}
}
```

Status → helper: `401` unauthenticated, `402` insufficient balance, `403` forbidden (missing scope / KYC), `404` not found, `422` `no_price`, `429` rate limited (send limit 600 req/min).

## 7. Receive delivery webhooks

Verify the **raw** request body (not re-serialised JSON) against the `X-Irnoti-Signature` header with your `whsec_…` secret. Signatures older than 5 minutes are rejected; events can repeat, so de-duplicate on `X-Irnoti-Delivery`.

```go
http.HandleFunc("/irnoti/webhook", func(w http.ResponseWriter, r *http.Request) {
	raw, _ := io.ReadAll(r.Body)
	ev, err := irnoti.ParseWebhook(raw, r.Header.Get("X-Irnoti-Signature"), os.Getenv("IRNOTI_WEBHOOK_SECRET"), 0) // 0 = 5 min tolerance
	if err != nil {
		http.Error(w, "bad signature", http.StatusBadRequest)
		return
	}
	log.Println(ev.Event, ev.Data)
	w.WriteHeader(http.StatusNoContent)
})
```

## Next

- Browse everything in the [function reference](README.md#what-can-this-sdk-do-function-reference).
- Exact signatures: `go doc github.com/irnoti/go-sms-panel-api SMS.Send` or [pkg.go.dev](https://pkg.go.dev/github.com/irnoti/go-sms-panel-api).
- Multi-tenant app? `c.WithAPIKey(customerKey)`.
