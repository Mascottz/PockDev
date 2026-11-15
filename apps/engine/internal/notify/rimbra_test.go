package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendWhatsAppAndEmail(t *testing.T) {
	type whatsappPayload struct {
		Channel string `json:"channel"`
		To      string `json:"to"`
		Content struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	type emailPayload struct {
		Subject string `json:"subject"`
		Text    string `json:"text"`
	}
	var (
		gotWA    whatsappPayload
		gotEmail emailPayload
		paths    []string
		auths    []string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		auths = append(auths, r.Header.Get("Authorization"))
		switch r.URL.Path {
		case "/whatsapp":
			_ = json.NewDecoder(r.Body).Decode(&gotWA)
		case "/email":
			_ = json.NewDecoder(r.Body).Decode(&gotEmail)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := &Client{APIKey: "rk", WhatsAppURL: srv.URL + "/whatsapp", EmailURL: srv.URL + "/email"}
	if !c.WhatsAppConfigured() || !c.EmailConfigured() {
		t.Fatal("should be configured")
	}
	if err := c.SendWhatsApp(context.Background(), "234", "hello"); err != nil {
		t.Fatal(err)
	}
	if err := c.SendEmail(context.Background(), "subj", "body"); err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 || paths[0] != "/whatsapp" || paths[1] != "/email" {
		t.Fatalf("paths %v", paths)
	}
	if gotWA.Channel != "whatsapp" || gotWA.To != "+234" || gotWA.Content.Text != "hello" {
		t.Fatalf("whatsapp %+v", gotWA)
	}
	if gotEmail.Subject != "subj" || gotEmail.Text != "body" {
		t.Fatalf("email %+v", gotEmail)
	}
	for _, auth := range auths {
		if auth != "Bearer rk" {
			t.Fatalf("auth %q", auth)
		}
	}
}

func TestSendNotConfigured(t *testing.T) {
	c := &Client{}
	if c.WhatsAppConfigured() || c.EmailConfigured() {
		t.Fatal("should not be configured")
	}
	if err := c.SendWhatsApp(context.Background(), "+1", "x"); err == nil {
		t.Fatal("expected error")
	}
	if err := c.SendEmail(context.Background(), "s", "b"); err == nil {
		t.Fatal("expected error")
	}
}

func TestSendErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()
	c := &Client{APIKey: "k", WhatsAppURL: srv.URL}
	if err := c.SendWhatsApp(context.Background(), "+1", "x"); err == nil {
		t.Fatal("expected error")
	}
}
