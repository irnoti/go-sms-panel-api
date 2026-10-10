package irnoti

import (
	"context"
	"net/http"
	"time"
)

// BatchResult is the answer of every send.
type BatchResult struct {
	BatchID    string `json:"batchId"`
	Count      int    `json:"count"`
	TotalPrice string `json:"totalPrice"` // integer Rial
	Replayed   bool   `json:"replayed"`
}

// Quote is the exact cost of a send.
type Quote struct {
	Count      int    `json:"count"`
	Parts      int    `json:"parts"`
	TotalPrice string `json:"totalPrice"`
}

type Line struct {
	ID     string `json:"id"`
	Number string `json:"number"`
	Kind   string `json:"kind"`
}

type Message struct {
	ID          string     `json:"id"`
	Kind        string     `json:"kind"`
	To          string     `json:"to"`
	Body        string     `json:"body"`
	Parts       int        `json:"parts"`
	TotalPrice  string     `json:"totalPrice"`
	Status      string     `json:"status"` // queued | sent | delivered | failed
	Error       *string    `json:"error"`
	Sandbox     bool       `json:"sandbox"`
	CreatedAt   time.Time  `json:"createdAt"`
	SentAt      *time.Time `json:"sentAt"`
	DeliveredAt *time.Time `json:"deliveredAt"`
}

type Group struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Contacts  int    `json:"contacts"`
	CreatedAt string `json:"createdAt"`
}

type Contact struct {
	Mobile   string            `json:"mobile"`
	Name     string            `json:"name,omitempty"`
	LastName string            `json:"lastName,omitempty"`
	Company  string            `json:"company,omitempty"`
	Email    string            `json:"email,omitempty"`
	City     string            `json:"city,omitempty"`
	Note     string            `json:"note,omitempty"`
	Birth    string            `json:"birth,omitempty"`
	Fields   map[string]string `json:"fields,omitempty"`
}

type AddContactsResult struct {
	Added      int `json:"added"`
	Duplicates int `json:"duplicates"`
	Invalid    int `json:"invalid"`
}

// Gradual releases ChunkSize messages every IntervalMinutes.
type Gradual struct {
	ChunkSize       int `json:"chunkSize"`
	IntervalMinutes int `json:"intervalMinutes"`
}

// SendParams for SMS.Send / SMS.Quote. LineID "sandbox" with an irn_test_ key sends nothing.
type SendParams struct {
	LineID         string
	Text           string
	To             []string
	GroupIDs       []string
	ScheduledAt    *time.Time // 1 minute – 60 days ahead (Send only)
	Gradual        *Gradual   // (Send only)
	IdempotencyKey string     // (Send only) retrying with the same key never sends twice
}

type SMS struct{ c *Client }

// Send requires scope sms.send.
func (s *SMS) Send(ctx context.Context, p SendParams) (*BatchResult, error) {
	var out BatchResult
	err := s.c.do(ctx, http.MethodPost, "/sms/send", reqOpts{headers: idemHeader(p.IdempotencyKey), body: Object{
		"lineId": p.LineID, "to": nilIfEmpty(p.To), "groupIds": nilIfEmpty(p.GroupIDs), "text": p.Text,
		"scheduledAt": p.ScheduledAt, "gradual": p.Gradual,
	}}, &out)
	return &out, err
}

// Quote requires scope sms.read and charges nothing.
func (s *SMS) Quote(ctx context.Context, p SendParams) (*Quote, error) {
	var out Quote
	err := s.c.do(ctx, http.MethodPost, "/sms/quote", reqOpts{body: Object{
		"lineId": p.LineID, "to": nilIfEmpty(p.To), "groupIds": nilIfEmpty(p.GroupIDs), "text": p.Text,
	}}, &out)
	return &out, err
}

// Pattern sends an approved template (OTP / transactional).
func (s *SMS) Pattern(ctx context.Context, code, to string, params map[string]string, idempotencyKey string) (*BatchResult, error) {
	if params == nil {
		params = map[string]string{}
	}
	var out BatchResult
	err := s.c.do(ctx, http.MethodPost, "/sms/pattern", reqOpts{headers: idemHeader(idempotencyKey), body: Object{"code": code, "to": to, "params": params}}, &out)
	return &out, err
}

