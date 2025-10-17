package db

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func RunMigrations(dsn string, migrationsPath string) error {
	if dsn == "" {
		return errors.New("DSN для миграций не может быть пустым")
	}
	if migrationsPath == "" {
		return errors.New("путь к файлам миграций не может быть пустым")
	}

	m, err := migrate.New("file://"+migrationsPath, dsn)
	if err != nil {
		return fmt.Errorf("не удалось создать экземпляр мигратора: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("ошибка при выполнении миграций: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil {
		return fmt.Errorf("ошибка при проверке версии миграций: %w", err)
	}
	if dirty {
		return fmt.Errorf("обнаружена 'грязная' миграция версии %d. Исправьте вручную", version)
	}

	return nil
}
