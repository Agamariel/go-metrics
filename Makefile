.PHONY: all build test run-server run-agent clean help

# Переменные
SERVER_BIN := bin/server
AGENT_BIN := bin/agent

all: build test

# Сборка проектов
build: build-server build-agent

build-server:
	@echo "Сборка сервера..."
	@go build -o $(SERVER_BIN) ./cmd/server
	@echo "✓ Сервер собран: $(SERVER_BIN)"

build-agent:
	@echo "Сборка агента..."
	@go build -o $(AGENT_BIN) ./cmd/agent
	@echo "✓ Агент собран: $(AGENT_BIN)"

# Запуск
run-server:
	@echo "Запуск сервера..."
	@go run ./cmd/server

run-agent:
	@echo "Запуск агента..."
	@go run ./cmd/agent

# Тестирование
test:
	@echo "Запуск тестов..."
	@go test ./... -v

test-cover:
	@echo "Запуск тестов с покрытием (по пакетам)..."
	@go test ./... -cover

# Суммарное покрытие по всему модулю.
# Флаг -coverpkg=./... гарантирует, что тесты каждого пакета
# инструментируют весь модуль — только так итоговый процент
# отражает реальное общее покрытие.
test-cover-total:
	@echo "Подсчёт суммарного покрытия по всему модулю..."
	@go test ./... -coverprofile=coverage.out -coverpkg=./... -count=1
	@go tool cover -func=coverage.out | grep total

test-cover-html:
	@echo "Генерация HTML отчета о покрытии..."
	@go test ./... -coverprofile=coverage.out -coverpkg=./... -count=1
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Отчет сохранен в coverage.html"

# Проверка кода
lint:
	@echo "Проверка кода..."
	@go vet ./...
	@echo "✓ Проверка завершена"

fmt:
	@echo "Форматирование кода..."
	@go fmt ./...
	@echo "✓ Форматирование завершено"

# Очистка
clean:
	@echo "Очистка..."
	@rm -f $(SERVER_BIN) $(AGENT_BIN)
	@rm -f coverage.out coverage.html
	@echo "✓ Очистка завершена"

# Установка зависимостей
deps:
	@echo "Обновление зависимостей..."
	@go mod tidy
	@echo "✓ Зависимости обновлены"

# Помощь
help:
	@echo "Доступные команды:"
	@echo "  make build         - Собрать сервер и агент"
	@echo "  make build-server  - Собрать только сервер"
	@echo "  make build-agent   - Собрать только агент"
	@echo "  make run-server    - Запустить сервер"
	@echo "  make run-agent     - Запустить агент"
	@echo "  make test          - Запустить тесты"
	@echo "  make test-cover    - Запустить тесты с покрытием"
	@echo "  make test-cover-html - Создать HTML отчет о покрытии"
	@echo "  make lint          - Проверить код (go vet)"
	@echo "  make fmt           - Форматировать код (go fmt)"
	@echo "  make clean         - Удалить собранные файлы"
	@echo "  make deps          - Обновить зависимости"
	@echo "  make help          - Показать эту справку"