// SOS sends critical alerts with provider failover (<= 200 recipients, sms.sos tariff).
func (s *SMS) SOS(ctx context.Context, to []string, text, idempotencyKey string) (*BatchResult, error) {
	var out BatchResult
	err := s.c.do(ctx, http.MethodPost, "/sms/sos", reqOpts{headers: idemHeader(idempotencyKey), body: Object{"to": to, "text": text}}, &out)
	return &out, err
}

func nilIfEmpty(v []string) any {
	if len(v) == 0 {
		return nil
	}
	return v
}

type Messages struct{ c *Client }

// List returns newest first; page with before = id of the last message. take 0 = server default.
func (m *Messages) List(ctx context.Context, take int, before string) ([]Message, error) {
	var out []Message
	err := m.c.do(ctx, http.MethodGet, "/sms/messages", reqOpts{query: q("take", take, "before", before)}, &out)
	return out, err
}
func (m *Messages) Get(ctx context.Context, id string) (*Message, error) {
	var out Message
	err := m.c.do(ctx, http.MethodGet, "/sms/messages/"+esc(id), reqOpts{}, &out)
	return &out, err
}

// Check asks the operator for the final state now (30/min).
func (m *Messages) Check(ctx context.Context, id string) (*Message, error) {
	var out Message
	err := m.c.do(ctx, http.MethodPost, "/sms/messages/"+esc(id)+"/check", reqOpts{}, &out)
	return &out, err
}

type Batches struct{ c *Client }

func (b *Batches) Scheduled(ctx context.Context) ([]Object, error) {
	var out []Object
	err := b.c.do(ctx, http.MethodGet, "/sms/scheduled", reqOpts{}, &out)
	return out, err
}
func (b *Batches) Cancel(ctx context.Context, batchID string) (Object, error) {
	var out Object
	err := b.c.do(ctx, http.MethodPost, "/sms/batches/"+esc(batchID)+"/cancel", reqOpts{}, &out)
	return out, err
}
func (b *Batches) Pause(ctx context.Context, batchID string) error {
	return b.c.do(ctx, http.MethodPost, "/sms/batches/"+esc(batchID)+"/pause", reqOpts{}, nil)
}
func (b *Batches) Resume(ctx context.Context, batchID string) error {
	return b.c.do(ctx, http.MethodPost, "/sms/batches/"+esc(batchID)+"/resume", reqOpts{}, nil)
}

type Lines struct{ c *Client }

func (l *Lines) List(ctx context.Context) ([]Line, error) {
	var out []Line
	err := l.c.do(ctx, http.MethodGet, "/sms/lines", reqOpts{}, &out)
	return out, err
}

type Wallet struct{ c *Client }

type Balance struct {
	Balance  string `json:"balance"` // integer Rial
	Currency string `json:"currency"`
}

func (w *Wallet) Balance(ctx context.Context) (*Balance, error) {
	var out Balance
	err := w.c.do(ctx, http.MethodGet, "/wallet", reqOpts{}, &out)
	return &out, err
}
func (w *Wallet) Transactions(ctx context.Context) ([]Object, error) {
	var out []Object
	err := w.c.do(ctx, http.MethodGet, "/wallet/transactions", reqOpts{}, &out)
	return out, err
}

type Phonebook struct{ c *Client }

func (p *Phonebook) Groups(ctx context.Context) ([]Group, error) {
	var out []Group
	err := p.c.do(ctx, http.MethodGet, "/phonebook/groups", reqOpts{}, &out)
	return out, err
}
func (p *Phonebook) CreateGroup(ctx context.Context, name string) (Object, error) {
	var out Object
	err := p.c.do(ctx, http.MethodPost, "/phonebook/groups", reqOpts{body: Object{"name": name}}, &out)
	return out, err
}
func (p *Phonebook) DeleteGroup(ctx context.Context, id string) error {
	return p.c.do(ctx, http.MethodDelete, "/phonebook/groups/"+esc(id), reqOpts{}, nil)
}
func (p *Phonebook) Contacts(ctx context.Context, groupID string, take int, after, query string) (any, error) {
	var out any
	err := p.c.do(ctx, http.MethodGet, "/phonebook/groups/"+esc(groupID)+"/contacts", reqOpts{query: q("take", take, "after", after, "q", query)}, &out)
	return out, err
}

