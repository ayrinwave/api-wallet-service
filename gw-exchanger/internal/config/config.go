package config

import (
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	GRPCPort string `envconfig:"GRPC_PORT" default:"50051"`
	DB       DBConfig
}

type DBConfig struct {
	Host     string `envconfig:"POSTGRES_HOST" required:"true"`
	Port     string `envconfig:"POSTGRES_PORT" required:"true"`
	User     string `envconfig:"POSTGRES_USER" required:"true"`
	Password string `envconfig:"POSTGRES_PASSWORD" required:"true"`
	DBName   string `envconfig:"POSTGRES_DB" required:"true"`
	SSLMode  string `envconfig:"POSTGRES_SSLMODE" default:"disable"`
}

func NewConfig() (*Config, error) {
	envFile := "config.env"

	if err := godotenv.Load(envFile); err != nil {
		log.Printf("warning: не удалось загрузить файл %s, используются только системные переменные окружения: %v", envFile, err)
	}

	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("ошибка парсинга конфигурации: %w", err)
	}

	return &cfg, nil
}
func (d *DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
	)
}

func (d *DBConfig) MigrationURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.DBName, d.SSLMode,
	)
}
