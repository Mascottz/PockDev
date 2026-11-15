// Command pockdev runs the PockDev engine: webhook shield, error watch, the
// Pocket AI Agent, and the dashboard REST API.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Mascottz/PockDev/apps/engine/internal/agent"
	"github.com/Mascottz/PockDev/apps/engine/internal/api"
	"github.com/Mascottz/PockDev/apps/engine/internal/auth"
	"github.com/Mascottz/PockDev/apps/engine/internal/billing"
	"github.com/Mascottz/PockDev/apps/engine/internal/blob"
	"github.com/Mascottz/PockDev/apps/engine/internal/config"
	"github.com/Mascottz/PockDev/apps/engine/internal/db"
	"github.com/Mascottz/PockDev/apps/engine/internal/errwatch"
	"github.com/Mascottz/PockDev/apps/engine/internal/httpx"
	"github.com/Mascottz/PockDev/apps/engine/internal/notify"
	"github.com/Mascottz/PockDev/apps/engine/internal/store/pgxstore"
	"github.com/Mascottz/PockDev/apps/engine/internal/webhook"
	"github.com/Mascottz/PockDev/apps/engine/internal/whatsapp"
	"github.com/Mascottz/PockDev/apps/engine/internal/worker"
)

func main() {
	cfg := config.FromEnv()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	st := pgxstore.New(pool)

	localBlobs, err := blob.NewLocal(cfg.DataDir)
	if err != nil {
		log.Fatalf("blob: %v", err)
	}
	var remote blob.Store
	if cfg.S3Enabled() {
		remote = blob.NewS3(cfg.S3Endpoint, cfg.S3Bucket, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3Region)
	}
	blobs := blob.NewSpool(localBlobs, remote)

	rimbra := &notify.Client{
		APIKey:      cfg.RimbraAPIKey,
		WhatsAppURL: cfg.RimbraWhatsAppURL,
		EmailURL:    cfg.RimbraEmailURL,
	}
	pocketAgent := &agent.Agent{
		Store: st,
		LLM: &agent.LLM{
			BaseURL: cfg.LLMBaseURL,
			APIKey:  cfg.LLMAPIKey,
			Model:   cfg.LLMModel,
		},
		GitHub:  &agent.GitHub{Token: cfg.GitHubToken},
		Notify:  rimbra,
		MuteFor: cfg.MuteDuration,
	}

	deadLetters := &webhook.DeadLetterHandler{
		Store:   st,
		Trigger: pocketAgent.Trigger,
	}

	forwarder := &worker.Forwarder{
		Store:        st,
		Blobs:        blobs,
		Workers:      cfg.ForwarderWorkers,
		Poll:         cfg.ForwarderPoll,
		MaxAttempts:  cfg.MaxDeliveryAttempts,
		RetryBase:    cfg.RetryBase,
		OnDeadLetter: deadLetters.Handle,
	}

	errorsIngest := &errwatch.Ingest{
		Store:          st,
		SpikeThreshold: cfg.SpikeThreshold,
		Window:         time.Hour,
		OnIncident:     pocketAgent.Trigger,
	}

	webhookSecret := cfg.RimbraWebhookSecret
	if webhookSecret == "" {
		webhookSecret = cfg.RimbraAPIKey
	}
	waHook := &whatsapp.Hook{
		Store:  st,
		Agent:  pocketAgent,
		Notify: rimbra,
		Secret: webhookSecret,
	}

	ingest := &webhook.Ingest{
		Store:   st,
		Blobs:   blobs,
		MaxBody: cfg.MaxBodyBytes,
	}

	dash := &api.API{
		Store:                st,
		Agent:                pocketAgent,
		APIToken:             cfg.APIToken,
		GitHubAuthConfigured: cfg.GitHubAuthConfigured(),
		Billing:              billing.NewClient(cfg.PaystackSecretKey, cfg.PaystackBaseURL, cfg.PaystackPlanCode),
		ProPriceKobo:         cfg.PaystackProAmountKobo,
		PaystackCurrency:     cfg.PaystackCurrency,
		FrontendOrigin:       cfg.FrontendOrigin,
		StartedAt:            time.Now(),
	}

	paystackHook := &billing.Webhook{
		Secret: cfg.PaystackSecretKey,
		Store:  st,
	}

	retention := &worker.RetentionSweeper{
		Store: st,
		Blobs: blobs,
	}

	githubAuth := &auth.Handler{
		ClientID:       cfg.GitHubClientID,
		ClientSecret:   cfg.GitHubClientSecret,
		OAuthBaseURL:   cfg.GitHubOAuthBaseURL,
		APIBaseURL:     cfg.GitHubAPIBaseURL,
		FrontendOrigin: cfg.FrontendOrigin,
		AdminLogin:     cfg.AdminGitHubLogin,
		Store:          st,
	}

	allowedOrigins := []string{"http://localhost:3000"}
	if cfg.FrontendOrigin != "" {
		allowedOrigins = append(allowedOrigins, cfg.FrontendOrigin)
	}

	mux := http.NewServeMux()
	mux.Handle("POST /in/{endpointKey}", ingest)
	mux.Handle("POST /v1/errors", errorsIngest)
	mux.Handle("POST /webhooks/whatsapp", waHook)
	mux.Handle("POST /webhooks/paystack", paystackHook)
	dash.Register(mux)
	githubAuth.Register(mux)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpx.CORS(allowedOrigins, mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go forwarder.Run(ctx)
	go pocketAgent.Run(ctx)
	go retention.Run(ctx)
	go func() {
		t := time.NewTicker(time.Minute)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if _, err := st.ExpireMutes(ctx, time.Now()); err != nil && ctx.Err() == nil {
					log.Printf("mute sweep: %v", err)
				}
			}
		}
	}()

	go func() {
		secLast4 := strings.TrimSpace(webhookSecret)
		if len(secLast4) > 4 {
			secLast4 = secLast4[len(secLast4)-4:]
		}
		log.Printf("pockdev engine listening on :%s (whatsapp_configured=%v webhook_secret_last4=%q)", cfg.Port, rimbra.WhatsAppConfigured(), secLast4)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	os.Exit(0)
}
