package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Mascottz/PockDev/apps/engine/internal/billing"
	"github.com/Mascottz/PockDev/apps/engine/internal/model"
	"github.com/Mascottz/PockDev/apps/engine/internal/store"
	"github.com/Mascottz/PockDev/apps/engine/internal/store/memstore"
)

// billingSetup builds an API in token mode against a fresh memstore with the
// default workspace.
func billingSetup(t *testing.T) (*API, *memstore.Mem, http.Handler) {
	t.Helper()
	st := memstore.New()
	a := &API{
		Store:            st,
		APIToken:         testToken,
		ProPriceKobo:     1500000,
		PaystackCurrency: "NGN",
		StartedAt:        time.Now(),
	}
	mux := http.NewServeMux()
	a.Register(mux)
	return a, st, mux
}

func hashTokenForTest(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func TestProjectQuotaFreeAndPro(t *testing.T) {
	_, st, h := billingSetup(t)

	// Free plan: two projects, then 402.
	for i := 0; i < 2; i++ {
		rec := do(t, h, http.MethodPost, "/api/projects", testToken, `{"name": "App `+strconv.Itoa(i)+`"}`)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create %d: %d %s", i, rec.Code, rec.Body.String())
		}
	}
	rec := do(t, h, http.MethodPost, "/api/projects", testToken, `{"name": "App 3"}`)
	if rec.Code != http.StatusPaymentRequired {
		t.Fatalf("third project: %d %s", rec.Code, rec.Body.String())
	}
	var errBody map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &errBody)
	if errBody["code"] != "quota_exceeded" {
		t.Fatalf("body %s", rec.Body.String())
	}
	if errBody["error"] != "Free plan includes up to 2 watched apps. Upgrade to Pro for up to 20 apps." {
		t.Fatalf("message %q", errBody["error"])
	}

	// Upgrade to Pro: more projects flow again.
	if err := st.UpgradeWorkspacePro(context.Background(), store.DefaultWorkspaceID, "ref_upgrade",
		"cus_1", "sub_1", "NGN", 1500000, time.Now().Add(31*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 18; i++ {
		rec := do(t, h, http.MethodPost, "/api/projects", testToken, `{"name": "Pro App `+strconv.Itoa(i)+`"}`)
		if rec.Code != http.StatusCreated {
			t.Fatalf("pro create %d: %d %s", i, rec.Code, rec.Body.String())
		}
	}
	rec = do(t, h, http.MethodPost, "/api/projects", testToken, `{"name": "One Too Many"}`)
	if rec.Code != http.StatusPaymentRequired {
		t.Fatalf("21st project: %d %s", rec.Code, rec.Body.String())
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &errBody)
	if errBody["code"] != "quota_exceeded" || errBody["error"] != "Pro plan includes up to 20 watched apps." {
		t.Fatalf("body %s", rec.Body.String())
	}
}

func TestBillingPayloadShape(t *testing.T) {
	_, st, h := billingSetup(t)
	ctx := context.Background()
	_ = st.CreateProject(ctx, &model.Project{
		ID: "prj_1", Name: "n", Slug: "n", EndpointKey: "e", SDKKey: "s",
		Status: "active", CreatedAt: time.Now(),
	})

	rec := do(t, h, http.MethodGet, "/api/billing", testToken, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("billing: %d %s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["plan"] != "free" || out["planStatus"] != "active" || out["planExpiresAt"] != nil {
		t.Fatalf("plan fields %s", rec.Body.String())
	}
	if out["paystackConfigured"] != false || out["currency"] != "NGN" || out["proPriceKobo"].(float64) != 1500000 {
		t.Fatalf("billing fields %s", rec.Body.String())
	}
	usage := out["usage"].(map[string]any)
	if usage["projectsUsed"].(float64) != 1 || usage["projectsLimit"].(float64) != 2 ||
		usage["monthlyEventsLimit"].(float64) != 5000 || usage["retentionDays"].(float64) != 3 {
		t.Fatalf("usage %v", usage)
	}
}

// fakePaystack serves the two Paystack endpoints the dashboard calls.
func fakePaystack(t *testing.T, verifyWorkspace string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /transaction/initialize", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if r.Header.Get("Authorization") != "Bearer sk_test_123" {
			t.Errorf("auth header %q", r.Header.Get("Authorization"))
		}
		var payload map[string]any
		_ = json.Unmarshal(body, &payload)
		if payload["email"] == "" || payload["amount"].(float64) != 1500000 || payload["currency"] != "NGN" {
			t.Errorf("initialize payload %v", payload)
		}
		meta := payload["metadata"].(map[string]any)
		if meta["plan"] != "pro" || meta["workspace_id"] == "" {
			t.Errorf("metadata %v", meta)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"data":{"authorization_url":"https://pay.stack/checkout","reference":"ref_9"}}`))
	})
	mux.HandleFunc("GET /transaction/verify/ref_9", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"data":{"status":"success","reference":"ref_9","amount":1500000,"currency":"NGN","subscription_code":"sub_9","customer":{"customer_code":"cus_9"},"metadata":{"workspace_id":"` + verifyWorkspace + `"}}}`))
	})
	mux.HandleFunc("GET /transaction/verify/ref_fail", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"data":{"status":"failed","reference":"ref_fail","metadata":{"workspace_id":"` + verifyWorkspace + `"}}}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestBillingCheckoutAndVerify(t *testing.T) {
	st := memstore.New()
	gh := fakePaystack(t, store.DefaultWorkspaceID)
	a := &API{
		Store:            st,
		APIToken:         testToken,
		Billing:          billing.NewClient("sk_test_123", gh.URL, ""),
		ProPriceKobo:     1500000,
		PaystackCurrency: "NGN",
		StartedAt:        time.Now(),
	}
	mux := http.NewServeMux()
	a.Register(mux)

	// Checkout with the default callback URL.
	rec := do(t, mux, http.MethodPost, "/api/billing/checkout", testToken, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("checkout: %d %s", rec.Code, rec.Body.String())
	}
	var out map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["authorizationUrl"] != "https://pay.stack/checkout" || out["reference"] != "ref_9" {
		t.Fatalf("checkout %v", out)
	}

	// Untrusted callback origins are refused.
	rec = do(t, mux, http.MethodPost, "/api/billing/checkout", testToken, `{"callbackUrl": "https://evil.com/x"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("evil callback: %d", rec.Code)
	}
	// Trusted explicit callback passes.
	rec = do(t, mux, http.MethodPost, "/api/billing/checkout", testToken, `{"callbackUrl": "https://pockdev.vercel.app/deck?billing=verify"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("trusted callback: %d %s", rec.Code, rec.Body.String())
	}

	// Failed payments report 402 and do not upgrade.
	rec = do(t, mux, http.MethodPost, "/api/billing/verify", testToken, `{"reference": "ref_fail"}`)
	if rec.Code != http.StatusPaymentRequired {
		t.Fatalf("failed payment: %d %s", rec.Code, rec.Body.String())
	}

	// Verify upgrades and answers with the billing payload.
	rec = do(t, mux, http.MethodPost, "/api/billing/verify", testToken, `{"reference": "ref_9"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("verify: %d %s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &payload)
	if payload["plan"] != "pro" || payload["planStatus"] != "active" {
		t.Fatalf("payload %s", rec.Body.String())
	}
	usage := payload["usage"].(map[string]any)
	if usage["projectsLimit"].(float64) != 20 || usage["monthlyEventsLimit"].(float64) != 100000 || usage["retentionDays"].(float64) != 30 {
		t.Fatalf("usage %v", usage)
	}
	wsp, err := st.GetWorkspace(context.Background(), store.DefaultWorkspaceID)
	if err != nil {
		t.Fatal(err)
	}
	if wsp.Plan != "pro" || wsp.PaystackCustomerCode != "cus_9" || wsp.PaystackSubscriptionCode != "sub_9" {
		t.Fatalf("workspace %+v", wsp)
	}

	// Missing reference is a 400.
	rec = do(t, mux, http.MethodPost, "/api/billing/verify", testToken, `{}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing reference: %d", rec.Code)
	}
}

func TestBillingVerifyWorkspaceMismatch(t *testing.T) {
	st := memstore.New()
	// The fake Paystack stamps metadata with a foreign workspace id.
	gh := fakePaystack(t, "wsp_not_mine")
	a := &API{
		Store:            st,
		APIToken:         testToken,
		Billing:          billing.NewClient("sk_test_123", gh.URL, ""),
		ProPriceKobo:     1500000,
		PaystackCurrency: "NGN",
		StartedAt:        time.Now(),
	}
	mux := http.NewServeMux()
	a.Register(mux)

	rec := do(t, mux, http.MethodPost, "/api/billing/verify", testToken, `{"reference": "ref_9"}`)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("mismatch: %d %s", rec.Code, rec.Body.String())
	}
	wsp, _ := st.GetWorkspace(context.Background(), store.DefaultWorkspaceID)
	if wsp.Plan == "pro" {
		t.Fatalf("workspace must not upgrade: %+v", wsp)
	}
}

