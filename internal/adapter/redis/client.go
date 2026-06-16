package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/statoon54/mailhive/internal/config"
)

// NewClient crée un client Redis.
func NewClient(ctx context.Context, cfg config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("impossible de contacter Redis : %w", err)
	}

	return client, nil
}
