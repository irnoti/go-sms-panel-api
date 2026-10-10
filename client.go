package irnoti

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const DefaultBaseURL = "https://api.irnoti.com"

// Object is a decoded JSON object.
type Object = map[string]any

// Client talks to the API. Create it with New; it is safe for concurrent use.
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
	retries int

	SMS         *SMS
	Messages    *Messages
	Batches     *Batches
	Lines       *Lines
	Wallet      *Wallet
	Phonebook   *Phonebook
	Services    *Services
	LinkTracker *LinkTracker
	NumberBanks *NumberBanks
	Verify      *Verify
	Bots        *Bots
	Voip        *Voip
}

// Option customises a Client.
type Option func(*Client)

func WithBaseURL(u string) Option          { return func(c *Client) { c.baseURL = strings.TrimRight(u, "/") } }
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.http = h } }

// WithRetries sets extra attempts for GETs and sends carrying an idempotency key (default 1).
func WithRetries(n int) Option { return func(c *Client) { c.retries = n } }

// New builds a client for an `irn_live_…` / `irn_test_…` key.
func New(apiKey string, opts ...Option) *Client {
	c := &Client{apiKey: apiKey, baseURL: DefaultBaseURL, http: &http.Client{Timeout: 15 * time.Second}, retries: 1}
	for _, o := range opts {
		o(c)
	}
	c.SMS = &SMS{c}
	c.Messages = &Messages{c}
	c.Batches = &Batches{c}
	c.Lines = &Lines{c}
	c.Wallet = &Wallet{c}
	c.Phonebook = &Phonebook{c}
	c.Services = &Services{c}
	c.LinkTracker = &LinkTracker{c}
	c.NumberBanks = &NumberBanks{c}
	c.Verify = &Verify{c}
	c.Bots = &Bots{c}
	c.Voip = &Voip{c}
	return c
}

// WithAPIKey returns an independent client bound to another key (multi-tenant apps).
func (c *Client) WithAPIKey(apiKey string) *Client {
	return New(apiKey, WithBaseURL(c.baseURL), WithHTTPClient(c.http), WithRetries(c.retries))
}

type reqOpts struct {
	query   url.Values
	body    any
	headers map[string]string
	file    *filePart
}

type filePart struct {
	field, name string
	data        []byte
}

func (c *Client) do(ctx context.Context, method, path string, o reqOpts, out any) error {
	u := c.baseURL + "/v1" + path
	if len(o.query) > 0 {
		u += "?" + o.query.Encode()
	}
	var payload []byte
	contentType := ""
	switch {
	case o.file != nil:
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		part, err := w.CreateFormFile(o.file.field, o.file.name)
		if err != nil {
			return err
		}
		part.Write(o.file.data)
		w.Close()
		payload, contentType = buf.Bytes(), w.FormDataContentType()
	case o.body != nil:
		b, err := json.Marshal(o.body)
		if err != nil {
			return err
		}
		payload, contentType = b, "application/json"
	}

	_, idem := o.headers["Idempotency-Key"]
	attempts := 1
	if method == http.MethodGet || idem {
		attempts += c.retries
	}
	var lastErr error
	for i := 0; i < attempts; i++ {
		req, err := http.NewRequestWithContext(ctx, method, u, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Accept", "application/json")
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		for k, v := range o.headers {
			req.Header.Set(k, v)
		}
		res, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		raw, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode >= 500 && i < attempts-1 {
			lastErr = parseError(res.StatusCode, raw)
			continue
		}
		if res.StatusCode >= 300 {
			return parseError(res.StatusCode, raw)
		}
		if out == nil || len(raw) == 0 {
			return nil
		}
		return json.Unmarshal(raw, out)
	}
	return lastErr
}

func parseError(status int, raw []byte) error {
	var body struct {
		Code    string `json:"code"`
		Message any    `json:"message"`
		Details any    `json:"details"`
	}
	_ = json.Unmarshal(raw, &body)
	msg := fmt.Sprintf("irnoti API error (HTTP %d)", status)
	switch m := body.Message.(type) {
	case string:
		if m != "" {
			msg = m
		}
	case []any:
		parts := make([]string, 0, len(m))
		for _, p := range m {
			parts = append(parts, fmt.Sprint(p))
		}
		msg = strings.Join(parts, " ")
	}
	return &APIError{Status: status, Code: body.Code, Message: msg, Details: body.Details}
}

func idemHeader(key string) map[string]string {
	if key == "" {
		return nil
	}
	return map[string]string{"Idempotency-Key": key}
}

func q(pairs ...any) url.Values {
	v := url.Values{}
	for i := 0; i+1 < len(pairs); i += 2 {
		switch x := pairs[i+1].(type) {
		case nil:
		case string:
			if x != "" {
				v.Set(pairs[i].(string), x)
			}
		case int:
			if x != 0 {
				v.Set(pairs[i].(string), fmt.Sprint(x))
			}
		default:
			v.Set(pairs[i].(string), fmt.Sprint(x))
		}
	}
	return v
}

func esc(id string) string { return url.PathEscape(id) }
