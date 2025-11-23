# Makefile для gw-currency-wallet

# Переменные
APP_NAME=gw-currency-wallet
MAIN_FILE=cmd/main.go
BINARY_NAME=wallet-service
DOCKER_IMAGE=$(APP_NAME):latest

# Go команды
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GORUN=$(GOCMD) run

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=1234
DB_NAME=wallet
DB_URL=postgresql://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

# Swagger
SWAG=swag

.PHONY: all build clean test run docker-build docker-run help

## help: Показать справку
help:
	@echo "Доступные команды:"
	@echo "  make run              - Запустить приложение локально"
	@echo "  make build            - Собрать бинарный файл"
	@echo "  make test             - Запустить все тесты"
	@echo "  make test-unit        - Запустить юнит-тесты"
	@echo "  make test-integration - Запустить интеграционные тесты"
	@echo "  make test-coverage    - Запустить тесты с покрытием"
	@echo "  make clean            - Очистить сборку"
	@echo "  make deps             - Установить зависимости"
	@echo "  make swagger          - Сгенерировать Swagger документацию"
	@echo "  make migrate-up       - Применить миграции"
	@echo "  make migrate-down     - Откатить миграции"
	@echo "  make docker-build     - Собрать Docker образ"
	@echo "  make docker-run       - Запустить Docker контейнер"
	@echo "  make lint             - Запустить линтер"
	@echo "  make fmt              - Форматировать код"

## run: Запустить приложение локально
run:
	$(GORUN) $(MAIN_FILE)

## build: Собрать бинарный файл
build:
	$(GOBUILD) -o bin/$(BINARY_NAME) -v $(MAIN_FILE)

## build-linux: Собрать для Linux
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o bin/$(BINARY_NAME)-linux -v $(MAIN_FILE)

## clean: Очистить сборку
clean:
	$(GOCLEAN)
	rm -rf bin/
	rm -f wallet.log

## test-all: Все тесты
test-all: test-unit test-integration

## test-unit: Юнит-тесты (без интеграционных)
test-unit:
	go test -v -short ./internal/service/...

## test-integration: Интеграционные тесты (требуется БД)
test-integration:
	go test -v -tags=integration ./internal/service/...

## test-coverage: Покрытие кода
test-coverage:
	go test -v -coverprofile=coverage.out ./internal/service/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Открыть coverage.html в браузере"

## deps: Установить зависимости
deps:
	$(GOGET) -v ./...
	$(GOMOD) download
	$(GOMOD) tidy

## swagger: Сгенерировать Swagger документацию
swagger:
	@which $(SWAG) > /dev/null || (echo "swag не установлен. Установите: go install github.com/swaggo/swag/cmd/swag@latest" && exit 1)
	rm -rf docs
	$(SWAG) init -g $(MAIN_FILE) --output ./docs --parseInternal --parseDependency
	@echo "Swagger документация сгенерирована в docs/"

## migrate-up: Применить миграции
migrate-up:
	@which migrate > /dev/null || (echo "migrate не установлен. Установите: https://github.com/golang-migrate/migrate" && exit 1)
	migrate -path migrations -database "$(DB_URL)" up

## migrate-down: Откатить миграции
migrate-down:
	@which migrate > /dev/null || (echo "migrate не установлен. Установите: https://github.com/golang-migrate/migrate" && exit 1)
	migrate -path migrations -database "$(DB_URL)" down

## migrate-create: Создать новую миграцию (использование: make migrate-create name=add_users_table)
migrate-create:
	@if [ -z "$(name)" ]; then echo "Укажите имя миграции: make migrate-create name=my_migration"; exit 1; fi
	@which migrate > /dev/null || (echo "migrate не установлен" && exit 1)
	migrate create -ext sql -dir migrations -seq $(name)

## docker-build: Собрать Docker образ
docker-build:
	docker build -t $(DOCKER_IMAGE) -f Dockerfile ..

## docker-run: Запустить Docker контейнер
docker-run:
	docker run -d --name $(APP_NAME) -p 8080:8080 --env-file config.env $(DOCKER_IMAGE)

## docker-stop: Остановить Docker контейнер
docker-stop:
	docker stop $(APP_NAME) && docker rm $(APP_NAME)

## lint: Запустить линтер
lint:
	@which golangci-lint > /dev/null || (echo "golangci-lint не установлен. Установите: https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run ./...

## fmt: Форматировать код
fmt:
	$(GOCMD) fmt ./...
	gofmt -s -w .

## vet: Запустить go vet
vet:
	$(GOCMD) vet ./...

## mod-verify: Проверить зависимости
mod-verify:
	$(GOMOD) verify

## dev: Запустить в режиме разработки (с auto-reload, требует air)
dev:
	@which air > /dev/null || (echo "air не установлен. Установите: go install github.com/cosmtrek/air@latest" && exit 1)
	air

## proto: Генерация protobuf (если используется)
proto:
	@echo "Protobuf не используется в этом сервисе"

.DEFAULT_GOAL := help