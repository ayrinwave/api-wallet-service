package mongodb

import (
	"context"
	"errors"
	"fmt"
	"gw-notification/internal/models"
	"gw-notification/internal/storage"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoStorage struct {
	client     *mongo.Client
	database   *mongo.Database
	collection *mongo.Collection
}

// NewMongoStorage создает новое подключение к MongoDB
func NewMongoStorage(ctx context.Context, uri, database, collection string, timeout time.Duration) (storage.Storage, error) {
	// Настройка клиента
	clientOpts := options.Client().
		ApplyURI(uri).
		SetConnectTimeout(timeout).
		SetServerSelectionTimeout(timeout)

	// Подключение к MongoDB
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Проверка соединения
	ctxPing, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := client.Ping(ctxPing, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := client.Database(database)
	coll := db.Collection(collection)

	// Создаем уникальный индекс для transaction_id (защита от дубликатов)
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "transaction_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	ctxIndex, cancelIndex := context.WithTimeout(ctx, timeout)
	defer cancelIndex()

	if _, err := coll.Indexes().CreateOne(ctxIndex, indexModel); err != nil {
		return nil, fmt.Errorf("failed to create index: %w", err)
	}

	return &MongoStorage{
		client:     client,
		database:   db,
		collection: coll,
	}, nil
}

// SaveNotification сохраняет уведомление в MongoDB
func (s *MongoStorage) SaveNotification(ctx context.Context, notification *models.LargeTransferNotification) error {
	// Устанавливаем время обработки
	notification.ProcessedAt = time.Now()

	// Вставляем документ
	_, err := s.collection.InsertOne(ctx, notification)
	if err != nil {
		// Проверяем, не дубликат ли это
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("notification with transaction_id %s already exists", notification.TransactionID)
		}
		return fmt.Errorf("failed to save notification: %w", err)
	}

	return nil
}

// GetNotificationByTransactionID получает уведомление по transaction_id
func (s *MongoStorage) GetNotificationByTransactionID(ctx context.Context, transactionID string) (*models.LargeTransferNotification, error) {
	var notification models.LargeTransferNotification

	filter := bson.M{"transaction_id": transactionID}
	err := s.collection.FindOne(ctx, filter).Decode(&notification)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("notification not found")
		}
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	return &notification, nil
}

// Close закрывает соединение с MongoDB
func (s *MongoStorage) Close(ctx context.Context) error {
	if s.client != nil {
		return s.client.Disconnect(ctx)
	}
	return nil
}
