package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"gw-currency-wallet/internal/models"
	"log/slog"
	"time"

	"github.com/IBM/sarama"
)

type Producer interface {
	SendLargeTransferEvent(ctx context.Context, event models.LargeTransferEvent) error
	Close() error
}

type KafkaProducer struct {
	producer sarama.SyncProducer
	topic    string
	log      *slog.Logger
}

func NewKafkaProducer(brokers []string, topic string, log *slog.Logger) (Producer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5
	config.Producer.Compression = sarama.CompressionSnappy
	config.Producer.Timeout = 5 * time.Second // реальный таймаут для SendMessage

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

func (p *KafkaProducer) SendLargeTransferEvent(ctx context.Context, event models.LargeTransferEvent) error {
	const op = "kafka.SendLargeTransferEvent"

	select {
	case <-ctx.Done():
		p.log.Warn("операция отменена до отправки",
			slog.String("op", op),
			slog.String("transaction_id", event.TransactionID),
			slog.String("reason", ctx.Err().Error()))
		return fmt.Errorf("%s: %w", op, ctx.Err())
	default:
	}

	eventData, err := json.Marshal(event)
	if err != nil {
		p.log.Error("ошибка сериализации события",
			slog.String("op", op),
			slog.String("error", err.Error()))
		return fmt.Errorf("%s: %w", op, err)
	}

	msg := &sarama.ProducerMessage{
		Topic: p.topic,
		Key:   sarama.StringEncoder(event.TransactionID),
		Value: sarama.ByteEncoder(eventData),
	}

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

func (p *KafkaProducer) Close() error {
	p.log.Info("закрытие kafka producer")
	return p.producer.Close()
}

type NoOpProducer struct {
	log *slog.Logger
}

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
