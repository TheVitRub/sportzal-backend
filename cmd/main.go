package main

import (
	"log"

	"workout-app/backend/internal/app"
	"workout-app/backend/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("не удалось загрузить конфигурацию: %v", err)
	}

	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("не удалось инициализировать приложение: %v", err)
	}

	if err := application.Run(); err != nil {
		log.Fatalf("ошибка запуска приложения: %v", err)
	}
}
