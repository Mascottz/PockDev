package config

import "testing"

func TestValidateNeedsOneAuthMode(t *testing.T) {
	c := Config{DatabaseURL: "postgres://db"}
	if err := c.Validate(); err == nil {
		t.Fatal("no auth configured must fail")
	}

	c.APIToken = "tok"
	if err := c.Validate(); err != nil {
		t.Fatalf("API_TOKEN alone must pass: %v", err)
	}

	gh := Config{DatabaseURL: "postgres://db", GitHubClientID: "id", GitHubClientSecret: "secret"}
	if err := gh.Validate(); err != nil {
		t.Fatalf("GitHub OAuth alone must pass: %v", err)
	}

	none := Config{GitHubClientID: "id"}
	if err := none.Validate(); err == nil {
		t.Fatal("missing database must still fail")
	}
}

func TestGitHubAuthConfigured(t *testing.T) {
	cases := []struct {
		id, secret string
		want       bool
	}{
		{"", "", false},
		{"id", "", false},
		{"", "secret", false},
		{"id", "secret", true},
	}
	for _, c := range cases {
		got := Config{GitHubClientID: c.id, GitHubClientSecret: c.secret}.GitHubAuthConfigured()
		if got != c.want {
			t.Fatalf("GitHubAuthConfigured(%q, %q) = %v want %v", c.id, c.secret, got, c.want)
		}
	}
}

func TestPaystackConfigured(t *testing.T) {
	cases := []struct {
		secret, plan string
		want         bool
	}{
		{"", "", false},
		{"sk_live_x", "", true},
		{"", "PLN_x", false},
		{"sk_live_x", "PLN_x", true},
	}
	for _, c := range cases {
		got := Config{PaystackSecretKey: c.secret, PaystackPlanCode: c.plan}.PaystackConfigured()
		if got != c.want {
			t.Fatalf("PaystackConfigured(%q, %q) = %v want %v", c.secret, c.plan, got, c.want)
		}
	}
}

func TestPaystackLoadDefaults(t *testing.T) {
	t.Setenv("PAYSTACK_SECRET_KEY", "sk_test_9")
	t.Setenv("PAYSTACK_PRO_PLAN_CODE", "PLN_pro")
	t.Setenv("PAYSTACK_PRO_AMOUNT_KOBO", "2500000")
	t.Setenv("PAYSTACK_CURRENCY", "GHS")
	t.Setenv("PAYSTACK_BASE_URL", "https://api.paystack.co")
	c := FromEnv()
	if c.PaystackSecretKey != "sk_test_9" || c.PaystackPlanCode != "PLN_pro" {
		t.Fatalf("credentials %+v", c)
	}
	if c.PaystackProAmountKobo != 2500000 || c.PaystackCurrency != "GHS" {
		t.Fatalf("amount/currency %+v", c)
	}

	t.Setenv("PAYSTACK_SECRET_KEY", "")
	t.Setenv("PAYSTACK_PRO_PLAN_CODE", "")
	t.Setenv("PAYSTACK_PRO_AMOUNT_KOBO", "")
	t.Setenv("PAYSTACK_CURRENCY", "")
	t.Setenv("PAYSTACK_BASE_URL", "")
	plain := FromEnv()
	if plain.PaystackProAmountKobo != 1500000 {
		t.Fatalf("default amount %d", plain.PaystackProAmountKobo)
	}
	if plain.PaystackCurrency != "NGN" || plain.PaystackBaseURL != "https://api.paystack.co" {
		t.Fatalf("defaults %+v", plain)
	}
	if plain.PaystackConfigured() {
		t.Fatal("without a secret key billing is not configured")
	}
}