// AddContacts takes up to 10,000 per call; invalid/duplicate numbers are counted, not fatal.
func (p *Phonebook) AddContacts(ctx context.Context, groupID string, contacts []Contact) (*AddContactsResult, error) {
	var out AddContactsResult
	err := p.c.do(ctx, http.MethodPost, "/phonebook/groups/"+esc(groupID)+"/contacts", reqOpts{body: Object{"contacts": contacts}}, &out)
	return &out, err
}

// ImportCSV uploads CSV/TXT, one `mobile[,name]` per line (max 5 MB).
func (p *Phonebook) ImportCSV(ctx context.Context, groupID string, csv []byte, filename string) (*AddContactsResult, error) {
	if filename == "" {
		filename = "contacts.csv"
	}
	var out AddContactsResult
	err := p.c.do(ctx, http.MethodPost, "/phonebook/groups/"+esc(groupID)+"/import", reqOpts{file: &filePart{"file", filename, csv}}, &out)
	return &out, err
}

// UpdateContact takes camelCase API fields: mobile, name, lastName, company, email, city, note, birth, fields, optedOut.
func (p *Phonebook) UpdateContact(ctx context.Context, id string, attrs Object) (Object, error) {
	var out Object
	err := p.c.do(ctx, http.MethodPatch, "/phonebook/contacts/"+esc(id), reqOpts{body: attrs}, &out)
	return out, err
}
func (p *Phonebook) DeleteContact(ctx context.Context, id string) error {
	return p.c.do(ctx, http.MethodDelete, "/phonebook/contacts/"+esc(id), reqOpts{}, nil)
}

// Service products.
const (
	Regional      = "regional"
	Targeted      = "targeted"
	Peyamap       = "peyamap"
	Smart         = "smart"
	Messenger     = "messenger"
	Voice         = "voice"
	International = "international"
)

// Services: quote -> create -> refresh -> cancel.
type Services struct{ c *Client }

func (s *Services) Prices(ctx context.Context) ([]Object, error) {
	var out []Object
	err := s.c.do(ctx, http.MethodGet, "/services/prices", reqOpts{}, &out)
	return out, err
}
func (s *Services) InternationalRates(ctx context.Context) (any, error) {
	var out any
	err := s.c.do(ctx, http.MethodGet, "/services/international/rates", reqOpts{}, &out)
	return out, err
}
func (s *Services) List(ctx context.Context, product string) ([]Object, error) {
	var out []Object
	err := s.c.do(ctx, http.MethodGet, "/services/"+product, reqOpts{}, &out)
	return out, err
}
func (s *Services) Quote(ctx context.Context, product string, body Object) (Object, error) {
	var out Object
	err := s.c.do(ctx, http.MethodPost, "/services/"+product+"/quote", reqOpts{body: body}, &out)
	return out, err
}
func (s *Services) Create(ctx context.Context, product string, body Object) (Object, error) {
	var out Object
	err := s.c.do(ctx, http.MethodPost, "/services/"+product, reqOpts{body: body}, &out)
	return out, err
}
func (s *Services) Get(ctx context.Context, product, id string) (Object, error) {
	var out Object
	err := s.c.do(ctx, http.MethodGet, "/services/"+product+"/"+esc(id), reqOpts{}, &out)
	return out, err
}
func (s *Services) Refresh(ctx context.Context, product, id string) (Object, error) {
	var out Object
	err := s.c.do(ctx, http.MethodPost, "/services/"+product+"/"+esc(id)+"/refresh", reqOpts{}, &out)
	return out, err
}

// Cancel works only before the provider starts; the cost is refunded.
func (s *Services) Cancel(ctx context.Context, product, id string) (Object, error) {
	var out Object
	err := s.c.do(ctx, http.MethodPost, "/services/"+product+"/"+esc(id)+"/cancel", reqOpts{}, &out)
	return out, err
}

type LinkTracker struct{ c *Client }

