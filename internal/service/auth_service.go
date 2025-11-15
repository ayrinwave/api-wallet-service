package service

import (
	"api_wallet/internal/custom_err"
	"api_wallet/internal/models"
	"api_wallet/internal/repository/postgres"
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// Auth интерфейс для работы с аутентификацией и авторизацией
type Auth interface {
	Register(ctx context.Context, req models.RegisterRequest) (*models.RegisterResponse, error)
	Login(ctx context.Context, req models.LoginRequest) (*models.LoginResponse, error)
	ValidateToken(tokenString string) (*models.JWTClaims, error)
}
type AuthService struct {
	userRepo      postgres.UserRepository
	walletRepo    postgres.WalletRepository
	txManager     TxManager
	jwtSecret     []byte
	jwtExpiration time.Duration
	log           *slog.Logger
}

func NewAuthService(
	userRepo postgres.UserRepository,
	walletRepo postgres.WalletRepository,
	txManager TxManager,
	jwtSecret string,
	jwtExpiration time.Duration,
	log *slog.Logger,
) Auth { // ← Возвращаем интерфейс, а не конкретный тип
	return &AuthService{
		userRepo:      userRepo,
		walletRepo:    walletRepo,
		txManager:     txManager,
		jwtSecret:     []byte(jwtSecret),
		jwtExpiration: jwtExpiration,
		log:           log,
	}
}

// Register регистрирует нового пользователя и создает 3 кошелька (USD, RUB, EUR)
func (s *AuthService) Register(ctx context.Context, req models.RegisterRequest) (*models.RegisterResponse, error) {
	const op = "service.Register"

	// Валидация входных данных
	if req.Username == "" || req.Password == "" || req.Email == "" {
		return nil, custom_err.ErrInvalidInput
	}

	// Проверяем существование username
	exists, err := s.userRepo.ExistsByUsername(ctx, req.Username)
	if err != nil {
		s.log.Error("failed to check username existence", slog.String("op", op), slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if exists {
		return nil, custom_err.ErrUsernameExists
	}

	// Проверяем существование email
	exists, err = s.userRepo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		s.log.Error("failed to check email existence", slog.String("op", op), slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if exists {
		return nil, custom_err.ErrEmailExists
	}

	// Хешируем пароль
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.log.Error("failed to hash password", slog.String("op", op), slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: failed to hash password: %w", op, err)
	}

	// Создаем пользователя и кошельки в транзакции
	user := &models.User{
		ID:           uuid.New(),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
	}

	err = s.txManager.WithTx(ctx, func(tx pgx.Tx) error {
		// Создаем пользователя
		createdUser, err := s.userRepo.CreateTx(ctx, tx, user)
		if err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}

		// Создаем 3 кошелька (USD, RUB, EUR)
		currencies := models.SupportedCurrencies()
		for _, currency := range currencies {
			_, err := s.walletRepo.CreateWalletTx(ctx, tx, createdUser.ID, currency)
			if err != nil {
				return fmt.Errorf("failed to create %s wallet: %w", currency, err)
			}
		}

		s.log.Info("user registered successfully",
			slog.String("op", op),
			slog.String("user_id", createdUser.ID.String()),
			slog.String("username", createdUser.Username))

		return nil
	})

	if err != nil {
		s.log.Error("failed to register user", slog.String("op", op), slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &models.RegisterResponse{
		Message: "User registered successfully",
	}, nil
}

// Login авторизует пользователя и возвращает JWT токен
func (s *AuthService) Login(ctx context.Context, req models.LoginRequest) (*models.LoginResponse, error) {
	const op = "service.Login"

	// Валидация
	if req.Username == "" || req.Password == "" {
		return nil, custom_err.ErrInvalidInput
	}

	// Получаем пользователя
	user, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		if err == custom_err.ErrNotFound {
			return nil, custom_err.ErrInvalidCredentials
		}
		s.log.Error("failed to get user", slog.String("op", op), slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Проверяем пароль
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		return nil, custom_err.ErrInvalidCredentials
	}

	// Генерируем JWT токен
	token, err := s.generateJWT(user)
	if err != nil {
		s.log.Error("failed to generate JWT", slog.String("op", op), slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	s.log.Info("user logged in successfully",
		slog.String("op", op),
		slog.String("user_id", user.ID.String()),
		slog.String("username", user.Username))

	return &models.LoginResponse{
		Token: token,
	}, nil
}

// ValidateToken валидирует JWT токен и возвращает claims
func (s *AuthService) ValidateToken(tokenString string) (*models.JWTClaims, error) {
	const op = "service.ValidateToken"

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Проверяем метод подписи
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, custom_err.ErrInvalidToken
	}

	if !token.Valid {
		return nil, custom_err.ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, custom_err.ErrInvalidToken
	}

	// Извлекаем данные из claims
	userIDStr, ok := claims["user_id"].(string)
	if !ok {
		return nil, custom_err.ErrInvalidToken
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, custom_err.ErrInvalidToken
	}

	username, ok := claims["username"].(string)
	if !ok {
		return nil, custom_err.ErrInvalidToken
	}

	return &models.JWTClaims{
		UserID:   userID,
		Username: username,
	}, nil
}

// generateJWT генерирует JWT токен для пользователя
func (s *AuthService) generateJWT(user *models.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  user.ID.String(),
		"username": user.Username,
		"exp":      time.Now().Add(s.jwtExpiration).Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}
