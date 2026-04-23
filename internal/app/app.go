package app

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5/pgxpool"

	"zedsellauto/internal/config"
	"zedsellauto/internal/database"
	"zedsellauto/internal/httpapi"
	"zedsellauto/internal/service"
)

type App struct {
	Router *gin.Engine

	db    *pgxpool.Pool
	redis *redis.Client
}

func New(cfg config.Config) (*App, error) {
	ctx := context.Background()

	db, err := database.NewPostgres(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}

	cache := database.NewRedis(cfg)
	if err := cache.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("connect redis: %w", err)
	}

	if err := database.Migrate(ctx, db); err != nil {
		return nil, fmt.Errorf("migrate schema: %w", err)
	}

	if err := database.Seed(ctx, db); err != nil {
		return nil, fmt.Errorf("seed data: %w", err)
	}

	services := service.NewServices(cfg, db, cache)
	router := httpapi.NewRouter(cfg, services)

	return &App{
		Router: router,
		db:     db,
		redis:  cache,
	}, nil
}

func (a *App) Close() {
	if a.redis != nil {
		_ = a.redis.Close()
	}
	if a.db != nil {
		a.db.Close()
	}
}
