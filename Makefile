.PHONY: run worker build build-worker test tidy up down logs migrate-up migrate-down lint

MIGRATE_DSN ?= postgres://ats:ats@localhost:5432/ats?sslmode=disable

run: ## Run the API server
	go run ./cmd/api

worker: ## Run the background worker
	go run ./cmd/worker

build: ## Build the API binary into ./bin
	go build -o bin/api ./cmd/api

build-worker: ## Build the worker binary into ./bin
	go build -o bin/worker ./cmd/worker

test: ## Run tests with race detector and coverage
	go test ./... -race -cover

tidy: ## Sync go.mod / go.sum
	go mod tidy

up: ## Start Postgres + Redis
	docker compose up -d

down: ## Stop Postgres + Redis
	docker compose down

logs: ## Tail infra logs
	docker compose logs -f

migrate-up: ## Apply all pending migrations
	migrate -path migrations -database "$(MIGRATE_DSN)" up

migrate-down: ## Roll back the last migration
	migrate -path migrations -database "$(MIGRATE_DSN)" down 1

lint: ## Run golangci-lint
	golangci-lint run
