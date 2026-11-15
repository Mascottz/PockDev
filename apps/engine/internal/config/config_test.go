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
