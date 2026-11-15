// Package whatsapp verifies Rimbra's incoming webhook and turns developer
// replies into actions: 1 opens a fix PR, 2 replays paused webhooks, 3 mutes
// the incident for one hour.
package whatsapp

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Mascottz/PockDev/apps/engine/internal/agent"
	"github.com/Mascottz/PockDev/apps/engine/internal/httpx"
	"github.com/Mascottz/PockDev/apps/engine/internal/model"
	"github.com/Mascottz/PockDev/apps/engine/internal/notify"
	"github.com/Mascottz/PockDev/apps/engine/internal/store"
)

// Hook handles POST /webhooks/whatsapp.
type Hook struct {
	Store  store.Store
	Agent  *agent.Agent
	Notify *notify.Client
	Secret string
	Now    func() time.Time
}

// Inbound is the developer reply extracted from Rimbra's webhook body. The
// mapping accepts the common shapes: {"from": "...", "text": "..."} and
// {"from": {"phone": "..."}, "message": {"text": "..."}}.
type Inbound struct {
	From string
	Text string
}

// ServeHTTP handles POST /webhooks/whatsapp.
func (h *Hook) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 256*1024))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !h.verify(r, raw) {
		secPrefix := strings.TrimSpace(h.Secret)
		if len(secPrefix) > 6 {
			secPrefix = secPrefix[:6]
		}
		log.Printf("whatsapp hook: 401 invalid signature (ts=%q sig=%q secret_prefix=%q)", r.Header.Get("X-Rimbra-Timestamp"), r.Header.Get("X-Rimbra-Signature"), secPrefix)
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	var env struct {
		Event string          `json:"event"`
		Data  json.RawMessage `json:"data"`
	}
	_ = json.Unmarshal(raw, &env)
	eventName := strings.TrimSpace(r.Header.Get("X-Rimbra-Event"))
	if eventName == "" {
		eventName = strings.TrimSpace(env.Event)
	}
	if eventName != "" && eventName != "message.received" {
		httpx.JSON(w, http.StatusOK, map[string]any{"ok": true, "ignored": eventName})
		return
	}

	payloadBytes := raw
	if len(env.Data) > 2 && string(env.Data) != "null" {
		payloadBytes = env.Data
	}

	var inbound Inbound
	if err := json.Unmarshal(payloadBytes, &inbound); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	inbound.From = normalizePhone(inbound.From)
	inbound.Text = strings.TrimSpace(inbound.Text)
	log.Printf("whatsapp hook: event=%q from=%q text=%q raw=%s", eventName, inbound.From, inbound.Text, string(payloadBytes))
	if inbound.From == "" || inbound.Text == "" {
		http.Error(w, "from and text are required", http.StatusBadRequest)
		return
	}

	// The newest incident this phone was alerted about owns the conversation.
	listed, err := h.Store.ListIncidents(r.Context(), model.IncidentFilter{AlertPhone: inbound.From, Limit: 20})
	if err != nil || len(listed) == 0 {
		h.reply(w, r.Context(), inbound.From, "I do not have an open alert for this number.")
		return
	}
	inc := listed[0]
	for _, cand := range listed {
		if cand.Status == model.IncidentOpen || cand.Status == model.IncidentFixing {
			inc = cand
			break
		}
	}

	answer := strings.TrimSpace(inbound.Text)
	switch {
	case answer == "1":
		h.reply(w, r.Context(), inbound.From, "Opening a fix PR for "+inc.Title+" now. I will send the link here.")
		go h.doDraftPR(inc, inbound.From)
	case answer == "2":
		updated, n, err := h.Agent.ReplayAll(r.Context(), inc.ID)
		_ = updated
		if err != nil {
			h.reply(w, r.Context(), inbound.From, "I could not replay the webhooks right now.")
			return
		}
		h.reply(w, r.Context(), inbound.From, fmt.Sprintf("Replaying %d paused webhooks for %s now.", n, inc.Title))
	case answer == "3":
		if _, err := h.Agent.Mute(r.Context(), inc.ID); err != nil {
			h.reply(w, r.Context(), inbound.From, "I could not mute this incident right now.")
			return
		}
		h.reply(w, r.Context(), inbound.From, "Muted "+inc.Title+" for 1 hour. The webhooks stay safe.")
	default:
		h.reply(w, r.Context(), inbound.From, "Reply 1 to open a fix PR, 2 to replay paused webhooks, or 3 to mute for 1 hour.")
	}
}

