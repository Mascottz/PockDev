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

type sentWhatsApp struct {
	Channel string `json:"channel"`
	To      string `json:"to"`
	Content struct {
		Text string `json:"text"`
	} `json:"content"`
}

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

func newHook(t *testing.T, st *memstore.Mem, sent *[]sentWhatsApp) *Hook {
	t.Helper()
	notifySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body sentWhatsApp
		_ = json.NewDecoder(r.Body).Decode(&body)
		*sent = append(*sent, body)
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
	var sent []sentWhatsApp
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
	var sent []sentWhatsApp
	h := newHook(t, st, &sent)

	body := `{"from": {"phone": "+234801"}, "message": {"text": "2"}}`
	if rec := post(t, h, body, sign(body)); rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if len(sent) != 1 || sent[0].Channel != "whatsapp" || sent[0].To != "+234801" || !strings.Contains(sent[0].Content.Text, "Replaying 1 paused webhooks") {
		t.Fatalf("replies %+v", sent)
	}
	d, _ := st.GetDelivery(context.Background(), "dlv_1")
	if d.Status != model.DeliveryRetrying {
		t.Fatalf("delivery %s", d.Status)
	}
}

func TestReplyThreeMutes(t *testing.T) {
	st := memstore.New()
	seed(t, st)
	var sent []sentWhatsApp
	h := newHook(t, st, &sent)

	body := `{"from": "+234801", "text": "3"}`
	if rec := post(t, h, body, sign(body)); rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	inc, _ := st.GetIncident(context.Background(), "inc_1")
	if inc.Status != model.IncidentMuted {
		t.Fatalf("incident %s", inc.Status)
	}
	if len(sent) != 1 || sent[0].Channel != "whatsapp" || sent[0].To != "+234801" || !strings.Contains(sent[0].Content.Text, "Muted") {
		t.Fatalf("replies %+v", sent)
	}
}

func TestNonCommandChatIgnored(t *testing.T) {
	st := memstore.New()
	seed(t, st)
	var sent []sentWhatsApp
	h := newHook(t, st, &sent)

	body := `{"from": "+234801", "text": "hello"}`
	if rec := post(t, h, body, sign(body)); rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if len(sent) != 0 {
		t.Fatalf("expected 0 replies for non-command chat, got %+v", sent)
	}
}

func TestOpaqueLIDSenderResolvesToAlertedPhone(t *testing.T) {
	st := memstore.New()
	seed(t, st)
	var sent []sentWhatsApp
	h := newHook(t, st, &sent)

	body := `{"event":"message.received","data":{"from":"55693746942040","content":{"text":"3"}}}`
	if rec := post(t, h, body, sign(body)); rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	inc, _ := st.GetIncident(context.Background(), "inc_1")
	if inc.Status != model.IncidentMuted {
		t.Fatalf("incident status %s, want muted", inc.Status)
	}
	if len(sent) != 1 {
		t.Fatalf("expected 1 reply, got %+v", sent)
	}
	if sent[0].Channel != "whatsapp" || sent[0].To != "+234801" {
		t.Fatalf("expected reply routed to LastAlertPhone +234801, got %+v", sent[0])
	}
	if !strings.Contains(sent[0].Content.Text, "Muted") {
		t.Fatalf("unexpected reply text: %q", sent[0].Content.Text)
	}
}

func TestShortTagMatchesAcrossMultiplePhones(t *testing.T) {
	st := memstore.New()
	now := time.Now()
	alertAt := now.Add(-5 * time.Minute)

	_ = st.CreateProject(context.Background(), &model.Project{
		ID: "prj_a", Name: "alpha-api", Slug: "alpha-api", EndpointKey: "ek_a",
		SDKKey: "pk_a", WhatsAppPhone: "+234801", Status: "active", CreatedAt: now,
	})
	_ = st.CreateProject(context.Background(), &model.Project{
		ID: "prj_b", Name: "beta-api", Slug: "beta-api", EndpointKey: "ek_b",
		SDKKey: "pk_b", WhatsAppPhone: "+234802", Status: "active", CreatedAt: now,
	})
	_ = st.CreateIncident(context.Background(), &model.Incident{
		ID: "inc_alpha_a1b2", ProjectID: "prj_a", Title: "Alpha crash",
		Status: model.IncidentOpen, LastAlertPhone: "+234801", LastAlertAt: &alertAt,
		LastSeenAt: now, CreatedAt: now,
	})
	_ = st.CreateIncident(context.Background(), &model.Incident{
		ID: "inc_beta_c3d4", ProjectID: "prj_b", Title: "Beta crash",
		Status: model.IncidentOpen, LastAlertPhone: "+234802", LastAlertAt: &alertAt,
		LastSeenAt: now, CreatedAt: now,
	})
	_ = st.CreateDelivery(context.Background(), &model.Delivery{
		ID: "dlv_a", ProjectID: "prj_a", Status: model.DeliveryPaused,
		LinkedIncidentID: "inc_alpha_a1b2", CreatedAt: now,
	})
	_ = st.CreateDelivery(context.Background(), &model.Delivery{
		ID: "dlv_b", ProjectID: "prj_b", Status: model.DeliveryPaused,
		LinkedIncidentID: "inc_beta_c3d4", CreatedAt: now,
	})

	var sent []sentWhatsApp
	h := newHook(t, st, &sent)

	// Without a 4-char tag, an unknown LID with multiple active phones sends 0 replies.
	ambiguous := `{"event":"message.received","data":{"from":"99887766554433","content":{"text":"2"}}}`
	if rec := post(t, h, ambiguous, sign(ambiguous)); rec.Code != http.StatusOK {
		t.Fatalf("ambiguous status %d", rec.Code)
	}
	if len(sent) != 0 {
		t.Fatalf("expected 0 replies for ambiguous LID, got %+v", sent)
	}

	// With the 4-char tag "c3d4", it matches inc_beta_c3d4 and replies to +234802.
	tagged := `{"event":"message.received","data":{"from":"99887766554433","content":{"text":"2 c3d4"}}}`
	if rec := post(t, h, tagged, sign(tagged)); rec.Code != http.StatusOK {
		t.Fatalf("tagged status %d: %s", rec.Code, rec.Body.String())
	}
	if len(sent) != 1 || sent[0].Channel != "whatsapp" || sent[0].To != "+234802" || !strings.Contains(sent[0].Content.Text, "Replaying 1 paused webhooks") {
		t.Fatalf("tagged replies %+v", sent)
	}
	dlvA, _ := st.GetDelivery(context.Background(), "dlv_a")
	dlvB, _ := st.GetDelivery(context.Background(), "dlv_b")
	if dlvA.Status != model.DeliveryPaused {
		t.Fatalf("dlv_a should stay paused, got %s", dlvA.Status)
	}
	if dlvB.Status != model.DeliveryRetrying {
		t.Fatalf("dlv_b should be retrying, got %s", dlvB.Status)
	}
}

func TestInboundShapes(t *testing.T) {
	cases := []struct {
		body, from, text string
	}{
		{`{"from": "+1", "text": "hi"}`, "+1", "hi"},
		{`{"from": {"phone": "+2"}, "message": {"text": "yo"}}`, "+2", "yo"},
		{`{"from": "55693746942040", "content": {"text": "3"}}`, "55693746942040", "3"},
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
