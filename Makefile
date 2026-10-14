SHELL := /bin/bash
-include .env
export

APP_NAME       := task-management-service
BIN_DIR        := bin
MAIN           := ./cmd/api
MIGRATIONS_DIR := db/migrations

DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/taskdb?sslmode=disable

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build the API binary
	go build -o $(BIN_DIR)/api $(MAIN)

.PHONY: run
run: ## Run the API without hot reload
	go run $(MAIN)

.PHONY: dev
dev: ## Run the API with hot reload (air)
	air

.PHONY: tidy
tidy: ## Tidy go.mod / go.sum
	go mod tidy

.PHONY: test
test: ## Run unit tests with the race detector
	go test ./... -race -count=1

.PHONY: test-cover
test-cover: ## Run tests with a coverage report
	go test ./... -race -count=1 -coverprofile=coverage.out
	go tool cover -func=coverage.out

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run

.PHONY: sqlc
sqlc: ## Generate sqlc code from db/queries
	sqlc generate

.PHONY: swag
swag: ## Generate swagger docs
	swag init -g cmd/api/main.go -o docs/swagger --parseDependency --parseInternal

.PHONY: install-air
install-air: ## Install air (hot reload)
	go install github.com/air-verse/air@latest

.PHONY: install-tools
install-tools: install-air ## Install dev CLIs (air, migrate, sqlc)
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

.PHONY: migrate-up
migrate-up: ## Apply all up migrations
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" up

.PHONY: migrate-down
migrate-down: ## Roll back one migration
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" down 1

.PHONY: migrate-create
migrate-create: ## Create a migration: make migrate-create name=add_x
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $(name)

.PHONY: migrate-force
migrate-force: ## Force a migration version: make migrate-force version=1
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" force $(version)

.PHONY: seed
seed: ## Seed dummy data
	psql "$(DATABASE_URL)" -f db/seed/seed.sql

.PHONY: idempotency-prune
idempotency-prune: ## Delete idempotency keys older than the 24h replay window (run via cron/CronJob)
	psql "$(DATABASE_URL)" -c "DELETE FROM idempotency_keys WHERE created_at < now() - interval '24 hours';"

.PHONY: docker-up
docker-up: ## Start local Postgres
	docker compose up -d db

.PHONY: docker-down
docker-down: ## Stop the local stack
	docker compose down
