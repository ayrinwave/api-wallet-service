package main

import (
	"gw-currency-wallet/internal/app"
	"log"
)

// @title           Currency Wallet API
// @version         1.0
// @description     API для управления криптовалютным кошельком с поддержкой мультивалютности
// @description     Поддерживаемые валюты: USD, RUB, EUR
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@example.com

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
func main() {
	app, err := app.NewApp()
	if err != nil {
		log.Fatalf("Ошибка создания приложения: %v", err)
	}

	// Собираем все слои приложения
	app.BuildAuthLayer()
	app.BuildWalletLayer()
	app.BuildExchangeLayer()

	if err := app.Run(); err != nil {
		log.Fatalf("Ошибка при работе приложения: %v", err)
	}
}
