package custom_err

import "errors"

var (
	ErrNotFound           = errors.New("запись не найдена")
	ErrInsufficientFunds  = errors.New("недостаточно средств на счете")
	ErrDuplicateRequest   = errors.New("повторяющийся запрос")
	ErrMaxRetriesExceeded = errors.New("превышено максимальное число повторных попыток")
	ErrConflict           = errors.New("конфликт оптимистической блокировки")
)
