// Package whatsapp verifies Rimbra's incoming webhook and turns developer
// replies into actions: 1 opens a fix PR, 2 replays paused webhooks, 3 mutes
// the incident for one hour.
package whatsapp

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Mascottz/PockDev/apps/engine/internal/agent"
	"github.com/Mascottz/PockDev/apps/engine/internal/httpx"
	"github.com/Mascottz/PockDev/apps/engine/internal/model"
	"github.com/Mascottz/PockDev/apps/engine/internal/notify"
	"github.com/Mascottz/PockDev/apps/engine/internal/store"
)

// Hook handles POST /webhooks/whatsapp.
type Hook struct {
	Store     store.Store
	Agent     *agent.Agent
	Notify    *notify.Client
	Secret    string
	Now       func() time.Time
	lidPhones sync.Map
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
		hdrSummary := make([]string, 0, len(r.Header))
		for k, vals := range r.Header {
			lk := strings.ToLower(k)
			if strings.Contains(lk, "sig") || strings.Contains(lk, "time") || strings.Contains(lk, "rimbra") || strings.Contains(lk, "webhook") || strings.Contains(lk, "svix") || strings.Contains(lk, "event") {
				hdrSummary = append(hdrSummary, fmt.Sprintf("%s=%q", k, strings.Join(vals, ",")))
			} else {
				hdrSummary = append(hdrSummary, k)
			}
		}
		log.Printf("whatsapp hook: 401 headers=%v body_len=%d", hdrSummary, len(raw))
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
	if inbound.From == "" || inbound.Text == "" {
		httpx.JSON(w, http.StatusOK, map[string]any{"ok": true, "ignored": "empty"})
		return
	}

	fields := strings.Fields(inbound.Text)
	if len(fields) == 0 || len(fields) > 2 || (fields[0] != "1" && fields[0] != "2" && fields[0] != "3") {
		// Never log or auto-reply to normal personal or group messages on the linked number.
		httpx.JSON(w, http.StatusOK, map[string]any{"ok": true, "ignored": "non_command"})
		return
	}
	answer := fields[0]
	code := ""
	if len(fields) == 2 {
		code = strings.ToLower(strings.TrimPrefix(fields[1], "#"))
	}

	lkupPhone := inbound.From
	if cached, ok := h.lidPhones.Load(inbound.From); ok {
		lkupPhone = cached.(string)
	}

	var inc *model.Incident
	if code != "" {
		if all, err := h.Store.ListIncidents(r.Context(), model.IncidentFilter{Limit: 50}); err == nil {
			for i := range all {
				if all[i].LastAlertPhone != "" && strings.HasSuffix(strings.ToLower(all[i].ID), code) {
					inc = &all[i]
					break
				}
			}
		}
	}
	if inc == nil {
		if listed, err := h.Store.ListIncidents(r.Context(), model.IncidentFilter{AlertPhone: lkupPhone, Limit: 20}); err == nil && len(listed) > 0 {
			inc = &listed[0]
			for i := range listed {
				if listed[i].Status == model.IncidentOpen || listed[i].Status == model.IncidentFixing {
					inc = &listed[i]
					break
				}
			}
		}
	}
	// Fallback when WhatsApp multi-device sends an opaque LID instead of E.164 sender phone.
	if inc == nil {
		if all, err := h.Store.ListIncidents(r.Context(), model.IncidentFilter{Limit: 50}); err == nil {
			phones := map[string]bool{}
			var newestOpen *model.Incident
			var newestAny *model.Incident
			for i := range all {
				c := &all[i]
				if c.LastAlertPhone == "" {
					continue
				}
				if newestAny == nil {
					newestAny = c
				}
				if c.Status == model.IncidentOpen || c.Status == model.IncidentFixing {
					phones[c.LastAlertPhone] = true
					if newestOpen == nil {
						newestOpen = c
					}
				}
			}
			if newestOpen != nil && len(phones) == 1 {
				inc = newestOpen
			} else if newestOpen == nil && newestAny != nil {
				inc = newestAny
			}
		}
	}

	if inc == nil {
		log.Printf("whatsapp hook: command=%q from=%q matched no alerted incident", answer, inbound.From)
		httpx.JSON(w, http.StatusOK, map[string]any{"ok": true, "ignored": "no_incident"})
		return
	}

	replyPhone := inc.LastAlertPhone
	if replyPhone == "" {
		replyPhone = inbound.From
	} else if replyPhone != inbound.From {
		h.lidPhones.Store(inbound.From, replyPhone)
	}
	log.Printf("whatsapp hook: command=%q incident=%s reply_to=%s", answer, inc.ID, replyPhone)

	switch {
	case answer == "1":
		h.reply(w, r.Context(), replyPhone, "Opening a fix PR for "+inc.Title+" now. I will send the link here.")
		go h.doDraftPR(*inc, replyPhone)
	case answer == "2":
		updated, n, err := h.Agent.ReplayAll(r.Context(), inc.ID)
		_ = updated
		if err != nil {
			h.reply(w, r.Context(), replyPhone, "I could not replay the webhooks right now.")
			return
		}
		h.reply(w, r.Context(), replyPhone, fmt.Sprintf("Replaying %d paused webhooks for %s now.", n, inc.Title))
	case answer == "3":
		if _, err := h.Agent.Mute(r.Context(), inc.ID); err != nil {
			h.reply(w, r.Context(), replyPhone, "I could not mute this incident right now.")
			return
		}
		h.reply(w, r.Context(), replyPhone, "Muted "+inc.Title+" for 1 hour. The webhooks stay safe.")
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
func firstHeader(r *http.Request, names ...string) string {
	for _, n := range names {
		if v := strings.TrimSpace(r.Header.Get(n)); v != "" {
			return v
		}
	}
	return ""
}

func (h *Hook) verify(r *http.Request, raw []byte) bool {
	secret := strings.TrimSpace(h.Secret)
	if secret == "" {
		return false
	}
	if q := strings.TrimSpace(r.URL.Query().Get("secret")); q != "" {
		if hmac.Equal([]byte(q), []byte(secret)) {
			return true
		}
	}
	rawSig := firstHeader(r,
		"X-Rimbra-Signature", "X-Rimbra-Signature-256", "Rimbra-Signature",
		"Webhook-Signature", "Svix-Signature", "X-Webhook-Signature",
		"X-Signature", "X-Signature-256", "X-Hub-Signature-256",
	)
	if rawSig == "" {
		return false
	}
	ts := firstHeader(r,
		"X-Rimbra-Timestamp", "Rimbra-Timestamp",
		"Webhook-Timestamp", "Svix-Timestamp", "X-Webhook-Timestamp", "X-Timestamp",
	)
	msgID := firstHeader(r, "Webhook-Id", "Svix-Id", "X-Rimbra-Id", "X-Webhook-Id")

	keys := [][]byte{[]byte(secret)}
	if trimmed := strings.TrimPrefix(secret, "whsec_"); trimmed != secret {
		keys = append(keys, []byte(trimmed))
		if dec, err := base64.StdEncoding.DecodeString(trimmed); err == nil && len(dec) > 0 {
			keys = append(keys, dec)
		}
	}

	payloads := [][]byte{raw}
	if ts != "" {
		payloads = append([][]byte{[]byte(ts + "." + string(raw))}, payloads...)
	}
	if msgID != "" && ts != "" {
		payloads = append([][]byte{[]byte(msgID + "." + ts + "." + string(raw))}, payloads...)
	}

	var candidates []string
	for _, part := range strings.Fields(strings.ReplaceAll(rawSig, ",", " ")) {
		p := strings.TrimSpace(part)
		p = strings.TrimPrefix(p, "sha256=")
		p = strings.TrimPrefix(p, "v1=")
		if p != "" && p != "v1" {
			candidates = append(candidates, p)
		}
	}

	for _, k := range keys {
		for _, pl := range payloads {
			mac := hmac.New(sha256.New, k)
			mac.Write(pl)
			sum := mac.Sum(nil)
			hexExp := hex.EncodeToString(sum)
			b64Exp := base64.StdEncoding.EncodeToString(sum)
			for _, cand := range candidates {
				if hmac.Equal([]byte(strings.ToLower(cand)), []byte(hexExp)) || hmac.Equal([]byte(cand), []byte(b64Exp)) {
					return true
				}
			}
		}
	}
	return false
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
