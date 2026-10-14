package irnoti

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type seen struct {
	method, uri, ctype, idem, auth string
	body                           []byte
}

func server(t *testing.T, status int, payload string) (*Client, *[]seen) {
	var calls []seen
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		calls = append(calls, seen{r.Method, r.URL.RequestURI(), r.Header.Get("Content-Type"), r.Header.Get("Idempotency-Key"), r.Header.Get("Authorization"), b})
		w.WriteHeader(status)
		io.WriteString(w, payload)
	}))
	t.Cleanup(srv.Close)
	return New("irn_test_x", WithBaseURL(srv.URL+"/"), WithRetries(0)), &calls
}

func TestSendBuildsRequest(t *testing.T) {
	c, calls := server(t, 201, `{"batchId":"1","count":2,"totalPrice":"1960","replayed":false}`)
	at := time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)
	r, err := c.SMS.Send(context.Background(), SendParams{LineID: "1", Text: "hi", To: []string{"0912"}, GroupIDs: []string{"12"}, ScheduledAt: &at, IdempotencyKey: "k1"})
	if err != nil || r.TotalPrice != "1960" || r.BatchID != "1" {
		t.Fatalf("unexpected: %+v %v", r, err)
	}
	s := (*calls)[0]
	if s.uri != "/v1/sms/send" || s.auth != "Bearer irn_test_x" || s.idem != "k1" || s.ctype != "application/json" {
		t.Fatalf("bad request: %+v", s)
	}
	var body map[string]any
	json.Unmarshal(s.body, &body)
	if body["lineId"] != "1" || body["scheduledAt"] != "2026-10-20T09:00:00Z" || fmt.Sprint(body["groupIds"]) != "[12]" || body["gradual"] != nil {
		t.Fatalf("bad body: %s", s.body)
	}
}

func TestQueryAndPatternParams(t *testing.T) {
	c, calls := server(t, 200, `[]`)
	c.Messages.List(context.Background(), 2, "90")
	c.Bots.ShopOrders(context.Background(), "5", "")
	if (*calls)[0].uri != "/v1/sms/messages?before=90&take=2" || (*calls)[1].uri != "/v1/botbuilder/bots/5/shop/orders" {
		t.Fatalf("uris: %v", *calls)
	}
	c.SMS.Pattern(context.Background(), "k3f9a2", "0912", nil, "")
	if !strings.Contains(string((*calls)[2].body), `"params":{}`) {
		t.Fatalf("body: %s", (*calls)[2].body)
	}
}

func TestBrokerRoutesAndBodies(t *testing.T) {
	c, calls := server(t, 200, `{}`)
	ctx := context.Background()
	c.Broker.Request(ctx, "7", "sendMessage", Object{"chat_id": 1, "text": "hi"})
	c.Broker.Updates(ctx, "7", UpdatesParams{Limit: 10, AllowedUpdates: []string{"message"}})
	c.Broker.SetWebhook(ctx, "7", WebhookParams{URL: "https://example.com/hook", SecretToken: "s"})
	c.Broker.DeleteWebhook(ctx, "7", true)
	c.Broker.ReplaceToken(ctx, "7", "123456:ABCDEF-token")
	want := [][2]string{{"POST", "/v1/broker/bots/7/request"}, {"POST", "/v1/broker/bots/7/updates"}, {"PUT", "/v1/broker/bots/7/webhook"}, {"DELETE", "/v1/broker/bots/7/webhook"}, {"PUT", "/v1/broker/bots/7/token"}}
	for i, w := range want {
		if (*calls)[i].method != w[0] || (*calls)[i].uri != w[1] {
			t.Fatalf("call %d: %+v want %v", i, (*calls)[i], w)
		}
	}
	if string((*calls)[0].body) != `{"method":"sendMessage","params":{"chat_id":1,"text":"hi"}}` || string((*calls)[1].body) != `{"limit":10,"allowedUpdates":["message"]}` || string((*calls)[3].body) != `{"dropPendingUpdates":true}` {
		t.Fatalf("bodies: %s | %s | %s", (*calls)[0].body, (*calls)[1].body, (*calls)[3].body)
	}
}