// Send body: lineId, text, url, to?, groupIds?, variants?, holdoutPercent?, decideAfterMinutes?.
func (l *LinkTracker) Send(ctx context.Context, body Object) (Object, error) {
	var out Object
	err := l.c.do(ctx, http.MethodPost, "/link-tracker/send", reqOpts{body: body}, &out)
	return out, err
}
func (l *LinkTracker) Holdouts(ctx context.Context) ([]Object, error) {
	var out []Object
	err := l.c.do(ctx, http.MethodGet, "/link-tracker/holdouts", reqOpts{}, &out)
	return out, err
}
func (l *LinkTracker) SendNow(ctx context.Context, holdoutID string) (Object, error) {
	var out Object
	err := l.c.do(ctx, http.MethodPost, "/link-tracker/holdouts/"+esc(holdoutID)+"/send-now", reqOpts{}, &out)
	return out, err
}
func (l *LinkTracker) Campaigns(ctx context.Context) ([]Object, error) {
	var out []Object
	err := l.c.do(ctx, http.MethodGet, "/link-tracker/campaigns", reqOpts{}, &out)
	return out, err
}
func (l *LinkTracker) Campaign(ctx context.Context, batchID string) (Object, error) {
	var out Object
	err := l.c.do(ctx, http.MethodGet, "/link-tracker/campaigns/"+esc(batchID), reqOpts{}, &out)
	return out, err
}

type NumberBanks struct{ c *Client }

// BankSend: filters may contain operator, province, city, tag.
type BankSend struct {
	BankID, LineID, Text string
	Count                int
	Filters              map[string]string
	IdempotencyKey       string // Send only
}

func (b BankSend) body() Object {
	f := b.Filters
	if f == nil {
		f = map[string]string{}
	}
	return Object{"bankId": b.BankID, "count": b.Count, "lineId": b.LineID, "text": b.Text, "filters": f}
}
func (n *NumberBanks) List(ctx context.Context) ([]Object, error) {
	var out []Object
	err := n.c.do(ctx, http.MethodGet, "/number-banks", reqOpts{}, &out)
	return out, err
}
func (n *NumberBanks) Facets(ctx context.Context, bankID string) (Object, error) {
	var out Object
	err := n.c.do(ctx, http.MethodGet, "/number-banks/"+esc(bankID)+"/facets", reqOpts{}, &out)
	return out, err
}
func (n *NumberBanks) Quote(ctx context.Context, p BankSend) (Object, error) {
	var out Object
	err := n.c.do(ctx, http.MethodPost, "/number-banks/quote", reqOpts{body: p.body()}, &out)
	return out, err
}
func (n *NumberBanks) Send(ctx context.Context, p BankSend) (Object, error) {
	var out Object
	err := n.c.do(ctx, http.MethodPost, "/number-banks/send", reqOpts{body: p.body(), headers: idemHeader(p.IdempotencyKey)}, &out)
	return out, err
}

type Verify struct{ c *Client }

func (v *Verify) Info(ctx context.Context) (Object, error) {
	var out Object
	err := v.c.do(ctx, http.MethodGet, "/verify/info", reqOpts{}, &out)
	return out, err
}

// Create body: exactly one of `to` (mobile) or `bot` ({id, ref}); options lineId, mode, length, ttlSec,
// maxAttempts, redirectUrl, callbackUrl, clientRef, brandName, brandColor.
func (v *Verify) Create(ctx context.Context, body Object) (Object, error) {
	var out Object
	err := v.c.do(ctx, http.MethodPost, "/verify/sessions", reqOpts{body: body}, &out)
	return out, err
}
func (v *Verify) List(ctx context.Context, take int, before string) ([]Object, error) {
	var out []Object
	err := v.c.do(ctx, http.MethodGet, "/verify/sessions", reqOpts{query: q("take", take, "before", before)}, &out)
	return out, err
}
func (v *Verify) Get(ctx context.Context, id string) (Object, error) {
	var out Object
	err := v.c.do(ctx, http.MethodGet, "/verify/sessions/"+esc(id), reqOpts{}, &out)
	return out, err
}
func (v *Verify) Cancel(ctx context.Context, id string) error {
	return v.c.do(ctx, http.MethodPost, "/verify/sessions/"+esc(id)+"/cancel", reqOpts{}, nil)
}
func (v *Verify) Stats(ctx context.Context) (Object, error) {
	var out Object
	err := v.c.do(ctx, http.MethodGet, "/verify/stats", reqOpts{}, &out)
	return out, err
}

