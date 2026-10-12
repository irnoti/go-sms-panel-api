# irnoti (Go)

[![Go Reference](https://pkg.go.dev/badge/github.com/irnoti/go-sms-panel-api.svg)](https://pkg.go.dev/github.com/irnoti/go-sms-panel-api) [![tests](https://github.com/irnoti/go-sms-panel-api/actions/workflows/tests.yml/badge.svg)](https://github.com/irnoti/go-sms-panel-api/actions/workflows/tests.yml) [![license](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE.md)

Go SDK for the irnoti API — standard library only.

```bash
go get github.com/irnoti/go-sms-panel-api
```

```go
c := irnoti.New(os.Getenv("IRNOTI_API_KEY")) // WithBaseURL, WithHTTPClient, WithRetries are optional
res, err := c.SMS.Send(ctx, irnoti.SendParams{LineID: "1", To: []string{"09121234567"}, Text: "Hi", IdempotencyKey: "k1"})
var apiErr *irnoti.APIError
if errors.As(err, &apiErr) && apiErr.IsInsufficientBalance() { /* top up */ }

// net/http handler: verify against the raw body
raw, _ := io.ReadAll(r.Body)
ev, err := irnoti.ParseWebhook(raw, r.Header.Get("X-Irnoti-Signature"), secret, 0)
```

The module path is `github.com/irnoti/go-sms-panel-api`.

### Telegram broker

```go
bot, _ := c.Broker.Create(ctx, "Shop bot", botFatherToken) // scope broker.write (account owner); returns the relay "link"
c.Broker.Request(ctx, bot["id"].(string), "sendMessage", irnoti.Object{"chat_id": 123, "text": "Hi"}) // scope broker.request
c.Broker.Updates(ctx, bot["id"].(string), irnoti.UpdatesParams{Limit: 20})                            // pull mode
c.Broker.SetWebhook(ctx, bot["id"].(string), irnoti.WebhookParams{URL: "https://example.com/tg", SecretToken: "s3cret"}) // push mode
```

## Behaviour shared by all irnoti SDKs

- **Auth:** `Authorization: Bearer irn_live_…`. Use an `irn_test_…` key with `lineId: "sandbox"` to try sends for free. Keys carry scopes (`sms.send`, `sms.read`, `wallet.read`, `phonebook.read|write`, `bot.send|read`, `broker.read|write|request`, `verify.send|read`); a missing scope is a `403`.
- **Money** is integer **Rial** returned as a **string** (`"1960"`) — never parse it into a float.
- **Idempotency:** every send takes an idempotency key; retrying with the same key never sends twice (`replayed: true`). Automatic retries apply only to `GET`s and sends that carry a key.
- **Errors:** one error type carrying `status`, the API's machine `code` (`insufficient_balance`, `invalid_recipients`, `no_price`, … — empty for plain 401/403/404) and `details`, with helpers such as `isInsufficientBalance` / `isRateLimited` / `isUnauthenticated`.
- **Limits:** send 600 req/min, pattern 1200 req/min per IP (`429` when exceeded).
- **Webhooks:** verify the **raw** body against `X-Irnoti-Signature: t=<unix>,v1=<hex>` (HMAC-SHA256 of `"<t>.<body>"` with your `whsec_…` key); signatures older than 5 minutes are rejected.
- Bot-builder creation/configuration and verify settings are panel-only; Telegram broker bots can be managed through the API.

Sibling SDKs: [Laravel](https://github.com/irnoti/laravel-sms-panel-api) · [Node.js](https://github.com/irnoti/node-sms-panel-api) · [Python](https://github.com/irnoti/python-sms-panel-api) · [Go](https://github.com/irnoti/go-sms-panel-api) · [PHP](https://github.com/irnoti/php-sms-panel-api). API reference: [irnoti.com/docs](https://irnoti.com/docs).

MIT licensed.
