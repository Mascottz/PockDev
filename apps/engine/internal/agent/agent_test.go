package agent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Mascottz/PockDev/apps/engine/internal/model"
	"github.com/Mascottz/PockDev/apps/engine/internal/notify"
	"github.com/Mascottz/PockDev/apps/engine/internal/store/memstore"
)

func seedIncident(t *testing.T, st *memstore.Mem) *model.Incident {
	t.Helper()
	now := time.Now()
	p := &model.Project{
		ID: "prj_1", Name: "orders-api", Slug: "orders-api", EndpointKey: "ek",
		SDKKey: "pk", GitHubRepo: "acme/orders", WhatsAppPhone: "+234801",
		Status: "active", CreatedAt: now,
	}
	if err := st.CreateProject(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	inc := &model.Incident{
		ID: "inc_1", ProjectID: "prj_1", Title: "Cannot read rider.phone",
		FilePath: "src/orders.ts", Origin: model.OriginErrorSpike,
		Status: model.IncidentOpen, LastSeenAt: now, CreatedAt: now,
	}
	if err := st.CreateIncident(context.Background(), inc); err != nil {
		t.Fatal(err)
	}
	_ = st.CreateDelivery(context.Background(), &model.Delivery{
		ID: "dlv_1", ProjectID: "prj_1", Status: model.DeliveryPaused,
		LinkedIncidentID: "inc_1", StorageKey: "k", CreatedAt: now,
	})
	_ = st.CreateDelivery(context.Background(), &model.Delivery{
		ID: "dlv_2", ProjectID: "prj_1", Status: model.DeliveryPaused,
		LinkedIncidentID: "inc_1", StorageKey: "k", CreatedAt: now,
	})
	return inc
}

func TestDiagnoseAlertsWithFallback(t *testing.T) {
	st := memstore.New()
	seedIncident(t, st)

	var whatsappBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer rimbra-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&whatsappBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a := &Agent{
		Store:  st,
		Notify: &notify.Client{APIKey: "rimbra-key", WhatsAppURL: srv.URL, EmailURL: srv.URL},
	}
	if err := a.Diagnose(context.Background(), "inc_1"); err != nil {
		t.Fatal(err)
	}

	if whatsappBody["to"] != "+234801" {
		t.Fatalf("to %v", whatsappBody)
	}
	msg := whatsappBody["message"]
	if !strings.Contains(msg, "orders-api") || !strings.Contains(msg, "2 webhooks are safely paused") {
		t.Fatalf("message %q", msg)
	}
	if !strings.Contains(msg, "Reply 1 to open a fix PR, 2 to replay paused webhooks, or 3 to mute for 1 hour.") {
		t.Fatalf("menu missing: %q", msg)
	}

	inc, err := st.GetIncident(context.Background(), "inc_1")
	if err != nil {
		t.Fatal(err)
	}
	if inc.LastAlertPhone != "+234801" || inc.PausedCount != 2 {
		t.Fatalf("incident %+v", inc)
	}
}

func TestDiagnoseUsesLLM(t *testing.T) {
	st := memstore.New()
	seedIncident(t, st)

	llm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		reply := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "The orders service crashed while reading the rider phone number after the order was cancelled."}},
			},
		}
		_ = json.NewEncoder(w).Encode(reply)
	}))
	defer llm.Close()

	var msg string
	notifySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		msg = body["message"]
		w.WriteHeader(http.StatusOK)
	}))
	defer notifySrv.Close()

	a := &Agent{
		Store:  st,
		LLM:    &LLM{BaseURL: llm.URL, Model: "test"},
		Notify: &notify.Client{APIKey: "k", WhatsAppURL: notifySrv.URL},
	}
	if err := a.Diagnose(context.Background(), "inc_1"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(msg, "rider phone number") {
		t.Fatalf("llm summary missing: %q", msg)
	}
}

func TestMuteResolve(t *testing.T) {
	st := memstore.New()
	seedIncident(t, st)
	a := &Agent{Store: st, MuteFor: time.Hour}

	inc, err := a.Mute(context.Background(), "inc_1")
	if err != nil {
		t.Fatal(err)
	}
	if inc.Status != model.IncidentMuted || inc.MuteUntil == nil {
		t.Fatalf("muted %+v", inc)
	}
	inc, err = a.Resolve(context.Background(), "inc_1")
	if err != nil {
		t.Fatal(err)
	}
	if inc.Status != model.IncidentResolved {
		t.Fatalf("resolved %+v", inc)
	}
}

func TestReplayAllAction(t *testing.T) {
	st := memstore.New()
	seedIncident(t, st)
	a := &Agent{Store: st}
	inc, n, err := a.ReplayAll(context.Background(), "inc_1")
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 || inc.PausedCount != 0 {
		t.Fatalf("replay %d %+v", n, inc)
	}
}

func TestDraftPR(t *testing.T) {
	st := memstore.New()
	seedIncident(t, st)

	var branchCreated, prOpened, committed string
	gh := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/acme/orders":
			_ = json.NewEncoder(w).Encode(map[string]string{"default_branch": "main"})
		case r.Method == http.MethodGet && r.URL.Path == "/repos/acme/orders/git/ref/heads/main":
			_ = json.NewEncoder(w).Encode(map[string]any{"object": map[string]string{"sha": "abc123"}})
		case r.Method == http.MethodPost && r.URL.Path == "/repos/acme/orders/git/refs":
			branchCreated = "refs/heads"
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{})
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/repos/acme/orders/contents/"):
			if r.URL.RawQuery != "" {
				// CommitFile sha lookup on the fresh branch: file does not exist yet.
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Write([]byte("const phone = rider.phone;\n"))
		case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/repos/acme/orders/contents/"):
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			committed = body["content"]
			_ = json.NewEncoder(w).Encode(map[string]any{})
		case r.Method == http.MethodPost && r.URL.Path == "/repos/acme/orders/pulls":
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			if !strings.HasPrefix(body["title"], "Mascott, ") {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "bad title"})
				return
			}
			prOpened = body["head"]
			_ = json.NewEncoder(w).Encode(map[string]string{"html_url": "https://github.com/acme/orders/pull/7"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer gh.Close()

	llm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reply := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "Here is the fix:\n```typescript\nconst phone = rider && rider.phone;\n```\n"}},
			},
		}
		_ = json.NewEncoder(w).Encode(reply)
	}))
	defer llm.Close()

	a := &Agent{
		Store:  st,
		LLM:    &LLM{BaseURL: llm.URL, Model: "test"},
		GitHub: &GitHub{Token: "ghp_test", BaseURL: gh.URL},
	}
	inc, err := a.DraftPR(context.Background(), "inc_1")
	if err != nil {
		t.Fatal(err)
	}
	if inc.Status != model.IncidentFixing {
		t.Fatalf("status %s", inc.Status)
	}
	if inc.PRURL != "https://github.com/acme/orders/pull/7" {
		t.Fatalf("pr %q", inc.PRURL)
	}
	if branchCreated == "" || prOpened == "" {
		t.Fatal("github flow incomplete")
	}
	raw, err := base64.StdEncoding.DecodeString(committed)
	if err != nil || !strings.Contains(string(raw), "rider && rider.phone") {
		t.Fatalf("committed content %q err %v", raw, err)
	}
}

func TestExtractCodeBlock(t *testing.T) {
	got := extractCodeBlock("text\n```go\nfmt.Println(1)\n```\nmore")
	if got != "fmt.Println(1)\n" {
		t.Fatalf("got %q", got)
	}
	if extractCodeBlock("no block") != "" {
		t.Fatal("expected empty")
	}
}
