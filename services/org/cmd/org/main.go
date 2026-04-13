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

	"github.com/mobpilot/mobpilot/internal/platform/authz"
	"github.com/mobpilot/mobpilot/internal/platform/httpmw"
	platformnats "github.com/mobpilot/mobpilot/internal/platform/nats"
	platformotel "github.com/mobpilot/mobpilot/internal/platform/otel"
	platformpg "github.com/mobpilot/mobpilot/internal/platform/postgres"
	"github.com/mobpilot/mobpilot/services/org/application"
	orghttp "github.com/mobpilot/mobpilot/services/org/infrastructure/http"
	orgnats "github.com/mobpilot/mobpilot/services/org/infrastructure/nats"
	orgpg "github.com/mobpilot/mobpilot/services/org/infrastructure/postgres"
	sqlcorg "github.com/mobpilot/mobpilot/services/org/infrastructure/postgres/sqlc"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	// ── OpenTelemetry ──────────────────────────────────────────────────────────
	otelShutdown, err := platformotel.InitProvider(ctx, "org", version())
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

	// Ensure MOBPILOT_EVENTS stream exists (idempotent).
	if err := platformnats.EnsureStream(ctx, js); err != nil {
		logger.Error("nats stream setup failed", "err", err)
		os.Exit(1)
	}

	// ── Authorization (Ory Keto) ──────────────────────────────────────────────
	authzChecker := authz.NewKetoChecker(
		requireEnv("KETO_READ_URL"),
		requireEnv("KETO_WRITE_URL"),
	)

	// ── Repositories (infrastructure adapters) ────────────────────────────────
	orgRepo        := orgpg.NewOrgRepo(pool)
	memberRepo     := orgpg.NewMemberRepo(pool)
	invitationRepo := orgpg.NewInvitationRepo(pool)
	groupRepo      := orgpg.NewGroupRepo(pool)
	roleRepo       := orgpg.NewRoleRepo(pool)
	publisher      := orgnats.NewEventPublisher(js)
	queries        := sqlcorg.New(pool)

	// ── Application services (use cases) ─────────────────────────────────────
	orgSvc    := application.NewOrgService(orgRepo, memberRepo, publisher)
	memberSvc := application.NewMemberService(memberRepo, invitationRepo, publisher)
	roleSvc   := application.NewRoleService(roleRepo)

	// Centrifugo token issuer (nil-safe: handler returns 503 when not configured).
	centrifugoSecret := os.Getenv("CENTRIFUGO_TOKEN_SECRET")
	var tokenIssuer *orghttp.CentrifugoTokenIssuer
	var groupTokenIssuer application.TokenIssuer // interface, nil when not configured
	if centrifugoSecret != "" {
		tokenIssuer = orghttp.NewCentrifugoTokenIssuer(centrifugoSecret)
		groupTokenIssuer = tokenIssuer
	}

	centrifugoProxySecret := os.Getenv("CENTRIFUGO_PROXY_SECRET")
	proxyHandler := orghttp.NewCentrifugoProxyHandler(authzChecker, centrifugoProxySecret)

	groupSvc := application.NewGroupService(groupRepo, publisher, authzChecker, groupTokenIssuer)

	// ── Background workers ────────────────────────────────────────────────────
	outboxWorker := orgpg.NewOutboxWorker(queries, js, logger)
	go outboxWorker.Run(ctx)

	expirySweeper := application.NewGroupExpirySweeper(groupRepo, publisher, authzChecker, logger)
	go expirySweeper.Run(ctx)

	// ── HTTP router ───────────────────────────────────────────────────────────
	hydraAdminURL := requireEnv("HYDRA_ADMIN_URL")

	h := orghttp.NewHandler(orgSvc, memberSvc, groupSvc, roleSvc, tokenIssuer, proxyHandler)

	// Public router — all routes require Hydra Bearer token.
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

	// Internal router — server-to-server routes, NOT behind Hydra auth.
	// The Centrifugo proxy endpoint verifies a shared secret header instead.
	internal := chi.NewRouter()
	internal.Use(middleware.RequestID)
	internal.Use(httpmw.Logger(logger))
	internal.Use(middleware.Recoverer)
	h.MountInternal(internal)

	addr := envOr("LISTEN_ADDR", ":8081")
	internalAddr := envOr("INTERNAL_LISTEN_ADDR", ":8082")

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
		logger.Info("org service listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "err", err)
			stop()
		}
	}()
	go func() {
		logger.Info("org internal service listening", "addr", internalAddr)
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
