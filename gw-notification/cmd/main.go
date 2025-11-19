package main

import (
	"gw-notification/internal/app"
	"log"
)

func main() {
	app, err := app.NewApp()
	if err != nil {
		log.Fatalf("не удалось создать приложение: %v", err)
	}

	if err := app.Run(); err != nil {
		log.Fatalf("ошибка при запуске приложения: %v", err)
	}
}
