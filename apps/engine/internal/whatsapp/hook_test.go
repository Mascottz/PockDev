package whatsapp

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Mascottz/PockDev/apps/engine/internal/agent"
	"github.com/Mascottz/PockDev/apps/engine/internal/model"
	"github.com/Mascottz/PockDev/apps/engine/internal/notify"
	"github.com/Mascottz/PockDev/apps/engine/internal/store/memstore"
)

const testSecret = "rimbra-secret"

func sign(body string) string {
	mac := hmac.New(sha256.New, []byte(testSecret))
	mac.Write([]byte(body))
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func seed(t *testing.T, st *memstore.Mem) {
	t.Helper()
	now := time.Now()
	alertAt := now.Add(-10 * time.Minute)
	_ = st.CreateProject(context.Background(), &model.Project{
		ID: "prj_1", Name: "orders-api", Slug: "orders-api", EndpointKey: "ek",
		SDKKey: "pk", GitHubRepo: "acme/orders", WhatsAppPhone: "+234801",
		Status: "active", CreatedAt: now,
	})
	_ = st.CreateIncident(context.Background(), &model.Incident{
		ID: "inc_1", ProjectID: "prj_1", Title: "Crash in orders",
		FilePath: "src/orders.ts", Origin: model.OriginErrorSpike,
		Status: model.IncidentOpen, LastAlertPhone: "+234801", LastAlertAt: &alertAt,
		LastSeenAt: now, CreatedAt: now,
	})
	_ = st.CreateDelivery(context.Background(), &model.Delivery{
		ID: "dlv_1", ProjectID: "prj_1", Status: model.DeliveryPaused,
		LinkedIncidentID: "inc_1", CreatedAt: now,
	})
}

func newHook(t *testing.T, st *memstore.Mem, sent *[]string) *Hook {
	t.Helper()
	notifySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		*sent = append(*sent, body["message"])
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(notifySrv.Close)
	return &Hook{
		Store:  st,
		Agent:  &agent.Agent{Store: st, MuteFor: time.Hour},
		Notify: &notify.Client{APIKey: "k", WhatsAppURL: notifySrv.URL},
		Secret: testSecret,
	}
}

func post(t *testing.T, h *Hook, body, sig string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/webhooks/whatsapp", strings.NewReader(body))
	if sig != "" {
		req.Header.Set("X-Rimbra-Signature", sig)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestSignatureRequired(t *testing.T) {
	st := memstore.New()
	seed(t, st)
	var sent []string
	h := newHook(t, st, &sent)

	body := `{"from": "+234801", "text": "2"}`
	if rec := post(t, h, body, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("no sig: %d", rec.Code)
	}
	if rec := post(t, h, body, "sha256=deadbeef"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("bad sig: %d", rec.Code)
	}
	if rec := post(t, h, body, sign(body)); rec.Code != http.StatusOK {
		t.Fatalf("good sig: %d %s", rec.Code, rec.Body.String())
	}
}

func TestReplyTwoReplays(t *testing.T) {
	st := memstore.New()
	seed(t, st)
	var sent []string
	h := newHook(t, st, &sent)

	body := `{"from": {"phone": "+234801"}, "message": {"text": "2"}}`
	if rec := post(t, h, body, sign(body)); rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if len(sent) != 1 || !strings.Contains(sent[0], "Replaying 1 paused webhooks") {
		t.Fatalf("replies %v", sent)
	}
	d, _ := st.GetDelivery(context.Background(), "dlv_1")
	if d.Status != model.DeliveryRetrying {
		t.Fatalf("delivery %s", d.Status)
	}
}

func TestReplyThreeMutes(t *testing.T) {
	st := memstore.New()
	seed(t, st)
	var sent []string
	h := newHook(t, st, &sent)

	body := `{"from": "+234801", "text": "3"}`
	if rec := post(t, h, body, sign(body)); rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	inc, _ := st.GetIncident(context.Background(), "inc_1")
	if inc.Status != model.IncidentMuted {
		t.Fatalf("incident %s", inc.Status)
	}
	if len(sent) != 1 || !strings.Contains(sent[0], "Muted") {
		t.Fatalf("replies %v", sent)
	}
}

func TestUnknownReplyGetsMenu(t *testing.T) {
	st := memstore.New()
	seed(t, st)
	var sent []string
	h := newHook(t, st, &sent)

	body := `{"from": "+234801", "text": "hello"}`
	if rec := post(t, h, body, sign(body)); rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if len(sent) != 1 || !strings.Contains(sent[0], "Reply 1 to open a fix PR") {
		t.Fatalf("replies %v", sent)
	}
}

func TestUnknownNumber(t *testing.T) {
	st := memstore.New()
	seed(t, st)
	var sent []string
	h := newHook(t, st, &sent)

	body := `{"from": "+1000", "text": "1"}`
	if rec := post(t, h, body, sign(body)); rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if len(sent) != 1 || !strings.Contains(sent[0], "do not have an open alert") {
		t.Fatalf("replies %v", sent)
	}
}

func TestInboundShapes(t *testing.T) {
	cases := []struct {
		body, from, text string
	}{
		{`{"from": "+1", "text": "hi"}`, "+1", "hi"},
		{`{"from": {"phone": "+2"}, "message": {"text": "yo"}}`, "+2", "yo"},
		{`{"phone": "+3", "body": "sup"}`, "+3", "sup"},
	}
	for _, c := range cases {
		var in Inbound
		if err := json.Unmarshal([]byte(c.body), &in); err != nil {
			t.Fatal(err)
		}
		if in.From != c.from || in.Text != c.text {
			t.Fatalf("%s -> %+v", c.body, in)
		}
	}
}
