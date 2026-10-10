package irnoti

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"regexp"
	"time"
)

var sigRe = regexp.MustCompile(`^t=(\d+),v1=([0-9a-f]{64})$`)

// WebhookEvent is a delivery webhook (message.sent | message.delivered | message.failed).
type WebhookEvent struct {
	ID        string `json:"id"`
	Event     string `json:"event"`
	CreatedAt string `json:"createdAt"`
	Data      Object `json:"data"`
}

// VerifyWebhook checks the X-Irnoti-Signature header (`t=<unix>,v1=<hex>`): HMAC-SHA256 of
// "<t>.<raw body>" with your whsec_ key. Pass the RAW body. tolerance 0 means 5 minutes.
func VerifyWebhook(rawBody []byte, signatureHeader, secret string, tolerance time.Duration) bool {
	m := sigRe.FindStringSubmatch(signatureHeader)
	if m == nil {
		return false
	}
	if tolerance == 0 {
		tolerance = 5 * time.Minute
	}
	var t int64
	for _, ch := range m[1] {
		t = t*10 + int64(ch-'0')
	}
	if math.Abs(float64(time.Now().Unix()-t)) > tolerance.Seconds() {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(m[1] + "."))
	mac.Write(rawBody)
	got, err := hex.DecodeString(m[2])
	return err == nil && hmac.Equal(mac.Sum(nil), got)
}

// ParseWebhook verifies and decodes in one step.
func ParseWebhook(rawBody []byte, signatureHeader, secret string, tolerance time.Duration) (*WebhookEvent, error) {
	if !VerifyWebhook(rawBody, signatureHeader, secret, tolerance) {
		return nil, errors.New("irnoti: invalid webhook signature")
	}
	var ev WebhookEvent
	if err := json.Unmarshal(rawBody, &ev); err != nil {
		return nil, err
	}
	return &ev, nil
}
