DC := docker compose
TEST_DC := $(DC) -f compose.test.yaml

# DSN по умолчанию для локальных интеграционных тестов (compose.test.yaml).
TEST_DATABASE_URI ?= postgres://gophprofile:userpassword@localhost:15432/gophprofile_test?sslmode=disable
TEST_FILE_STORAGE_ENDPOINT ?= localhost:19000
TEST_FILE_STORAGE_ACCESS_KEY ?= gophprofile
TEST_FILE_STORAGE_SECRET_KEY ?= userpassword
TEST_FILE_STORAGE_BUCKET ?= goph-profile-test
TEST_QUEUE_URL ?= amqp://gophprofile:userpassword@localhost:15672/

.PHONY: help fmt test lint vet up down integration-up integration-down test-integration

help: ## Показать доступные цели
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z0-9_.-]+:.*##/ {printf "%-22s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

fmt: ## Отформатировать Go-файлы
	gofmt -w ./cmd ./internal ./migrations ./web

test: ## Запустить unit-тесты
	go test ./...

lint: ## Запустить линтер golangci-lint
	golangci-lint run

vet: ## Запустить базовые статические проверки go vet
	go vet ./...

up: ## Запустить контейнеры Docker Compose
	$(DC) up -d

down: ## Остановить и удалить контейнеры Docker Compose
	$(DC) down

# --- Интеграционная инфраструктура (Docker Compose) ---

integration-up: ## Запустить изолированные PostgreSQL, MinIO и RabbitMQ для интеграционных тестов
	$(TEST_DC) up -d --wait

integration-down: ## Остановить и удалить контейнеры PostgreSQL, MinIO и RabbitMQ для интеграционных тестов
	$(TEST_DC) down

# --- Интеграционные тесты ---

test-integration: integration-up ## Запустить интеграционные тесты
	TEST_DATABASE_URI='$(TEST_DATABASE_URI)' \
	TEST_FILE_STORAGE_ENDPOINT='$(TEST_FILE_STORAGE_ENDPOINT)' \
	TEST_FILE_STORAGE_ACCESS_KEY='$(TEST_FILE_STORAGE_ACCESS_KEY)' \
	TEST_FILE_STORAGE_SECRET_KEY='$(TEST_FILE_STORAGE_SECRET_KEY)' \
	TEST_FILE_STORAGE_BUCKET='$(TEST_FILE_STORAGE_BUCKET)' \
	TEST_QUEUE_URL='$(TEST_QUEUE_URL)' \
	go test -p 1 -tags=integration ./internal/...
