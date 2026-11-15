package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds every knob the engine reads from the environment.
type Config struct {
	Port           string
	DatabaseURL    string
	DataDir        string
	FrontendOrigin string
	APIToken       string

	// Blob storage: when S3 is set the raw bodies go to the bucket with a
	// local .data cache in front; otherwise bodies live on local disk only.
	S3Endpoint  string
	S3Bucket    string
	S3AccessKey string
	S3SecretKey string
	S3Region    string

	// Pocket AI Agent: any OpenAI compatible chat completions endpoint.
	LLMBaseURL string
	LLMAPIKey  string
	LLMModel   string

	GitHubToken string

	// Rimbra messaging.
	RimbraAPIKey        string
	RimbraWhatsAppURL   string
	RimbraEmailURL      string
	RimbraWebhookSecret string
	NotifyEmail         string

	ForwarderWorkers    int
	ForwarderPoll       time.Duration
	MaxDeliveryAttempts int
	RetryBase           time.Duration
	MaxBodyBytes        int64
	SpikeThreshold      int
	MuteDuration        time.Duration

	// GitHub OAuth sign in. The base URLs exist so tests can point at fake
	// GitHub servers.
	GitHubClientID     string
	GitHubClientSecret string
	GitHubOAuthBaseURL string
	GitHubAPIBaseURL   string
	AdminGitHubLogin   string
}

// FromEnv reads config from environment variables with sane defaults.
func FromEnv() Config {
	return Config{
		Port:                getenv("PORT", "8080"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		DataDir:             getenv("DATA_DIR", "/tmp/pockdev-data"),
		FrontendOrigin:      os.Getenv("FRONTEND_ORIGIN"),
		APIToken:            os.Getenv("API_TOKEN"),
		S3Endpoint:          os.Getenv("S3_ENDPOINT"),
		S3Bucket:            os.Getenv("S3_BUCKET"),
		S3AccessKey:         os.Getenv("S3_ACCESS_KEY"),
		S3SecretKey:         os.Getenv("S3_SECRET_KEY"),
		S3Region:            getenv("S3_REGION", "us-east-1"),
		LLMBaseURL:          os.Getenv("LLM_BASE_URL"),
		LLMAPIKey:           os.Getenv("LLM_API_KEY"),
		LLMModel:            getenv("LLM_MODEL", "default"),
		GitHubToken:         os.Getenv("GITHUB_TOKEN"),
		RimbraAPIKey:        os.Getenv("RIMBRA_API_KEY"),
		RimbraWhatsAppURL:   getenv("RIMBRA_WHATSAPP_URL", "https://api.rimbra.app/ntf/v1/messages"),
		RimbraEmailURL:      os.Getenv("RIMBRA_EMAIL_URL"),
		RimbraWebhookSecret: os.Getenv("RIMBRA_WEBHOOK_SECRET"),
		NotifyEmail:         os.Getenv("NOTIFY_EMAIL"),
		ForwarderWorkers:    getenvInt("FORWARDER_WORKERS", 4),
		ForwarderPoll:       time.Duration(getenvInt("FORWARDER_POLL_MS", 250)) * time.Millisecond,
		MaxDeliveryAttempts: getenvInt("MAX_DELIVERY_ATTEMPTS", 3),
		RetryBase:           time.Duration(getenvInt("RETRY_BASE_SECONDS", 5)) * time.Second,
		MaxBodyBytes:        int64(getenvInt("MAX_BODY_BYTES", 2*1024*1024)),
		SpikeThreshold:      getenvInt("SPIKE_THRESHOLD", 1),
		MuteDuration:        time.Duration(getenvInt("MUTE_MINUTES", 60)) * time.Minute,
		GitHubClientID:      os.Getenv("GITHUB_CLIENT_ID"),
		GitHubClientSecret:  os.Getenv("GITHUB_CLIENT_SECRET"),
		GitHubOAuthBaseURL:  getenv("GITHUB_OAUTH_BASE_URL", "https://github.com"),
		GitHubAPIBaseURL:    getenv("GITHUB_API_BASE_URL", "https://api.github.com"),
		AdminGitHubLogin:    getenv("ADMIN_GITHUB_LOGIN", "Mascottz"),
	}
}

// Validate checks the values a running server cannot do without.
func (c Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.APIToken == "" && !c.GitHubAuthConfigured() {
		return fmt.Errorf("API_TOKEN or GitHub OAuth credentials are required")
	}
	return nil
}

// GitHubAuthConfigured reports whether GitHub OAuth sign in can run.
func (c Config) GitHubAuthConfigured() bool {
	return c.GitHubClientID != "" && c.GitHubClientSecret != ""
}

// S3Enabled reports whether S3 backed blob storage is configured.
func (c Config) S3Enabled() bool {
	return c.S3Endpoint != "" && c.S3Bucket != "" && c.S3AccessKey != "" && c.S3SecretKey != ""
}

// AgentEnabled reports whether the Pocket AI Agent can call an LLM.
func (c Config) AgentEnabled() bool {
	return c.LLMBaseURL != ""
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return n
}
