# Корневой Makefile для управления всеми микросервисами

.PHONY: help up down restart logs build test clean

## help: Показать справку
help:
	@echo "=== Управление всем проектом ==="
	@echo ""
	@echo "Инфраструктура:"
	@echo "  make up               - Запустить все сервисы (docker-compose)"
	@echo "  make down             - Остановить все сервисы"
	@echo "  make restart          - Перезапустить все сервисы"
	@echo "  make logs             - Показать логи всех сервисов"
	@echo "  make logs-wallet      - Логи gw-currency-wallet"
	@echo "  make logs-exchanger   - Логи gw-exchanger"
	@echo "  make logs-notification- Логи gw-notification"
	@echo "  make ps               - Статус всех контейнеров"
	@echo ""
	@echo "Инфраструктура (только БД):"
	@echo "  make infra-up         - Запустить только БД и Kafka"
	@echo "  make infra-down       - Остановить инфраструктуру"
	@echo ""
	@echo "Сборка:"
	@echo "  make build            - Собрать Docker образы для всех сервисов"
	@echo "  make build-wallet     - Собрать образ gw-currency-wallet"
	@echo "  make build-exchanger  - Собрать образ gw-exchanger"
	@echo "  make build-notification - Собрать образ gw-notification"
	@echo ""
	@echo "Тестирование:"
	@echo "  make test             - Запустить тесты всех сервисов"
	@echo "  make test-wallet      - Тесты gw-currency-wallet"
	@echo "  make test-exchanger   - Тесты gw-exchanger"
	@echo "  make test-notification- Тесты gw-notification"
	@echo ""
	@echo "Очистка:"
	@echo "  make clean            - Очистить все сервисы"
	@echo "  make clean-volumes    - Удалить volumes (ОСТОРОЖНО!)"
	@echo ""
	@echo "Разработка:"
	@echo "  make dev-wallet       - Запустить wallet локально"
	@echo "  make dev-exchanger    - Запустить exchanger локально"
	@echo "  make dev-notification - Запустить notification локально"

## up: Запустить все сервисы
up:
	docker-compose up -d

## down: Остановить все сервисы
down:
	docker-compose down

## restart: Перезапустить все сервисы
restart:
	docker-compose restart

## logs: Показать логи всех сервисов
logs:
	docker-compose logs -f

## logs-wallet: Логи gw-currency-wallet
logs-wallet:
	docker-compose logs -f wallet

## logs-exchanger: Логи gw-exchanger
logs-exchanger:
	docker-compose logs -f exchanger

## logs-notification: Логи gw-notification
logs-notification:
	docker-compose logs -f notification

## ps: Статус контейнеров
ps:
	docker-compose ps

## infra-up: Запустить только инфраструктуру (БД, Kafka, MongoDB)
infra-up:
	docker-compose up -d postgres-wallet postgres-exchanger kafka zookeeper mongodb

## infra-down: Остановить инфраструктуру
infra-down:
	docker-compose stop postgres-wallet postgres-exchanger kafka zookeeper mongodb

## build: Собрать все Docker образы
build:
	docker-compose build

## build-wallet: Собрать образ gw-currency-wallet
build-wallet:
	docker-compose build wallet

## build-exchanger: Собрать образ gw-exchanger
build-exchanger:
	docker-compose build exchanger

## build-notification: Собрать образ gw-notification
build-notification:
	docker-compose build notification

## test: Запустить тесты всех сервисов
test: test-wallet test-exchanger test-notification

## test-wallet: Тесты gw-currency-wallet
test-wallet:
	cd gw-currency-wallet && make test

## test-exchanger: Тесты gw-exchanger
test-exchanger:
	cd gw-exchanger && make test

## test-notification: Тесты gw-notification
test-notification:
	cd gw-notification && make test

## clean: Очистить все сервисы
clean:
	cd gw-currency-wallet && make clean
	cd gw-exchanger && make clean
	cd gw-notification && make clean
	docker-compose down

## clean-volumes: Удалить все volumes (ОСТОРОЖНО!)
clean-volumes:
	@echo "ВНИМАНИЕ: Это удалит ВСЕ данные из баз данных!"
	@read -p "Продолжить? [y/N]: " confirm && [ "$$confirm" = "y" ] || exit 1
	docker-compose down -v

## dev-wallet: Запустить wallet локально (с БД в Docker)
dev-wallet:
	@echo "Запуск gw-currency-wallet локально..."
	cd gw-currency-wallet && make run

## dev-exchanger: Запустить exchanger локально
dev-exchanger:
	@echo "Запуск gw-exchanger локально..."
	cd gw-exchanger && make run

## dev-notification: Запустить notification локально
dev-notification:
	@echo "Запуск gw-notification локально..."
	cd gw-notification && make run

## proto: Генерация protobuf для exchanger
proto:
	cd gw-exchanger && make proto

## swagger: Генерация Swagger для wallet
swagger:
	cd gw-currency-wallet && make swagger

## fmt: Форматировать код всех сервисов
fmt:
	cd gw-currency-wallet && make fmt
	cd gw-exchanger && make fmt
	cd gw-notification && make fmt

## lint: Запустить линтер для всех сервисов
lint:
	cd gw-currency-wallet && make lint
	cd gw-exchanger && make lint
	cd gw-notification && make lint

## deps: Обновить зависимости всех сервисов
deps:
	cd gw-currency-wallet && make deps
	cd gw-exchanger && make deps
	cd gw-notification && make deps

.DEFAULT_GOAL := help