package config

import (
	"fmt"
	"log"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Kafka   KafkaConfig
	MongoDB MongoDBConfig
	App     AppConfig
}

type KafkaConfig struct {
	Brokers []string      `envconfig:"KAFKA_BROKERS" default:"localhost:9092"`
	Topic   string        `envconfig:"KAFKA_TOPIC" default:"large-transfers"`
	GroupID string        `envconfig:"KAFKA_GROUP_ID" default:"notification-service"`
	Workers int           `envconfig:"KAFKA_WORKERS" default:"5"` // Количество воркеров
	Timeout time.Duration `envconfig:"KAFKA_TIMEOUT" default:"10s"`
}

type MongoDBConfig struct {
	URI        string        `envconfig:"MONGO_URI" required:"true"`
	Database   string        `envconfig:"MONGO_DATABASE" default:"notifications"`
	Collection string        `envconfig:"MONGO_COLLECTION" default:"large_transfers"`
	Timeout    time.Duration `envconfig:"MONGO_TIMEOUT" default:"10s"`
}

type AppConfig struct {
	LogLevel string `envconfig:"LOG_LEVEL" default:"info"` // debug, info, warn, error
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
