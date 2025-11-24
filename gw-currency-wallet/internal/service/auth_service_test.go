package service

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"

	"gw-currency-wallet/internal/custom_err"
	"gw-currency-wallet/internal/models"
)

func setupAuthService() (*AuthService, *MockUserRepository, *MockWalletRepo, *MockTxManager) {
	userRepo := new(MockUserRepository)
	walletRepo := new(MockWalletRepo)
	txManager := new(MockTxManager)

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	service := &AuthService{
		userRepo:      userRepo,
		walletRepo:    walletRepo,
		txManager:     txManager,
		jwtSecret:     []byte("test-secret"),
		jwtExpiration: time.Hour,
		log:           log,
	}

	return service, userRepo, walletRepo, txManager
}

func TestAuthService_Register_Success(t *testing.T) {
	service, userRepo, walletRepo, txManager := setupAuthService()
	ctx := context.Background()

	req := models.RegisterRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}

	userRepo.On("ExistsByUsername", ctx, req.Username).Return(false, nil)
	userRepo.On("ExistsByEmail", ctx, req.Email).Return(false, nil)

	userRepo.On("CreateTx", ctx, mock.Anything, mock.AnythingOfType("*models.User")).
		Return(&models.User{
			ID:       uuid.New(),
			Username: req.Username,
			Email:    req.Email,
		}, nil)

	for _, currency := range models.SupportedCurrencies() {
		walletRepo.On("CreateWalletTx", ctx, mock.Anything, mock.AnythingOfType("uuid.UUID"), currency).
			Return(&models.Wallet{
				ID:       uuid.New(),
				UserID:   uuid.New(),
				Currency: string(currency),
				Balance:  0,
			}, nil)
	}

	txManager.On("WithTx", ctx, mock.AnythingOfType("func(pgx.Tx) error")).Return(nil)

	resp, err := service.Register(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "User registered successfully", resp.Message)

	userRepo.AssertExpectations(t)
	walletRepo.AssertExpectations(t)
	txManager.AssertExpectations(t)
}

func TestAuthService_Register_UsernameExists(t *testing.T) {
	service, userRepo, _, _ := setupAuthService()
	ctx := context.Background()

	req := models.RegisterRequest{
		Username: "existinguser",
		Email:    "test@example.com",
		Password: "password123",
	}

	userRepo.On("ExistsByUsername", ctx, req.Username).Return(true, nil)

	// Выполняем тест
	resp, err := service.Register(ctx, req)

	// Проверки
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, custom_err.ErrUsernameExists, err)

	userRepo.AssertExpectations(t)
}

func TestAuthService_Register_EmailExists(t *testing.T) {
	service, userRepo, _, _ := setupAuthService()
	ctx := context.Background()

	req := models.RegisterRequest{
		Username: "testuser",
		Email:    "existing@example.com",
		Password: "password123",
	}

	// Мокаем проверки
	userRepo.On("ExistsByUsername", ctx, req.Username).Return(false, nil)
	userRepo.On("ExistsByEmail", ctx, req.Email).Return(true, nil)

	// Выполняем тест
	resp, err := service.Register(ctx, req)

	// Проверки
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, custom_err.ErrEmailExists, err)

	userRepo.AssertExpectations(t)
}

func TestAuthService_Register_InvalidInput(t *testing.T) {
	service, _, _, _ := setupAuthService()
	ctx := context.Background()

	tests := []struct {
		name string
		req  models.RegisterRequest
	}{
		{
			name: "empty username",
			req: models.RegisterRequest{
				Username: "",
				Email:    "test@example.com",
				Password: "password123",
			},
		},
		{
			name: "empty email",
			req: models.RegisterRequest{
				Username: "testuser",
				Email:    "",
				Password: "password123",
			},
		},
		{
			name: "empty password",
			req: models.RegisterRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.Register(ctx, tt.req)

			assert.Error(t, err)
			assert.Nil(t, resp)
			assert.Equal(t, custom_err.ErrInvalidInput, err)
		})
	}
}

