package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendWhatsAppAndEmail(t *testing.T) {
	type rec struct {
		path, auth string
		payload    map[string]string
	}
	var got []rec
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]string
		_ = json.NewDecoder(r.Body).Decode(&payload)
		got = append(got, rec{path: r.URL.Path, auth: r.Header.Get("Authorization"), payload: payload})
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := &Client{APIKey: "rk", WhatsAppURL: srv.URL + "/whatsapp", EmailURL: srv.URL + "/email"}
	if !c.WhatsAppConfigured() || !c.EmailConfigured() {
		t.Fatal("should be configured")
	}
	if err := c.SendWhatsApp(context.Background(), "+234", "hello"); err != nil {
		t.Fatal(err)
	}
	if err := c.SendEmail(context.Background(), "subj", "body"); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %v", got)
	}
	if got[0].path != "/whatsapp" || got[0].payload["to"] != "+234" || got[0].payload["message"] != "hello" {
		t.Fatalf("whatsapp %v", got[0])
	}
	if got[1].path != "/email" || got[1].payload["subject"] != "subj" || got[1].payload["text"] != "body" {
		t.Fatalf("email %v", got[1])
	}
	for _, r := range got {
		if r.auth != "Bearer rk" {
			t.Fatalf("auth %q", r.auth)
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
