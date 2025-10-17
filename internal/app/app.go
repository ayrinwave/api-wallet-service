package app

import (
	"api_wallet/internal/api/middlew"
	"api_wallet/internal/repository/postgres"
	"api_wallet/pkg/logger"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"api_wallet/internal/api/handlers"
	"api_wallet/internal/config"
	"api_wallet/internal/db"
	"api_wallet/internal/server"
	"api_wallet/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	log    *slog.Logger
	server *server.Server
	pool   *pgxpool.Pool
}

func NewApp() (*App, error) {
	log := logger.NewLogger()
	log.Info("инициализация приложения")

	cfg, err := config.NewConfig()
	if err != nil {
		return nil, fmt.Errorf("ошибка инициализации конфига: %w", err)
	}
	log.Info("конфигурация загружена", slog.String("port", cfg.HTTPPort))

	log.Info("выполнение миграций базы данных")
	if err := db.RunMigrations(cfg.DB.MigrationURL(), "migrations"); err != nil {
		return nil, fmt.Errorf("ошибка выполнения миграций: %w", err)
	}
	log.Info("миграции успешно применены")

	pool, err := db.NewPool(context.Background(), cfg.DB.DSN())
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к базе данных: %w", err)
	}
	log.Info("подключение к базе данных установлено")

	srv := server.NewServer(cfg.HTTPPort)
	log.Info("сервер инициализирован", slog.String("port", cfg.HTTPPort))

	srv.Router.Use(middleware.RequestID)
	srv.Router.Use(middlew.WithLogger(log))
	srv.Router.Use(middleware.RealIP)
	srv.Router.Use(middleware.Recoverer)

	return &App{
		log:    log,
		server: srv,
		pool:   pool,
	}, nil
}

func (a *App) BuildWalletLayer() {
	walletRepo := postgres.NewWalletRepository(a.pool)
	walletService := service.NewWalletService(walletRepo, a.pool)
	walletHandler := handlers.NewWalletHandler(walletService)

	a.server.Router.Route("/api/v1", func(r chi.Router) {
		r.Get("/wallets/{walletID}", walletHandler.GetWalletByID)
		r.Post("/wallet", walletHandler.UpdateBalance)
	})

	a.log.Info("слой 'wallet' собран и маршруты зарегистрированы")
}

func (a *App) Run() error {
	a.log.Info("сервер запускается")

	serverErr := make(chan error, 1)
	go func() {
		if err := a.server.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- fmt.Errorf("ошибка запуска сервера: %w", err)
		}
	}()

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return err
	case sig := <-shutdownChan:
		a.log.Info("получен сигнал завершения", slog.String("signal", sig.String()))
	}

	a.log.Info("приложение останавливается")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		a.log.Error("ошибка при остановке http сервера", slog.String("error", err.Error()))
	}

	a.log.Info("закрытие соединения с базой данных")
	a.pool.Close()

	a.log.Info("приложение остановлено")
	return nil
}