// Тесты для Login
func TestAuthService_Login_Success(t *testing.T) {
	service, userRepo, _, _ := setupAuthService()
	ctx := context.Background()

	password := "password123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	user := &models.User{
		ID:           uuid.New(),
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: string(hashedPassword),
	}

	req := models.LoginRequest{
		Username: "testuser",
		Password: password,
	}

	// Мокаем получение пользователя
	userRepo.On("GetByUsername", ctx, req.Username).Return(user, nil)

	// Выполняем тест
	resp, err := service.Login(ctx, req)

	// Проверки
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.Token)

	// Проверяем, что токен валидный
	claims, err := service.ValidateToken(resp.Token)
	assert.NoError(t, err)
	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, user.Username, claims.Username)

	userRepo.AssertExpectations(t)
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	service, userRepo, _, _ := setupAuthService()
	ctx := context.Background()

	req := models.LoginRequest{
		Username: "nonexistent",
		Password: "password123",
	}

	// Мокаем получение пользователя - не найден
	userRepo.On("GetByUsername", ctx, req.Username).Return(nil, custom_err.ErrNotFound)

	// Выполняем тест
	resp, err := service.Login(ctx, req)

	// Проверки
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, custom_err.ErrInvalidCredentials, err)

	userRepo.AssertExpectations(t)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	service, userRepo, _, _ := setupAuthService()
	ctx := context.Background()

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)

	user := &models.User{
		ID:           uuid.New(),
		Username:     "testuser",
		PasswordHash: string(hashedPassword),
	}

	req := models.LoginRequest{
		Username: "testuser",
		Password: "wrongpassword",
	}

	// Мокаем получение пользователя
	userRepo.On("GetByUsername", ctx, req.Username).Return(user, nil)

	// Выполняем тест
	resp, err := service.Login(ctx, req)

	// Проверки
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, custom_err.ErrInvalidCredentials, err)

	userRepo.AssertExpectations(t)
}

func TestAuthService_Login_InvalidInput(t *testing.T) {
	service, _, _, _ := setupAuthService()
	ctx := context.Background()

	tests := []struct {
		name string
		req  models.LoginRequest
	}{
		{
			name: "empty username",
			req: models.LoginRequest{
				Username: "",
				Password: "password123",
			},
		},
		{
			name: "empty password",
			req: models.LoginRequest{
				Username: "testuser",
				Password: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.Login(ctx, tt.req)

			assert.Error(t, err)
			assert.Nil(t, resp)
			assert.Equal(t, custom_err.ErrInvalidInput, err)
		})
	}
}

// Тесты для ValidateToken
func TestAuthService_ValidateToken_Success(t *testing.T) {
	service, _, _, _ := setupAuthService()

	userID := uuid.New()
	username := "testuser"

	// Создаем валидный токен
	user := &models.User{
		ID:       userID,
		Username: username,
	}

	token, err := service.generateJWT(user)
	assert.NoError(t, err)

	// Валидируем токен
	claims, err := service.ValidateToken(token)

	// Проверки
	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, username, claims.Username)
}

func TestAuthService_ValidateToken_InvalidToken(t *testing.T) {
	service, _, _, _ := setupAuthService()

	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "empty token",
			token: "",
		},
		{
			name:  "malformed token",
			token: "invalid.token.here",
		},
		{
			name:  "token with wrong signature",
			token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMTIzIiwidXNlcm5hbWUiOiJ0ZXN0In0.wrong_signature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := service.ValidateToken(tt.token)

			assert.Error(t, err)
			assert.Nil(t, claims)
			assert.Equal(t, custom_err.ErrInvalidToken, err)
		})
	}
}

func TestAuthService_ValidateToken_ExpiredToken(t *testing.T) {
	userRepo := new(MockUserRepository)
	walletRepo := new(MockWalletRepo)
	txManager := new(MockTxManager)
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Создаем сервис с коротким временем жизни токена
	service := &AuthService{
		userRepo:      userRepo,
		walletRepo:    walletRepo,
		txManager:     txManager,
		jwtSecret:     []byte("test-secret"),
		jwtExpiration: -time.Hour, // Токен уже истек
		log:           log,
	}

	user := &models.User{
		ID:       uuid.New(),
		Username: "testuser",
	}

	token, err := service.generateJWT(user)
	assert.NoError(t, err)

	// Валидируем истекший токен
	claims, err := service.ValidateToken(token)

	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.Equal(t, custom_err.ErrInvalidToken, err)
}
