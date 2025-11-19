package storage

import (
	"context"
	"gw-notification/internal/models"
)

// Storage интерфейс для работы с хранилищем уведомлений
type Storage interface {
	// SaveNotification сохраняет уведомление о крупном переводе
	SaveNotification(ctx context.Context, notification *models.LargeTransferNotification) error

	// GetNotificationByTransactionID получает уведомление по ID транзакции (для проверки дубликатов)
	GetNotificationByTransactionID(ctx context.Context, transactionID string) (*models.LargeTransferNotification, error)

	// Close закрывает соединение с хранилищем
	Close(ctx context.Context) error
}
