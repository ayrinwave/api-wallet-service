# Makefile для gw-exchanger

# Переменные
APP_NAME=gw-exchanger
MAIN_FILE=cmd/main.go
BINARY_NAME=exchanger-service
DOCKER_IMAGE=$(APP_NAME):latest

# Go команды
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GORUN=$(GOCMD) run

# Protobuf
PROTO_DIR=proto-exchange
PROTO_FILE=$(PROTO_DIR)/exchange.proto
PROTOC=protoc

# Database
DB_HOST=localhost
DB_PORT=5433
DB_USER=postgres
DB_PASSWORD=1234
DB_NAME=exchanger
DB_URL=postgresql://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

.PHONY: all build clean test run docker-build docker-run help

## help: Показать справку
help:
	@echo "Доступные команды:"
	@echo "  make run              - Запустить приложение локально"
	@echo "  make build            - Собрать бинарный файл"
	@echo "  make test             - Запустить тесты"
	@echo "  make clean            - Очистить сборку"
	@echo "  make deps             - Установить зависимости"
	@echo "  make proto            - Сгенерировать gRPC код из proto"
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
	rm -f exchanger.log

## test: Запустить тесты
test:
	$(GOTEST) -v ./...

## test-coverage: Запустить тесты с покрытием
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Отчет о покрытии: coverage.html"

## deps: Установить зависимости
deps:
	$(GOGET) -v ./...
	$(GOMOD) download
	$(GOMOD) tidy

## proto: Сгенерировать gRPC код из proto файла
proto:
	@which $(PROTOC) > /dev/null || (echo "protoc не установлен. Установите: https://grpc.io/docs/protoc-installation/" && exit 1)
	$(PROTOC) --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		$(PROTO_FILE)
	@echo "gRPC код сгенерирован из $(PROTO_FILE)"

## proto-install: Установить protoc плагины
proto-install:
	$(GOGET) google.golang.org/protobuf/cmd/protoc-gen-go@latest
	$(GOGET) google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

## migrate-up: Применить миграции
migrate-up:
	@which migrate > /dev/null || (echo "migrate не установлен. Установите: https://github.com/golang-migrate/migrate" && exit 1)
	migrate -path migrations -database "$(DB_URL)" up

## migrate-down: Откатить миграции
migrate-down:
	@which migrate > /dev/null || (echo "migrate не установлен" && exit 1)
	migrate -path migrations -database "$(DB_URL)" down

## migrate-create: Создать новую миграцию
migrate-create:
	@if [ -z "$(name)" ]; then echo "Укажите имя: make migrate-create name=my_migration"; exit 1; fi
	@which migrate > /dev/null || (echo "migrate не установлен" && exit 1)
	migrate create -ext sql -dir migrations -seq $(name)

## docker-build: Собрать Docker образ
docker-build:
	docker build -t $(DOCKER_IMAGE) -f Dockerfile ..

## docker-run: Запустить Docker контейнер
docker-run:
	docker run -d --name $(APP_NAME) -p 50051:50051 --env-file config.env $(DOCKER_IMAGE)

## docker-stop: Остановить Docker контейнер
docker-stop:
	docker stop $(APP_NAME) && docker rm $(APP_NAME)

## lint: Запустить линтер
lint:
	@which golangci-lint > /dev/null || (echo "golangci-lint не установлен" && exit 1)
	golangci-lint run ./...

## fmt: Форматировать код
fmt:
	$(GOCMD) fmt ./...
	gofmt -s -w .

## vet: Запустить go vet
vet:
	$(GOCMD) vet ./...

## dev: Запустить в режиме разработки
dev:
	@which air > /dev/null || (echo "air не установлен. Установите: go install github.com/cosmtrek/air@latest" && exit 1)
	air

.DEFAULT_GOAL := help