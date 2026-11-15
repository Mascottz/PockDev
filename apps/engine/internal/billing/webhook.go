package billing

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/Mascottz/PockDev/apps/engine/internal/httpx"
	"github.com/Mascottz/PockDev/apps/engine/internal/store"
)

// ProValidity is how long one successful payment keeps a workspace on Pro.
const ProValidity = 31 * 24 * time.Hour

// Webhook consumes Paystack event callbacks at POST /webhooks/paystack.
type Webhook struct {
	Secret string
	Store  store.Store
	Now    func() time.Time
}

func (h *Webhook) now() time.Time {
	if h.Now != nil {
		return h.Now()
	}
	return time.Now()
}

// ValidSignature reports whether the request is authentic: either the
// x-paystack-signature HMAC-SHA512 of the raw body matches, or the secret
// query parameter matches the shared secret (fallback for proxies that strip
// custom headers). Both comparisons are constant time.
func ValidSignature(secret string, body []byte, headerSig, querySecret string) bool {
	if secret == "" {
		return false
	}
	if headerSig != "" {
		mac := hmac.New(sha512.New, []byte(secret))
		mac.Write(body)
		want := hex.EncodeToString(mac.Sum(nil))
		if hmac.Equal([]byte(want), []byte(headerSig)) {
			return true
		}
	}
	if querySecret != "" {
		if hmac.Equal([]byte(secret), []byte(querySecret)) {
			return true
		}
	}
	return false
}

// event is the Paystack callback envelope.
type event struct {
	Event string `json:"event"`
	Data  struct {
		Reference        string `json:"reference"`
		Amount           int64  `json:"amount"`
		Currency         string `json:"currency"`
		SubscriptionCode string `json:"subscription_code"`
		Customer         struct {
			CustomerCode string `json:"customer_code"`
		} `json:"customer"`
		Metadata struct {
			WorkspaceID string `json:"workspace_id"`
		} `json:"metadata"`
	} `json:"data"`
}

// ServeHTTP handles POST /webhooks/paystack.
func (h *Webhook) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad request")
		return
	}
	if !ValidSignature(h.Secret, body, r.Header.Get("x-paystack-signature"), r.URL.Query().Get("secret")) {
		httpx.Error(w, http.StatusUnauthorized, "invalid signature")
		return
	}

	var ev event
	if err := json.Unmarshal(body, &ev); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	switch ev.Event {
	case "charge.success", "subscription.create":
		h.upgrade(r.Context(), ev)
	case "subscription.disable", "subscription.not_renew":
		h.cancel(r.Context(), ev)
	}

	// Paystack retries on non 200, so events always confirm.
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Webhook) upgrade(ctx context.Context, ev event) {
	if ev.Data.Reference == "" || ev.Data.Metadata.WorkspaceID == "" {
		return
	}
	_ = h.Store.UpgradeWorkspacePro(ctx,
		ev.Data.Metadata.WorkspaceID,
		ev.Data.Reference,
		ev.Data.Customer.CustomerCode,
		ev.Data.SubscriptionCode,
		ev.Data.Currency,
		ev.Data.Amount,
		h.now().Add(ProValidity))
}

func (h *Webhook) cancel(ctx context.Context, ev event) {
	code := ev.Data.SubscriptionCode
	if code == "" {
		code = ev.Data.Customer.CustomerCode
	}
	if code == "" {
		return
	}
	_ = h.Store.CancelWorkspaceSubscription(ctx, code)
}
