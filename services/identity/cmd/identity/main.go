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
	platformotel "github.com/mobpilot/mobpilot/internal/platform/otel"
	platformpg "github.com/mobpilot/mobpilot/internal/platform/postgres"
	"github.com/mobpilot/mobpilot/services/identity/application"
	identityhttp "github.com/mobpilot/mobpilot/services/identity/infrastructure/http"
	identitynats "github.com/mobpilot/mobpilot/services/identity/infrastructure/nats"
	identitypg "github.com/mobpilot/mobpilot/services/identity/infrastructure/postgres"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	// ── OpenTelemetry ──────────────────────────────────────────────────────────
	otelShutdown, err := platformotel.InitProvider(ctx, "identity", version())
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
	defer nc.Drain()

	js, err := natsjs.New(nc)
	if err != nil {
		logger.Error("nats jetstream failed", "err", err)
		os.Exit(1)
	}

	// ── Repositories (infrastructure adapters) ────────────────────────────────
	userRepo     := identitypg.NewUserRepo(pool)
	deviceRepo   := identitypg.NewDeviceRepo(pool)
	artifactRepo := identitypg.NewArtifactRepo(pool)
	publisher    := identitynats.NewEventPublisher(js)

	// ── Application services (use cases) ─────────────────────────────────────
	userSvc     := application.NewUserService(userRepo, publisher)
	deviceSvc   := application.NewDeviceService(deviceRepo)
	artifactSvc := application.NewArtifactService(artifactRepo)

	// ── HTTP router ───────────────────────────────────────────────────────────
	hydraAdminURL := requireEnv("HYDRA_ADMIN_URL")
	internalAPIKey := os.Getenv("INTERNAL_API_KEY")

	h := identityhttp.NewHandler(userSvc, deviceSvc, artifactSvc)

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

	h.Mount(r)

	// Internal routes (no Hydra auth — verified by X-Internal-Api-Key).
	internal := chi.NewRouter()
	internal.Use(middleware.RequestID)
	internal.Use(httpmw.Logger(logger))
	internal.Use(middleware.Recoverer)
	h.MountInternal(internal, internalAPIKey)

	addr := envOr("LISTEN_ADDR", ":8080")
	internalAddr := envOr("INTERNAL_LISTEN_ADDR", ":8084")

	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	internalSrv := &http.Server{
		Addr:         internalAddr,
		Handler:      internal,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("identity service listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "err", err)
			stop()
		}
	}()
	go func() {
		logger.Info("identity internal service listening", "addr", internalAddr)
		if err := internalSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("internal server error", "err", err)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	_ = internalSrv.Shutdown(shutdownCtx)
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
