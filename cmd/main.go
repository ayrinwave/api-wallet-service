package main

import (
	"api_wallet/internal/app"
	"log"
)

func main() {
	app, err := app.NewApp()
	if err != nil {
		log.Fatalf("Ошибка создания приложения: %v", err)
	}

	app.BuildWalletLayer()
	if err := app.Run(); err != nil {
		log.Fatalf("Ошибка при работе приложения: %v", err)
	}
}
