// Package billing talks to the Paystack payment API and verifies its webhooks.
package billing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Client calls the Paystack REST API.
type Client struct {
	SecretKey  string
	BaseURL    string
	PlanCode   string
	HTTPClient *http.Client
}

// NewClient builds a Paystack client.
func NewClient(secretKey, baseURL, planCode string) *Client {
	return &Client{
		SecretKey: secretKey,
		BaseURL:   strings.TrimSuffix(baseURL, "/"),
		PlanCode:  planCode,
	}
}

func (c *Client) base() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return "https://api.paystack.co"
}

func (c *Client) client() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 30 * time.Second}
}

// Configured reports whether a secret key is present.
func (c *Client) Configured() bool {
	return strings.TrimSpace(c.SecretKey) != ""
}

// InitializeResult is the checkout handoff returned to the dashboard.
type InitializeResult struct {
	AuthorizationURL string `json:"authorization_url"`
	Reference        string `json:"reference"`
}

// InitializeInput describes one Pro checkout.
type InitializeInput struct {
	Email        string
	AmountKobo   int64
	Currency     string
	CallbackURL  string
	WorkspaceID  string
}

// InitializeTransaction opens a Paystack checkout for the Pro plan.
func (c *Client) InitializeTransaction(ctx context.Context, in InitializeInput) (*InitializeResult, error) {
	payload := map[string]any{
		"email":        in.Email,
		"amount":       in.AmountKobo,
		"currency":     in.Currency,
		"callback_url": in.CallbackURL,
		"metadata": map[string]any{
			"workspace_id": in.WorkspaceID,
			"plan":         "pro",
		},
	}
	if c.PlanCode != "" {
		payload["plan"] = c.PlanCode
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base()+"/transaction/initialize", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.SecretKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := c.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out struct {
		Status bool              `json:"status"`
		Data   InitializeResult  `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if !out.Status || out.Data.AuthorizationURL == "" || out.Data.Reference == "" {
		return nil, fmt.Errorf("paystack initialize failed with status %d", resp.StatusCode)
	}
	return &out.Data, nil
}

// VerifyResult is the transaction state returned by Paystack.
type VerifyResult struct {
	Status         string `json:"status"`
	Reference      string `json:"reference"`
	Amount         int64  `json:"amount"`
	Currency       string `json:"currency"`
	SubscriptionCode string `json:"subscription_code"`
	Customer       struct {
		CustomerCode string `json:"customer_code"`
	} `json:"customer"`
	Metadata struct {
		WorkspaceID string `json:"workspace_id"`
	} `json:"metadata"`
}

// VerifyTransaction loads the state of one transaction reference.
func (c *Client) VerifyTransaction(ctx context.Context, reference string) (*VerifyResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.base()+"/transaction/verify/"+reference, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.SecretKey)
	req.Header.Set("Accept", "application/json")
	resp, err := c.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out struct {
		Status bool          `json:"status"`
		Data   VerifyResult  `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if !out.Status {
		return nil, fmt.Errorf("paystack verify failed with status %d", resp.StatusCode)
	}
	return &out.Data, nil
}