func TestBillingCheckoutSessionEmailFallback(t *testing.T) {
	st := memstore.New()
	user, wsp, err := st.UpsertGitHubUser(context.Background(), 4004, "carol", "Carol", "", "", false)
	if err != nil {
		t.Fatal(err)
	}
	raw := "pds_" + strings.Repeat("c", 48)
	if err := st.CreateSession(context.Background(), &model.Session{
		TokenHash: hashTokenForTest(raw), UserID: user.ID, WorkspaceID: wsp.ID,
		ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	var gotEmail, gotWorkspace string
	gh := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		_ = json.Unmarshal(body, &payload)
		gotEmail, _ = payload["email"].(string)
		gotWorkspace, _ = payload["metadata"].(map[string]any)["workspace_id"].(string)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"data":{"authorization_url":"https://pay.stack/x","reference":"ref_e"}}`))
	}))
	t.Cleanup(gh.Close)

	a := &API{
		Store:            st,
		Billing:          billing.NewClient("sk_test_123", gh.URL, "PLN_pro"),
		ProPriceKobo:     1500000,
		PaystackCurrency: "NGN",
		StartedAt:        time.Now(),
	}
	mux := http.NewServeMux()
	a.Register(mux)

	rec := do(t, mux, http.MethodPost, "/api/billing/checkout", raw, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("checkout: %d %s", rec.Code, rec.Body.String())
	}
	if gotEmail != "carol@users.noreply.github.com" {
		t.Fatalf("email fallback %q", gotEmail)
	}
	if gotWorkspace != wsp.ID {
		t.Fatalf("workspace %q want %q", gotWorkspace, wsp.ID)
	}
}
