// Package notify sends alerts through Rimbra WhatsApp and Email APIs.
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client wraps the Rimbra messaging endpoints.
type Client struct {
	APIKey      string
	WhatsAppURL string
	EmailURL    string
	HTTP        *http.Client
}

// WhatsAppConfigured reports whether WhatsApp alerts can be sent.
func (c *Client) WhatsAppConfigured() bool {
	return c != nil && c.APIKey != "" && c.WhatsAppURL != ""
}

// EmailConfigured reports whether email alerts can be sent.
func (c *Client) EmailConfigured() bool {
	return c != nil && c.APIKey != "" && c.EmailURL != ""
}

// SendWhatsApp posts one message through the Rimbra WhatsApp API.
func (c *Client) SendWhatsApp(ctx context.Context, to, message string) error {
	if !c.WhatsAppConfigured() {
		return fmt.Errorf("Rimbra WhatsApp is not configured")
	}
	return c.post(ctx, c.WhatsAppURL, map[string]string{
		"to":      to,
		"message": message,
	})
}

// SendEmail posts one email through the Rimbra Email API.
func (c *Client) SendEmail(ctx context.Context, subject, body string) error {
	if !c.EmailConfigured() {
		return fmt.Errorf("Rimbra Email is not configured")
	}
	return c.post(ctx, c.EmailURL, map[string]string{
		"subject": subject,
		"text":    body,
	})
}

func (c *Client) post(ctx context.Context, url string, payload map[string]string) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSuffix(url, "/"), bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("rimbra %s: %s %s", url, resp.Status, string(body))
	}
	return nil
}
