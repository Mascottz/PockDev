// Package webhook implements the Webhook Shield: catch, store, forward,
// replay, and schema drift detection for third party webhooks.
package webhook

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/Mascottz/PockDev/apps/engine/internal/blob"
	"github.com/Mascottz/PockDev/apps/engine/internal/id"
	"github.com/Mascottz/PockDev/apps/engine/internal/model"
	"github.com/Mascottz/PockDev/apps/engine/internal/store"
)

// Ingest catches incoming webhooks and persists them before answering.
type Ingest struct {
	Store   store.Store
	Blobs   blob.Store
	MaxBody int64
	Now     func() time.Time
}

// Response is the catch receipt sent back to the webhook sender.
type Response struct {
	Received   bool   `json:"received"`
	DeliveryID string `json:"deliveryId"`
}

// ServeHTTP handles POST /in/{endpointKey}.
func (h *Ingest) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	endpointKey := r.PathValue("endpointKey")
	if endpointKey == "" {
		http.Error(w, "missing endpoint key", http.StatusNotFound)
		return
	}
	project, err := h.Store.GetProjectByEndpointKey(r.Context(), endpointKey)
	if err != nil {
		http.Error(w, "unknown endpoint", http.StatusNotFound)
		return
	}

	now := time.Now()
	if h.Now != nil {
		now = h.Now()
	}

	// Monthly event quota for the project workspace.
	if store.MonthlyEventQuotaReached(r.Context(), h.Store, project.WorkspaceID, now) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, `{"error": "monthly event quota reached for workspace", "code": "quota_exceeded"}`)
		return
	}

	limit := h.MaxBody
	if limit <= 0 {
		limit = 2 * 1024 * 1024
	}
	r.Body = http.MaxBytesReader(w, r.Body, limit+1)
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if int64(len(raw)) > limit {
		http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
		return
	}
	body := raw

	deliveryID := id.New("dlv_", 24)
	storageKey := fmt.Sprintf("deliveries/%s/%s.body", project.ID, deliveryID)

	headers := map[string]string{}
	for k, v := range r.Header {
		if len(v) > 0 && isForwardableHeader(k) {
			headers[strings.ToLower(k)] = v[0]
		}
	}

	d := &model.Delivery{
		ID:             deliveryID,
		ProjectID:      project.ID,
		EndpointKey:    endpointKey,
		Status:         model.DeliveryRetrying,
		PayloadPreview: preview(body),
		PayloadKeys:    ExtractKeys(body),
		Headers:        headers,
		StorageKey:     storageKey,
		NextAttemptAt:  now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// Persist the raw body first, then the metadata row: a 200 only goes out
	// once both are safe.
	if err := h.Blobs.Put(r.Context(), storageKey, body); err != nil {
		http.Error(w, "storage unavailable", http.StatusServiceUnavailable)
		return
	}
	if err := h.Store.CreateDelivery(r.Context(), d); err != nil {
		http.Error(w, "storage unavailable", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"received": true, "deliveryId": %q}`, deliveryID)
}

func isForwardableHeader(k string) bool {
	switch strings.ToLower(k) {
	case "host", "content-length", "connection", "accept-encoding", "transfer-encoding",
		"x-pockdev-delivery-id", "authorization":
		return false
	}
	return true
}

func preview(body []byte) string {
	const max = 300
	s := strings.TrimSpace(string(body))
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > max {
		s = s[:max]
	}
	return s
}

// ExtractKeys returns the top level and nested JSON keys of a payload as
// dotted paths, so ["order","order.id","order.rider.phone", ...]. Non JSON
// payloads return no keys.
func ExtractKeys(body []byte) []string {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" || (trimmed[0] != '{' && trimmed[0] != '[') {
		return []string{}
	}
	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		return []string{}
	}
	keys := map[string]bool{}
	collectKeys(v, "", keys)
	out := make([]string, 0, len(keys))
	for k := range keys {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func collectKeys(v any, prefix string, out map[string]bool) {
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			p := k
			if prefix != "" {
				p = prefix + "." + k
			}
			out[p] = true
			collectKeys(child, p, out)
		}
	case []any:
		for _, child := range t {
			collectKeys(child, prefix, out)
		}
	}
}

// FingerprintKeys builds a stable digest of a key set for grouping.
func FingerprintKeys(keys []string) string {
	sorted := append([]string(nil), keys...)
	sort.Strings(sorted)
	h := sha256.New()
	for _, k := range sorted {
		h.Write([]byte(k))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}
