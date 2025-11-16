package app

import (
	"context"
	"errors"
	"fmt"
	"gw-currency-wallet/internal/api/middlew"
	"gw-currency-wallet/internal/repository/postgres"
	"gw-currency-wallet/pkg/logger"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gw-currency-wallet/internal/api/handlers"
	"gw-currency-wallet/internal/config"
	"gw-currency-wallet/internal/db"
	"gw-currency-wallet/internal/server"
	"gw-currency-wallet/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	log         *slog.Logger
	server      *server.Server
	pool        *pgxpool.Pool
	logFile     *os.File
	cfg         *config.Config
	authService service.Auth // ← Используем интерфейс
}

func NewApp() (*App, error) {
	loggerWithFile := logger.NewLogger()
	log := loggerWithFile.Logger

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

	// Middlewares
	srv.Router.Use(middleware.RequestID)
	srv.Router.Use(middlew.WithLogger(log))
	srv.Router.Use(middleware.RealIP)
	srv.Router.Use(middleware.Recoverer)

	return &App{
		log:     log,
		server:  srv,
		pool:    pool,
		logFile: loggerWithFile.LogFile,
		cfg:     cfg,
	}, nil
}

// BuildAuthLayer собирает слой аутентификации и регистрации
func (a *App) BuildAuthLayer() {
	txManager := service.NewPgxTxManager(a.pool)
	userRepo := postgres.NewUserRepository(a.pool)
	walletRepo := postgres.NewWalletRepository(a.pool)

	a.authService = service.NewAuthService(
		userRepo,
		walletRepo,
		txManager,
		a.cfg.JWT.Secret,
		a.cfg.JWT.Expiration,
		a.log,
	)

	authHandler := handlers.NewAuthHandler(a.authService)

	// Публичные маршруты (без JWT)
	a.server.Router.Post("/api/v1/register", authHandler.Register)
	a.server.Router.Post("/api/v1/login", authHandler.Login)

	a.log.Info("слой 'auth' собран и маршруты зарегистрированы")
}

// BuildWalletLayer собирает слой работы с кошельками
func (a *App) BuildWalletLayer() {
	if a.authService == nil {
		a.log.Error("authService not initialized, call BuildAuthLayer first")
		panic("authService not initialized")
	}

	txManager := service.NewPgxTxManager(a.pool)
	walletRepo := postgres.NewWalletRepository(a.pool)
	walletService := service.NewWalletService(walletRepo, txManager)
	walletHandler := handlers.NewWalletHandler(walletService)

	// Защищенные маршруты (требуют JWT)
	a.server.Router.Group(func(r chi.Router) {
		// Применяем middleware для проверки JWT
		r.Use(middlew.RequireAuth(a.authService))

		// Wallet endpoints
		r.Get("/api/v1/wallets/{walletID}", walletHandler.GetWalletByID)
		r.Post("/api/v1/wallet", walletHandler.UpdateBalance)
		r.Get("/api/v1/balance", walletHandler.GetBalance)
		r.Post("/api/v1/wallet/deposit", walletHandler.Deposit)
		r.Post("/api/v1/wallet/withdraw", walletHandler.Withdraw)
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

	a.log.Info("закрытие файла логов")
	if a.logFile != nil {
		if err := a.logFile.Close(); err != nil {
			a.log.Error("ошибка при закрытии файла логов", slog.String("error", err.Error()))
		}
	}

	a.log.Info("приложение остановлено")
	return nil
}