func TestErrors(t *testing.T) {
	c, _ := server(t, 400, `{"statusCode":400,"code":"invalid_recipients","details":["0912"]}`)
	_, err := c.SMS.Send(context.Background(), SendParams{LineID: "1", Text: "x", To: []string{"0912"}})
	var ae *APIError
	if !errors.As(err, &ae) || ae.Status != 400 || !ae.IsInvalidRecipients() || !ae.IsValidationError() {
		t.Fatalf("err: %v", err)
	}
	c, _ = server(t, 402, `{"statusCode":402,"code":"insufficient_balance","message":["Not","enough"]}`)
	_, err = c.Wallet.Balance(context.Background())
	if !errors.As(err, &ae) || !ae.IsInsufficientBalance() || ae.Message != "Not enough" {
		t.Fatalf("err: %v", err)
	}
	c, _ = server(t, 401, `{"statusCode":401,"message":"Invalid API key"}`)
	_, err = c.Lines.List(context.Background())
	if !errors.As(err, &ae) || !ae.IsUnauthenticated() || ae.Code != "" {
		t.Fatalf("err: %v", err)
	}
}

func TestNoContentAndCSV(t *testing.T) {
	c, calls := server(t, 204, ``)
	if err := c.Batches.Pause(context.Background(), "7"); err != nil || (*calls)[0].uri != "/v1/sms/batches/7/pause" {
		t.Fatalf("pause: %v", err)
	}
	c, calls = server(t, 200, `{"added":1,"duplicates":0,"invalid":0}`)
	r, err := c.Phonebook.ImportCSV(context.Background(), "12", []byte("0912,Ali\n"), "")
	if err != nil || r.Added != 1 || !strings.HasPrefix((*calls)[0].ctype, "multipart/form-data") {
		t.Fatalf("csv: %v %+v", err, r)
	}
}

func TestWithAPIKey(t *testing.T) {
	c, _ := server(t, 200, `{}`)
	if c.WithAPIKey("irn_live_y") == c {
		t.Fatal("expected a new client")
	}
}

func TestWebhook(t *testing.T) {
	body := []byte(`{"id":"1","event":"message.delivered","createdAt":"x","data":{}}`)
	sign := func(ts int64, b []byte) string {
		m := hmac.New(sha256.New, []byte("whsec"))
		m.Write([]byte(fmt.Sprintf("%d.", ts)))
		m.Write(b)
		return fmt.Sprintf("t=%d,v1=%s", ts, hex.EncodeToString(m.Sum(nil)))
	}
	now := time.Now().Unix()
	if !VerifyWebhook(body, sign(now, body), "whsec", 0) {
		t.Fatal("valid signature rejected")
	}
	ev, err := ParseWebhook(body, sign(now, body), "whsec", 0)
	if err != nil || ev.Event != "message.delivered" {
		t.Fatalf("parse: %v", err)
	}
	if VerifyWebhook(append(body, ' '), sign(now, body), "whsec", 0) || VerifyWebhook(body, sign(now-3600, body), "whsec", 0) || VerifyWebhook(body, "junk", "whsec", 0) {
		t.Fatal("bad signature accepted")
	}
}

func TestSendOmitsLineIDWhenEmpty(t *testing.T) {
	c, calls := server(t, 201, `{"batchId":"1","count":1,"totalPrice":"980","replayed":false}`)
	if _, err := c.SMS.Send(context.Background(), SendParams{Text: "hi", To: []string{"0912"}}); err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	_ = json.Unmarshal((*calls)[0].body, &body)
	if _, ok := body["lineId"]; ok {
		t.Fatalf("lineId must be omitted, got %v", body)
	}
}
