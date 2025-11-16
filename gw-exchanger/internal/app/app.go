package app

import (
	"context"
	"fmt"
	"gw-exchanger/internal/config"
	"gw-exchanger/internal/db"
	"gw-exchanger/internal/grpc_server"
	"gw-exchanger/internal/storage/postgres"
	pb "gw-exchanger/proto-exchange"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
)

type App struct {
	log        *slog.Logger
	cfg        *config.Config
	pool       *pgxpool.Pool
	grpcServer *grpc.Server
	listener   net.Listener
}

func NewApp() (*App, error) {
	// Инициализация логгера
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	log.Info("инициализация gw-exchanger приложения")

	// Загрузка конфигурации
	cfg, err := config.NewConfig()
	if err != nil {
		return nil, fmt.Errorf("ошибка загрузки конфигурации: %w", err)
	}
	log.Info("конфигурация загружена", slog.String("grpc_port", cfg.GRPCPort))

	// Выполнение миграций
	log.Info("выполнение миграций базы данных")
	if err := db.RunMigrations(cfg.DB.MigrationURL(), "migrations"); err != nil {
		return nil, fmt.Errorf("ошибка выполнения миграций: %w", err)
	}
	log.Info("миграции успешно применены")

	// Подключение к БД
	pool, err := db.NewPool(context.Background(), cfg.DB.DSN())
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к базе данных: %w", err)
	}
	log.Info("подключение к базе данных установлено")

	// Инициализация хранилища
	storage := postgres.NewPostgresStorage(pool)

	// Инициализация gRPC сервера
	exchangeServer := grpc_server.NewExchangeServer(storage, log)

	// Создание gRPC сервера
	grpcServer := grpc.NewServer()
	pb.RegisterExchangeServiceServer(grpcServer, exchangeServer)

	// Слушаем порт
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("ошибка создания listener: %w", err)
	}

	log.Info("gRPC сервер инициализирован", slog.String("port", cfg.GRPCPort))

	return &App{
		log:        log,
		cfg:        cfg,
		pool:       pool,
		grpcServer: grpcServer,
		listener:   listener,
	}, nil
}

func (a *App) Run() error {
	a.log.Info("gRPC сервер запускается", slog.String("port", a.cfg.GRPCPort))

	// Запуск сервера в горутине
	serverErr := make(chan error, 1)
	go func() {
		if err := a.grpcServer.Serve(a.listener); err != nil {
			serverErr <- fmt.Errorf("ошибка запуска gRPC сервера: %w", err)
		}
	}()

	// Graceful shutdown
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return err
	case sig := <-shutdownChan:
		a.log.Info("получен сигнал завершения", slog.String("signal", sig.String()))
	}

	a.log.Info("приложение останавливается")

	// Graceful stop gRPC сервера
	a.log.Info("остановка gRPC сервера")
	a.grpcServer.GracefulStop()

	// Закрытие соединения с БД
	a.log.Info("закрытие соединения с базой данных")
	a.pool.Close()

	a.log.Info("приложение остановлено")
	return nil
}
