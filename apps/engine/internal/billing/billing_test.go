package billing

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Mascottz/PockDev/apps/engine/internal/store/memstore"
)

func sign(secret string, body []byte) string {
	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestValidSignature(t *testing.T) {
	body := []byte(`{"event":"charge.success"}`)
	good := sign("shh", body)

	if !ValidSignature("shh", body, good, "") {
		t.Fatal("correct HMAC must pass")
	}
	if ValidSignature("shh", body, sign("other", body), "") {
		t.Fatal("wrong HMAC must fail")
	}
	if ValidSignature("", body, "", "") {
		t.Fatal("empty secret must fail")
	}
	if ValidSignature("shh", []byte("tampered"), good, "") {
		t.Fatal("tampered body must fail")
	}
	if !ValidSignature("shh", body, "", "shh") {
		t.Fatal("query secret fallback must pass")
	}
	if ValidSignature("shh", body, "", "nope") {
		t.Fatal("wrong query secret must fail")
	}
	if !ValidSignature("shh", body, "garbage", "shh") {
		t.Fatal("query fallback must work even with a bad header")
	}
}

func webhookSetup(t *testing.T, now *time.Time) (*memstore.Mem, http.Handler) {
	t.Helper()
	st := memstore.New()
	h := &Webhook{
		Secret: "whsec",
		Store:  st,
		Now: func() time.Time {
			if now != nil {
				return *now
			}
			return time.Now()
		},
	}
	return st, h
}

func postWebhook(t *testing.T, h http.Handler, target string, body []byte, sig string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, target, bytes.NewReader(body))
	if sig != "" {
		req.Header.Set("x-paystack-signature", sig)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestWebhookUpgradeAndIdempotentDelivery(t *testing.T) {
	base := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	now := base
	st, h := webhookSetup(t, &now)
	ctx := context.Background()

	// Seed a workspace through the normal GitHub sign in path.
	_, wsp, err := st.UpsertGitHubUser(ctx, 1, "Mascottz", "Mascott", "m@example.com", "", true)
	if err != nil {
		t.Fatal(err)
	}

	body := []byte(`{"event":"charge.success","data":{"reference":"ref_1","amount":1500000,"currency":"NGN","subscription_code":"sub_1","customer":{"customer_code":"cus_1"},"metadata":{"workspace_id":"` + wsp.ID + `"}}}`)
	rec := postWebhook(t, h, "/webhooks/paystack", body, sign("whsec", body))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("upgrade: %d %s", rec.Code, rec.Body.String())
	}

	got, err := st.GetWorkspace(ctx, wsp.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Plan != "pro" || got.PlanStatus != "active" {
		t.Fatalf("workspace %+v", got)
	}
	if got.PaystackCustomerCode != "cus_1" || got.PaystackSubscriptionCode != "sub_1" {
		t.Fatalf("codes %+v", got)
	}
	if got.PlanExpiresAt == nil || !got.PlanExpiresAt.After(base.Add(30*24*time.Hour)) {
		t.Fatalf("expires %v", got.PlanExpiresAt)
	}
	firstExpiry := *got.PlanExpiresAt

	// Double delivery of the same reference must be a no-op.
	now = base.Add(2 * time.Hour)
	rec = postWebhook(t, h, "/webhooks/paystack", body, sign("whsec", body))
	if rec.Code != http.StatusOK {
		t.Fatalf("duplicate: %d %s", rec.Code, rec.Body.String())
	}
	got, _ = st.GetWorkspace(ctx, wsp.ID)
	if !got.PlanExpiresAt.Equal(firstExpiry) {
		t.Fatalf("idempotent replay moved expiry: %v want %v", got.PlanExpiresAt, firstExpiry)
	}

	// Unknown events still answer 200 so Paystack stops retrying.
	rec = postWebhook(t, h, "/webhooks/paystack", []byte(`{"event":"transfer.failed","data":{}}`), sign("whsec", []byte(`{"event":"transfer.failed","data":{}}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("unknown event: %d", rec.Code)
	}
}

func TestWebhookCancelKeepsExpiry(t *testing.T) {
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	st, h := webhookSetup(t, &now)
	ctx := context.Background()
	_, wsp, err := st.UpsertGitHubUser(ctx, 2, "alice", "Alice", "a@example.com", "", false)
	if err != nil {
		t.Fatal(err)
	}
	upgrade := []byte(`{"event":"subscription.create","data":{"reference":"ref_2","amount":1500000,"currency":"NGN","subscription_code":"sub_2","customer":{"customer_code":"cus_2"},"metadata":{"workspace_id":"` + wsp.ID + `"}}}`)
	if rec := postWebhook(t, h, "/webhooks/paystack", upgrade, sign("whsec", upgrade)); rec.Code != http.StatusOK {
		t.Fatalf("upgrade: %d", rec.Code)
	}
	got, _ := st.GetWorkspace(ctx, wsp.ID)
	expiry := *got.PlanExpiresAt

	cancel := []byte(`{"event":"subscription.disable","data":{"reference":"ref_2","subscription_code":"sub_2","customer":{"customer_code":"cus_2"},"metadata":{"workspace_id":"` + wsp.ID + `"}}}`)
	if rec := postWebhook(t, h, "/webhooks/paystack", cancel, sign("whsec", cancel)); rec.Code != http.StatusOK {
		t.Fatalf("cancel: %d", rec.Code)
	}
	got, _ = st.GetWorkspace(ctx, wsp.ID)
	if got.PlanStatus != "cancelled" {
		t.Fatalf("status %q", got.PlanStatus)
	}
	if got.PlanExpiresAt == nil || !got.PlanExpiresAt.Equal(expiry) {
		t.Fatalf("expiry must stay intact: %v want %v", got.PlanExpiresAt, expiry)
	}
	// Cancel matching by customer code works too.
	cancel2 := []byte(`{"event":"subscription.not_renew","data":{"reference":"ref_2","customer":{"customer_code":"cus_2"}}}`)
	if rec := postWebhook(t, h, "/webhooks/paystack", cancel2, sign("whsec", cancel2)); rec.Code != http.StatusOK {
		t.Fatalf("cancel2: %d", rec.Code)
	}
}

func TestWebhookRejectsBadSignatures(t *testing.T) {
	st, h := webhookSetup(t, nil)
	ctx := context.Background()
	_, wsp, _ := st.UpsertGitHubUser(ctx, 3, "bob", "Bob", "b@example.com", "", false)

	body := []byte(`{"event":"charge.success","data":{"reference":"ref_x","metadata":{"workspace_id":"` + wsp.ID + `"}}}`)
	if rec := postWebhook(t, h, "/webhooks/paystack", body, "bad"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("bad header: %d", rec.Code)
	}
	if rec := postWebhook(t, h, "/webhooks/paystack", body, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("missing signature: %d", rec.Code)
	}
	got, _ := st.GetWorkspace(ctx, wsp.ID)
	if got.Plan != "" && got.Plan != "free" {
		t.Fatalf("workspace must not upgrade: %+v", got)
	}

	// The query fallback works in case a proxy strips custom headers.
	if rec := postWebhook(t, h, "/webhooks/paystack?secret=whsec", body, ""); rec.Code != http.StatusOK {
		t.Fatalf("query fallback: %d %s", rec.Code, rec.Body.String())
	}
	got, _ = st.GetWorkspace(ctx, wsp.ID)
	if got.Plan != "pro" {
		t.Fatalf("workspace %+v", got)
	}
}