// Bots is the bot builder API. Creating/configuring bots is panel-only; `to`/`message`
// follow the panel's bot sender shapes.
type Bots struct{ c *Client }

func (b *Bots) Info(ctx context.Context) (Object, error) {
	var out Object
	err := b.c.do(ctx, http.MethodGet, "/botbuilder/info", reqOpts{}, &out)
	return out, err
}
func (b *Bots) List(ctx context.Context) ([]Object, error) {
	var out []Object
	err := b.c.do(ctx, http.MethodGet, "/botbuilder/bots", reqOpts{}, &out)
	return out, err
}
func (b *Bots) Get(ctx context.Context, botID string) (Object, error) {
	var out Object
	err := b.c.do(ctx, http.MethodGet, "/botbuilder/bots/"+esc(botID), reqOpts{}, &out)
	return out, err
}
func (b *Bots) Stats(ctx context.Context, botID string) (Object, error) {
	var out Object
	err := b.c.do(ctx, http.MethodGet, "/botbuilder/bots/"+esc(botID)+"/stats", reqOpts{}, &out)
	return out, err
}
func (b *Bots) Subscribers(ctx context.Context, botID string, take int, before string) ([]Object, error) {
	var out []Object
	err := b.c.do(ctx, http.MethodGet, "/botbuilder/bots/"+esc(botID)+"/subscribers", reqOpts{query: q("take", take, "before", before)}, &out)
	return out, err
}
func (b *Bots) Messages(ctx context.Context, botID string, take int, before string) ([]Object, error) {
	var out []Object
	err := b.c.do(ctx, http.MethodGet, "/botbuilder/bots/"+esc(botID)+"/messages", reqOpts{query: q("take", take, "before", before)}, &out)
	return out, err
}
func (b *Bots) Send(ctx context.Context, botID string, to, message Object, idempotencyKey string) (Object, error) {
	var out Object
	err := b.c.do(ctx, http.MethodPost, "/botbuilder/bots/"+esc(botID)+"/send", reqOpts{body: Object{"to": to, "message": message}, headers: idemHeader(idempotencyKey)}, &out)
	return out, err
}

// OTP: length 0 / empty template mean the server defaults.
func (b *Bots) OTP(ctx context.Context, botID string, to Object, length int, template, idempotencyKey string) (Object, error) {
	body := Object{"to": to}
	if length > 0 {
		body["length"] = length
	}
	if template != "" {
		body["template"] = template
	}
	var out Object
	err := b.c.do(ctx, http.MethodPost, "/botbuilder/bots/"+esc(botID)+"/otp", reqOpts{body: body, headers: idemHeader(idempotencyKey)}, &out)
	return out, err
}
func (b *Bots) ShopOrders(ctx context.Context, botID, status string) ([]Object, error) {
	var out []Object
	err := b.c.do(ctx, http.MethodGet, "/botbuilder/bots/"+esc(botID)+"/shop/orders", reqOpts{query: q("status", status)}, &out)
	return out, err
}
func (b *Bots) ShopStats(ctx context.Context, botID string) (Object, error) {
	var out Object
	err := b.c.do(ctx, http.MethodGet, "/botbuilder/bots/"+esc(botID)+"/shop/stats", reqOpts{}, &out)
	return out, err
}

type Voip struct{ c *Client }

func (v *Voip) Calls(ctx context.Context) ([]Object, error) {
	var out []Object
	err := v.c.do(ctx, http.MethodGet, "/voip/calls", reqOpts{}, &out)
	return out, err
}
func (v *Voip) Price(ctx context.Context) (Object, error) {
	var out Object
	err := v.c.do(ctx, http.MethodGet, "/voip/price", reqOpts{}, &out)
	return out, err
}
func (v *Voip) Call(ctx context.Context, from, to string) (Object, error) {
	var out Object
	err := v.c.do(ctx, http.MethodPost, "/voip/calls", reqOpts{body: Object{"from": from, "to": to}}, &out)
	return out, err
}
