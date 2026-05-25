.PHONY: help run test lint tidy migrate-up migrate-down build docker-up docker-down

# Цвета для вывода
GREEN := \033[0;32m
NC := \033[0m # No Color

help:
	@echo "$(GREEN)Available commands:$(NC)"
	@echo "  make run          - Запустить бота локально"
	@echo "  make test         - Запустить тесты с детектором гонок"
	@echo "  make lint         - Запустить линтер"
	@echo "  make tidy         - Очистить go.mod"
	@echo "  make migrate-up   - Применить миграции к БД"
	@echo "  make migrate-down - Откатить последнюю миграцию"
	@echo "  make build        - Собрать бинарник"
	@echo "  make docker-up    - Поднять инфраструктуру (БД)"
	@echo "  make docker-down  - Остановить инфраструктуру"

run:
	@echo "$(GREEN)🚀 Запуск бота...$(NC)"
	go run ./cmd/bot

test:
	@echo "$(GREEN)🧪 Запуск тестов...$(NC)"
	go test ./... -race -cover -v

lint:
	@echo "$(GREEN)🔍 Запуск линтера...$(NC)"
	golangci-lint run ./...

tidy:
	@echo "$(GREEN)🧹 Очистка go.mod...$(NC)"
	go mod tidy

migrate-up:
	@echo "$(GREEN)📈 Применение миграций...$(NC)"
	@if [ -z "$(DATABASE_URL)" ]; then \
		echo "❌ DATABASE_URL не установлен. Загрузите .env файл или установите переменную окружения"; \
		exit 1; \
	fi
	migrate -path ./migrations -database "$(DATABASE_URL)" up

migrate-down:
	@echo "$(GREEN)📉 Откат миграции...$(NC)"
	@if [ -z "$(DATABASE_URL)" ]; then \
		echo "❌ DATABASE_URL не установлен"; \
		exit 1; \
	fi
	migrate -path ./migrations -database "$(DATABASE_URL)" down -limit 1

build:
	@echo "$(GREEN)🔨 Сборка бинарника...$(NC)"
	CGO_ENABLED=0 GOOS=linux go build -o bin/bot ./cmd/bot

docker-up:
	@echo "$(GREEN)🐳 Поднимаем инфраструктуру...$(NC)"
	docker-compose up -d postgres

docker-down:
	@echo "$(GREEN)🛑 Останавливаем инфраструктуру...$(NC)"
	docker-compose down