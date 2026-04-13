package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/nats-io/nats.go"
	natsjs "github.com/nats-io/nats.go/jetstream"

	"github.com/mobpilot/mobpilot/internal/platform/httpmw"
	platformnats "github.com/mobpilot/mobpilot/internal/platform/nats"
	platformotel "github.com/mobpilot/mobpilot/internal/platform/otel"
	platformpg "github.com/mobpilot/mobpilot/internal/platform/postgres"
	"github.com/mobpilot/mobpilot/services/notification/application"
	domainports "github.com/mobpilot/mobpilot/services/notification/domain/ports"
	notificationhttp "github.com/mobpilot/mobpilot/services/notification/infrastructure/http"
	emailadapter "github.com/mobpilot/mobpilot/services/notification/infrastructure/email"
	notificationnats "github.com/mobpilot/mobpilot/services/notification/infrastructure/nats"
	notificationpg "github.com/mobpilot/mobpilot/services/notification/infrastructure/postgres"
	"github.com/mobpilot/mobpilot/services/notification/infrastructure/push"
	"github.com/mobpilot/mobpilot/services/notification/infrastructure/realtime"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	// ── OpenTelemetry ──────────────────────────────────────────────────────────
	otelShutdown, err := platformotel.InitProvider(ctx, "notification", version())
	if err != nil {
		logger.Error("otel init failed", "err", err)
		os.Exit(1)
	}
	defer func() { _ = otelShutdown(context.Background()) }()

	// ── PostgreSQL ─────────────────────────────────────────────────────────────
	pgDSN := requireEnv("POSTGRES_DSN")
	pool, err := platformpg.NewPool(ctx, platformpg.DefaultConfig(pgDSN))
	if err != nil {
		logger.Error("postgres connect failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	// ── NATS JetStream ─────────────────────────────────────────────────────────
	nc, err := nats.Connect(requireEnv("NATS_URL"))
	if err != nil {
		logger.Error("nats connect failed", "err", err)
		os.Exit(1)
	}
	defer func() { _ = nc.Drain() }()

	js, err := natsjs.New(nc)
	if err != nil {
		logger.Error("nats jetstream failed", "err", err)
		os.Exit(1)
	}

	// Ensure stream exists (idempotent; org service also calls this at startup).
	if err := platformnats.EnsureStream(ctx, js); err != nil {
		logger.Error("nats stream setup failed", "err", err)
		os.Exit(1)
	}

	// ── Repositories ──────────────────────────────────────────────────────────
	notificationRepo := notificationpg.NewNotificationRepo(pool)
	preferenceRepo   := notificationpg.NewPreferenceRepo(pool)

	// ── Push adapter ──────────────────────────────────────────────────────────
	// NoopPushPort in dev; swap for a real Dispatcher with FCM/APNs in production.
	var pushPort domainports.PushPort = &push.NoopPushPort{}

	// ── Email adapter ─────────────────────────────────────────────────────────
	var emailPort domainports.EmailPort = &emailadapter.NoopEmailPort{}
	if smtpHost := os.Getenv("SMTP_HOST"); smtpHost != "" {
		emailPort = emailadapter.NewSMTPAdapter(
			smtpHost,
			envOr("SMTP_PORT", "1025"),
			envOr("SMTP_FROM", "no-reply@mobpilot.dev"),
			os.Getenv("SMTP_USER"),
			os.Getenv("SMTP_PASSWORD"),
		)
	}

	// ── Centrifugo real-time adapter ──────────────────────────────────────────
	var realtimePort domainports.RealtimePort = &realtime.NoopRealtimePort{}
	if centrifugoURL := os.Getenv("CENTRIFUGO_API_URL"); centrifugoURL != "" {
		realtimePort = realtime.NewCentrifugoAdapter(centrifugoURL, requireEnv("CENTRIFUGO_API_KEY"))
	}

	// ── Service-to-service HTTP clients ───────────────────────────────────────
	internalAPIKey := envOr("INTERNAL_API_KEY", "")
	orgClient      := notificationhttp.NewOrgClient(requireEnv("ORG_SERVICE_URL"), internalAPIKey)
	identityClient := notificationhttp.NewIdentityClient(requireEnv("IDENTITY_SERVICE_URL"), internalAPIKey)

	// ── Application services ──────────────────────────────────────────────────
	notificationSvc := application.NewNotificationService(
		notificationRepo,
		preferenceRepo,
		pushPort,
		emailPort,
		realtimePort,
		orgClient,
		identityClient,
		logger,
	)
	preferenceSvc := application.NewPreferenceService(preferenceRepo)

	// ── NATS subscriber (background) ──────────────────────────────────────────
	subscriber := notificationnats.NewSubscriber(js, notificationSvc, logger)
	go subscriber.Run(ctx)

	// ── HTTP router ───────────────────────────────────────────────────────────
	hydraAdminURL := requireEnv("HYDRA_ADMIN_URL")

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(httpmw.Logger(logger))
	r.Use(middleware.Recoverer)
	r.Use(httpmw.BearerAuth(hydraAdminURL))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	h := notificationhttp.NewHandler(notificationSvc, preferenceSvc)
	h.Mount(r)

	addr := envOr("LISTEN_ADDR", ":8083")
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("notification service listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		slog.Error("required environment variable not set", "key", key)
		os.Exit(1)
	}
	return v
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func version() string {
	if v := os.Getenv("SERVICE_VERSION"); v != "" {
		return v
	}
	return "dev"
}
