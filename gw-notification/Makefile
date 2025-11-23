# Makefile для gw-notification

# Переменные
APP_NAME=gw-notification
MAIN_FILE=cmd/main.go
BINARY_NAME=notification-service
DOCKER_IMAGE=$(APP_NAME):latest

# Go команды
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GORUN=$(GOCMD) run

# MongoDB
MONGO_URI=mongodb://admin:admin123@localhost:27017
MONGO_DB=notifications

# Kafka
KAFKA_BROKERS=localhost:9092
KAFKA_TOPIC=large-transfers

.PHONY: all build clean test run docker-build docker-run help

## help: Показать справку
help:
	@echo "Доступные команды:"
	@echo "  make run              - Запустить приложение локально"
	@echo "  make build            - Собрать бинарный файл"
	@echo "  make test             - Запустить тесты"
	@echo "  make clean            - Очистить сборку"
	@echo "  make deps             - Установить зависимости"
	@echo "  make docker-build     - Собрать Docker образ"
	@echo "  make docker-run       - Запустить Docker контейнер"
	@echo "  make kafka-create     - Создать Kafka топик"
	@echo "  make kafka-consume    - Просмотреть сообщения из Kafka"
	@echo "  make mongo-shell      - Открыть MongoDB shell"
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
	rm -f notification.log

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

## kafka-create: Создать Kafka топик
kafka-create:
	docker exec -it kafka kafka-topics --create \
		--bootstrap-server localhost:9092 \
		--topic $(KAFKA_TOPIC) \
		--partitions 3 \
		--replication-factor 1 \
		--if-not-exists

## kafka-list: Список Kafka топиков
kafka-list:
	docker exec -it kafka kafka-topics --list --bootstrap-server localhost:9092

## kafka-consume: Просмотреть сообщения из Kafka (для отладки)
kafka-consume:
	docker exec -it kafka kafka-console-consumer \
		--bootstrap-server localhost:9092 \
		--topic $(KAFKA_TOPIC) \
		--from-beginning

## kafka-describe: Информация о топике
kafka-describe:
	docker exec -it kafka kafka-topics --describe \
		--bootstrap-server localhost:9092 \
		--topic $(KAFKA_TOPIC)

## mongo-shell: Открыть MongoDB shell
mongo-shell:
	docker exec -it mongodb mongosh $(MONGO_URI)

## mongo-show: Показать документы из коллекции
mongo-show:
	docker exec -it mongodb mongosh $(MONGO_URI) \
		--eval "db.large_transfers.find().pretty()"

## mongo-count: Количество документов
mongo-count:
	docker exec -it mongodb mongosh $(MONGO_URI) \
		--eval "db.large_transfers.countDocuments()"

## mongo-drop: Очистить коллекцию (ОСТОРОЖНО!)
mongo-drop:
	@echo "ВНИМАНИЕ: Это удалит все данные из коллекции!"
	@read -p "Продолжить? [y/N]: " confirm && [ "$$confirm" = "y" ] || exit 1
	docker exec -it mongodb mongosh $(MONGO_URI) \
		--eval "db.large_transfers.drop()"

## docker-build: Собрать Docker образ
docker-build:
	docker build -t $(DOCKER_IMAGE) -f Dockerfile ..

## docker-run: Запустить Docker контейнер
docker-run:
	docker run -d --name $(APP_NAME) --env-file config.env $(DOCKER_IMAGE)

## docker-stop: Остановить Docker контейнер
docker-stop:
	docker stop $(APP_NAME) && docker rm $(APP_NAME)

## docker-logs: Показать логи контейнера
docker-logs:
	docker logs -f $(APP_NAME)

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