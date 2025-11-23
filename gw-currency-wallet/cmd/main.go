package main

import (
	_ "gw-currency-wallet/docs"
	"gw-currency-wallet/internal/app"
	"log"
)

// @title           Currency Wallet API
// @version         1.0
// @description     API для управления мультивалютным кошельком с поддержкой обмена валют
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	app, err := app.NewApp()
	if err != nil {
		log.Fatalf("Ошибка создания приложения: %v", err)
	}

	//слои приложения
	app.BuildAuthLayer()
	app.BuildWalletLayer()
	app.BuildExchangeLayer()

	if err := app.Run(); err != nil {
		log.Fatalf("Ошибка при работе приложения: %v", err)
	}
}
