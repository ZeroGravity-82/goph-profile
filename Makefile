DC := docker compose
TEST_ENV_FILE ?= .env.test
TEST_DC := $(DC) --env-file $(TEST_ENV_FILE) -f compose.test.yaml

.PHONY: help fmt test lint vet up down check-test-env integration-up integration-down test-integration

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

up: ## Собрать образ и запустить локальный стек
	$(DC) up -d --build --wait

down: ## Остановить и удалить контейнеры Docker Compose
	$(DC) down

# --- Интеграционная инфраструктура (Docker Compose) ---

check-test-env: ## Проверить .env.test
	@test -f $(TEST_ENV_FILE) || (echo "create $(TEST_ENV_FILE) from .env.test.example"; exit 1)

integration-up: check-test-env ## Запустить интеграционное окружение
	$(TEST_DC) up -d --wait

integration-down: check-test-env ## Остановить интеграционное окружение
	$(TEST_DC) down

# --- Интеграционные тесты ---

test-integration: integration-up ## Запустить интеграционные тесты
	env_file='$(TEST_ENV_FILE)'; case "$$env_file" in /*) ;; *) env_file="./$$env_file";; esac; \
	set -a; . "$$env_file"; set +a; \
	test_database_uri="postgres://$${GOPH_PROFILE_TEST_POSTGRES_USER}:$${GOPH_PROFILE_TEST_POSTGRES_PASSWORD}"; \
	test_database_uri="$$test_database_uri@localhost:15432/$${GOPH_PROFILE_TEST_POSTGRES_DB}?sslmode=disable"; \
	TEST_DATABASE_URI="$$test_database_uri" \
	TEST_FILE_STORAGE_ENDPOINT="localhost:19000" \
	TEST_FILE_STORAGE_ACCESS_KEY="$${GOPH_PROFILE_TEST_MINIO_ROOT_USER}" \
	TEST_FILE_STORAGE_SECRET_KEY="$${GOPH_PROFILE_TEST_MINIO_ROOT_PASSWORD}" \
	TEST_FILE_STORAGE_BUCKET="goph-profile-test" \
	TEST_QUEUE_URL="amqp://$${GOPH_PROFILE_TEST_RABBITMQ_USER}:$${GOPH_PROFILE_TEST_RABBITMQ_PASSWORD}@localhost:15673/" \
	go test -p 1 -tags=integration ./internal/...
