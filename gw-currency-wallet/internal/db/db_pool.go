package db

import (
	"context"
	"fmt"
	"time"

	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PoolConfig struct {
	MaxConns          int
	MinConns          int
	HealthCheckPeriod time.Duration
	PoolTimeout       time.Duration
	RetryAttempts     int
	RetryDelay        time.Duration
}

func NewPool(ctx context.Context, dsn string, cfg PoolConfig) (*pgxpool.Pool, error) {
	if cfg.MaxConns <= 0 {
		cfg.MaxConns = 50
	}
	if cfg.MinConns < 0 {
		cfg.MinConns = 5
	}
	if cfg.HealthCheckPeriod <= 0 {
		cfg.HealthCheckPeriod = 30 * time.Second
	}
	if cfg.PoolTimeout <= 0 {
		cfg.PoolTimeout = 5 * time.Second
	}
	if cfg.RetryAttempts <= 0 {
		cfg.RetryAttempts = 5
	}
	if cfg.RetryDelay <= 0 {
		cfg.RetryDelay = 1 * time.Second
	}

	conf, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("не удалось распарсить DSN: %w", err)
	}

	conf.MaxConns = int32(cfg.MaxConns)
	conf.MinConns = int32(cfg.MinConns)
	conf.HealthCheckPeriod = cfg.HealthCheckPeriod
	conf.MaxConnLifetime = 30 * time.Minute
	conf.MaxConnIdleTime = 5 * time.Minute
	conf.ConnConfig.RuntimeParams["application_name"] = "wallet-app"
	conf.ConnConfig.ConnectTimeout = cfg.PoolTimeout

	var pool *pgxpool.Pool
	for i := 0; i < cfg.RetryAttempts; i++ {
		pool, err = pgxpool.NewWithConfig(ctx, conf)
		if err != nil {
			log.Printf("Попытка %d: не удалось создать пул соединений: %v", i+1, err)
			time.Sleep(cfg.RetryDelay * time.Duration(1<<i))
			continue
		}

		if err = pool.Ping(ctx); err != nil {
			log.Printf("Попытка %d: ping БД не удался: %v", i+1, err)
			pool.Close()
			time.Sleep(cfg.RetryDelay * time.Duration(1<<i))
			continue
		}

		log.Println("Подключение к базе данных успешно")
		return pool, nil
	}

	return nil, fmt.Errorf("не удалось создать пул соединений после %d попыток: %w", cfg.RetryAttempts, err)
}
