package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"gw-currency-wallet/internal/models"
	"log/slog"

	"github.com/IBM/sarama"
)

// Producer интерфейс для отправки событий в Kafka
type Producer interface {
	SendLargeTransferEvent(ctx context.Context, event models.LargeTransferEvent) error
	Close() error
}

type KafkaProducer struct {
	producer sarama.SyncProducer
	topic    string
	log      *slog.Logger
}

// NewKafkaProducer создает новый Kafka producer
func NewKafkaProducer(brokers []string, topic string, log *slog.Logger) (Producer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5
	config.Producer.Compression = sarama.CompressionSnappy

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka producer: %w", err)
	}

	log.Info("kafka producer создан", slog.String("topic", topic), slog.Any("brokers", brokers))

	return &KafkaProducer{
		producer: producer,
		topic:    topic,
		log:      log,
	}, nil
}

// SendLargeTransferEvent отправляет событие о крупном переводе в Kafka
func (p *KafkaProducer) SendLargeTransferEvent(ctx context.Context, event models.LargeTransferEvent) error {
	const op = "kafka.SendLargeTransferEvent"

	// Сериализуем событие в JSON
	eventData, err := json.Marshal(event)
	if err != nil {
		p.log.Error("ошибка сериализации события",
			slog.String("op", op),
			slog.String("error", err.Error()))
		return fmt.Errorf("%s: %w", op, err)
	}

	// Создаем сообщение для Kafka
	msg := &sarama.ProducerMessage{
		Topic: p.topic,
		Key:   sarama.StringEncoder(event.TransactionID), // Ключ для партиционирования
		Value: sarama.ByteEncoder(eventData),
	}

	// Отправляем сообщение
	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		p.log.Error("ошибка отправки события в kafka",
			slog.String("op", op),
			slog.String("transaction_id", event.TransactionID),
			slog.String("error", err.Error()))
		return fmt.Errorf("%s: %w", op, err)
	}

	p.log.Info("событие отправлено в kafka",
		slog.String("op", op),
		slog.String("transaction_id", event.TransactionID),
		slog.String("user_id", event.UserID.String()),
		slog.Float64("amount", event.Amount),
		slog.Int("partition", int(partition)),
		slog.Int64("offset", offset))

	return nil
}

// Close закрывает producer
func (p *KafkaProducer) Close() error {
	p.log.Info("закрытие kafka producer")
	return p.producer.Close()
}

// NoOpProducer - заглушка для случаев, когда Kafka недоступен или отключен
type NoOpProducer struct {
	log *slog.Logger
}

// NewNoOpProducer создает no-op producer
func NewNoOpProducer(log *slog.Logger) Producer {
	return &NoOpProducer{log: log}
}

func (p *NoOpProducer) SendLargeTransferEvent(ctx context.Context, event models.LargeTransferEvent) error {
	p.log.Debug("kafka отключен, событие не отправлено",
		slog.String("transaction_id", event.TransactionID))
	return nil
}

func (p *NoOpProducer) Close() error {
	return nil
}