func (h *Hook) doDraftPR(inc model.Incident, phone string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	updated, err := h.Agent.DraftPR(ctx, inc.ID)
	if err != nil {
		h.send(ctx, phone, "I could not open a fix PR: "+err.Error())
		return
	}
	h.send(ctx, phone, "Fix PR ready: "+updated.PRURL+". Review and merge when happy.")
}

func (h *Hook) reply(w http.ResponseWriter, ctx context.Context, phone, text string) {
	h.send(ctx, phone, text)
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Hook) send(ctx context.Context, phone, text string) {
	if h.Notify == nil || !h.Notify.WhatsAppConfigured() {
		return
	}
	if err := h.Notify.SendWhatsApp(ctx, phone, text); err != nil {
		fmt.Printf("whatsapp: reply to %s failed: %v\n", phone, err)
	}
}

// verify checks Rimbra's HMAC SHA 256 webhook signature in X-Rimbra-Signature
// (format sha256=<hex>) against the raw body.
func (h *Hook) verify(r *http.Request, raw []byte) bool {
	secret := strings.TrimSpace(h.Secret)
	if secret == "" {
		return false
	}
	sig := r.Header.Get("X-Rimbra-Signature")
	if sig == "" {
		sig = r.Header.Get("X-Rimbra-Signature-256")
	}
	sig = strings.TrimPrefix(sig, "sha256=")
	if sig == "" {
		return false
	}
	ts := strings.TrimSpace(r.Header.Get("X-Rimbra-Timestamp"))
	if ts != "" {
		macTs := hmac.New(sha256.New, []byte(secret))
		macTs.Write([]byte(ts + "."))
		macTs.Write(raw)
		expTs := hex.EncodeToString(macTs.Sum(nil))
		if hmac.Equal([]byte(strings.ToLower(sig)), []byte(expTs)) {
			return true
		}
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(raw)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(strings.ToLower(sig)), []byte(expected))
}

// UnmarshalJSON pulls from and text out of the common Rimbra body shapes.
func (i *Inbound) UnmarshalJSON(data []byte) error {
	var flat map[string]json.RawMessage
	if err := json.Unmarshal(data, &flat); err != nil {
		return err
	}
	i.From = firstString(flat, "from", "phone", "sender", "wa_id")
	i.Text = firstString(flat, "text", "body", "message", "content")
	var nested struct {
		From    any            `json:"from"`
		Message map[string]any `json:"message"`
		Content map[string]any `json:"content"`
		Sender  any            `json:"sender"`
		Phone   any            `json:"phone"`
	}
	if err := json.Unmarshal(data, &nested); err == nil {
		if i.From == "" {
			i.From = asString(nested.From)
		}
		if i.From == "" {
			i.From = asString(nested.Sender)
		}
		if i.From == "" {
			i.From = asString(nested.Phone)
		}
		if i.Text == "" && nested.Content != nil {
			for _, k := range []string{"text", "body", "content"} {
				if v, ok := nested.Content[k]; ok {
					i.Text = asString(v)
					break
				}
			}
		}
		if i.Text == "" && nested.Message != nil {
			for _, k := range []string{"text", "body", "content"} {
				if v, ok := nested.Message[k]; ok {
					i.Text = asString(v)
					break
				}
			}
		}
	}
	return nil
}

func firstString(flat map[string]json.RawMessage, keys ...string) string {
	for _, k := range keys {
		if raw, ok := flat[k]; ok {
			var s string
			if json.Unmarshal(raw, &s) == nil && strings.TrimSpace(s) != "" {
				return s
			}
		}
	}
	return ""
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case map[string]any:
		for _, k := range []string{"phone", "text", "body", "content"} {
			if s, ok := t[k].(string); ok {
				return s
			}
		}
	}
	return ""
}

func normalizePhone(p string) string {
	p = strings.TrimSpace(p)
	p = strings.TrimPrefix(p, "whatsapp:")
	if idx := strings.IndexByte(p, '@'); idx > 0 {
		p = p[:idx]
	}
	if idx := strings.IndexByte(p, ':'); idx > 0 {
		p = p[:idx]
	}
	p = strings.NewReplacer(" ", "", "-", "").Replace(p)
	if p != "" && p[0] >= '0' && p[0] <= '9' {
		p = "+" + p
	}
	return p
}
