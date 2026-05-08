package app

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"xugeaneeu/pollify/internal/platform/auth"
	"xugeaneeu/pollify/internal/platform/clock"
	"xugeaneeu/pollify/internal/platform/config"
	"xugeaneeu/pollify/internal/platform/httpserver"
	platformpg "xugeaneeu/pollify/internal/platform/postgres"

	pollshttp "xugeaneeu/pollify/internal/polls/adapters/http"
	pollsrepo "xugeaneeu/pollify/internal/polls/adapters/postgres"
	polls "xugeaneeu/pollify/internal/polls/core"

	usershttp "xugeaneeu/pollify/internal/users/adapters/http"
	usersrepo "xugeaneeu/pollify/internal/users/adapters/postgres"
	users "xugeaneeu/pollify/internal/users/core"

	votinghttp "xugeaneeu/pollify/internal/voting/adapters/http"
	votingrepo "xugeaneeu/pollify/internal/voting/adapters/postgres"
	voting "xugeaneeu/pollify/internal/voting/core"

	moderationhttp "xugeaneeu/pollify/internal/moderation/adapters/http"
	moderationrepo "xugeaneeu/pollify/internal/moderation/adapters/postgres"
	moderation "xugeaneeu/pollify/internal/moderation/core"
)

var ErrDatabaseURLRequired = errors.New("app: DATABASE_URL is required")

func Run(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	if cfg.DatabaseURL == "" {
		return ErrDatabaseURLRequired
	}

	pool, err := platformpg.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := platformpg.ApplyMigrations(ctx, pool, cfg.MigrationsPath); err != nil {
		return err
	}
	logger.Info("postgres migrations applied", "path", cfg.MigrationsPath)

	handler, err := buildHandler(cfg, logger, pool)
	if err != nil {
		return err
	}

	server := httpserver.New(cfg, logger, handler)
	logger.Info("starting api server", "address", cfg.HTTPAddress)
	return server.Run(ctx)
}

func BuildHandler(cfg config.Config, logger *slog.Logger, pool *pgxpool.Pool) (http.Handler, error) {
	return buildHandler(cfg, logger, pool)
}

func buildHandler(cfg config.Config, logger *slog.Logger, pool *pgxpool.Pool) (http.Handler, error) {
	systemClock := clock.SystemClock{}
	hasher := auth.NewBcryptHasher(0)
	issuer := auth.NewHMACIssuer([]byte(cfg.JWTSecret), time.Hour, time.Now)
	verifier := auth.NewHMACVerifier([]byte(cfg.JWTSecret), time.Now)

	userRepo := usersrepo.NewRepository(pool)
	userService, err := users.NewService(userRepo, hasher, issuer, systemClock)
	if err != nil {
		return nil, err
	}
	userHandler := usershttp.NewHandler(userService)

	pollRepo := pollsrepo.NewRepository(pool)
	pollService, err := polls.NewService(pollRepo, pollRepo, systemClock)
	if err != nil {
		return nil, err
	}
	pollHandler := pollshttp.NewHandler(pollService, pollStatsAdapter{repo: pollRepo}, pollRepo, systemClock)

	voteRepo := votingrepo.NewRepository(pool)
	voteService, err := voting.NewService(pollRepo, voteRepo, systemClock)
	if err != nil {
		return nil, err
	}
	voteHandler := votinghttp.NewHandler(voteService)

	moderationRepo := moderationrepo.NewRepository(pool)
	moderationService, err := moderation.NewService(moderationRepo, pollRepo, pollRepo, systemClock, moderation.DefaultQuorum)
	if err != nil {
		return nil, err
	}
	moderationHandler := moderationhttp.NewHandler(moderationService, moderationRepo)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(slogRequestLogger(logger))

	r.Get("/healthz", healthzHandler)

	r.Route("/api/v1", func(r chi.Router) {
		r.Mount("/auth", userHandler.PublicRoutes())

		r.Group(func(protected chi.Router) {
			protected.Use(auth.Middleware(verifier))
			protected.Mount("/users", userHandler.ProtectedRoutes())
			protected.Route("/polls", func(p chi.Router) {
				pollHandler.RegisterRoutes(p)
				voteHandler.RegisterRoutes(p)
				p.Post("/{pollId}/reports", moderationHandler.CreateReport)
				p.With(auth.RequireRole(users.RoleAdmin)).Get("/{pollId}/reports", moderationHandler.ListReportsByPoll)
			})
			protected.Route("/reports", func(rep chi.Router) {
				rep.Use(auth.RequireRole(users.RoleAdmin))
				rep.Get("/", moderationHandler.ListReports)
				rep.Get("/{reportId}", moderationHandler.GetReport)
				rep.Post("/{reportId}/reviews", moderationHandler.SubmitReview)
			})
		})
	})

	return r, nil
}

type pollStatsAdapter struct {
	repo *pollsrepo.Repository
}

func (a pollStatsAdapter) Count(ctx context.Context, pollID string) (int, error) {
	return a.repo.ParticipationCount(ctx, pollID)
}

func (a pollStatsAdapter) HasVoted(ctx context.Context, pollID, userID string) (bool, error) {
	return a.repo.HasUserVoted(ctx, pollID, userID)
}

func healthzHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func slogRequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()
			next.ServeHTTP(ww, r)
			logger.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", middleware.GetReqID(r.Context()),
			)
		})
	}
}
