package main

import (
	"gw-exchanger/internal/app"
	"log"
)

func main() {
	application, err := app.NewApp()
	if err != nil {
		log.Fatalf("Ошибка создания приложения: %v", err)
	}

	if err := application.Run(); err != nil {
		log.Fatalf("Ошибка при работе приложения: %v", err)
	}
}
