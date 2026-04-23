package app

import (
	"context"
	"log/slog"

	"xugeaneeu/pollify/internal/platform/auth"
	"xugeaneeu/pollify/internal/platform/config"
	"xugeaneeu/pollify/internal/platform/httpserver"
	platformpg "xugeaneeu/pollify/internal/platform/postgres"
)

func Run(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	if cfg.DatabaseURL != "" {
		pool, err := platformpg.Open(ctx, cfg.DatabaseURL)
		if err != nil {
			return err
		}
		defer pool.Close()
		if err := platformpg.ApplyMigrations(ctx, pool, cfg.MigrationsPath); err != nil {
			return err
		}
		logger.Info("postgres connection established")
		logger.Info("postgres migrations applied", "path", cfg.MigrationsPath)
	}

	server := httpserver.New(cfg, logger, auth.NewHMACVerifier([]byte(cfg.JWTSecret), nil))
	logger.Info("starting api server", "address", cfg.HTTPAddress, "openapi_path", cfg.OpenAPIPath)
	return server.Run(ctx)
}
