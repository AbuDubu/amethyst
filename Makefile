# Run from the repository root. `make help` lists targets.
.DEFAULT_GOAL := help
.PHONY: help web-install web-build db-up db-down db-reset fed-up fed-down smoke migrate generate jobs dev dev-web test check build clean

COMPOSE := docker compose -f deploy/compose.yaml
# Matches deploy/compose.yaml; development only.
export AMETHYST_DATABASE_URL ?= postgres://amethyst:amethyst@localhost:5432/amethyst?sslmode=disable
export AMETHYST_CANONICAL_ORIGIN ?= http://localhost:8080
# Development mail goes to Mailpit (http://localhost:8025), never to real inboxes.
export AMETHYST_SMTP_HOST ?= localhost
export AMETHYST_SMTP_PORT ?= 1025
export AMETHYST_SMTP_TLS ?= none
export AMETHYST_MAIL_FROM ?= Amethyst <noreply@localhost>
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
# Admin connection used by tests to create a throwaway database per test.
export AMETHYST_TEST_DATABASE_URL ?= postgres://amethyst:amethyst@localhost:5432/postgres?sslmode=disable
# Mailpit for email integration tests (SMTP on 1025, API on 8025).
export AMETHYST_TEST_MAILPIT_HOST ?= localhost

help: ## List targets
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-12s %s\n", $$1, $$2}'

web/node_modules: web/package-lock.json
	cd web && npm ci
	@touch web/node_modules

web-install: web/node_modules ## Install frontend dependencies

web-build: web/node_modules ## Build the frontend into web/dist
	cd web && npm run build

db-up: ## Start PostgreSQL and Mailpit (waits until healthy)
	$(COMPOSE) up -d --wait

db-down: ## Stop local dependencies (data is kept)
	$(COMPOSE) down

db-reset: ## Stop local dependencies and delete their data
	$(COMPOSE) --profile federation down --volumes

fed-up: ## Build the image and run two servers: http://a.localhost:8081 and http://b.localhost:8082
	AMETHYST_VERSION=$(VERSION) $(COMPOSE) --profile federation up -d --build

fed-down: ## Stop the two servers (dependencies keep running)
	$(COMPOSE) --profile federation stop server-a server-b

smoke: ## Run the two-server smoke test
	AMETHYST_VERSION=$(VERSION) deploy/smoke-federation.sh

generate: web/node_modules ## Regenerate code from SQL queries and the OpenAPI contract
	cd server && go tool sqlc generate
	cd server && go tool oapi-codegen -config internal/api/apigen/oapi-codegen.yaml ../api/openapi.yaml
	cd web && npm run generate

migrate: db-up ## Apply pending database migrations
	cd server && go run ./cmd/amethyst migrate

jobs: ## Show background job queue status
	cd server && go run ./cmd/amethyst jobs

dev: web-build migrate ## Run the Go server on :8080, serving the built frontend
	cd server && AMETHYST_WEB_DIR=../web/dist go run ./cmd/amethyst serve

dev-web: web/node_modules ## Run the Vite dev server on :5173 (hot reload; proxies /api to :8080)
	cd web && npm run dev

test: web/node_modules db-up ## Run Go and frontend tests
	cd server && go test -race ./...
	cd web && npm test

check: web/node_modules db-up ## Run every CI check locally
	@cd server && test -z "$$(gofmt -l .)" || (echo "gofmt needed:"; gofmt -l .; exit 1)
	cd server && go vet ./...
	cd server && go tool staticcheck ./...
	cd server && go test -race ./...
	cd web && npm run typecheck
	cd web && npm run lint
	cd web && npm test
	cd web && npm run build

build: web-build ## Build the frontend and the server binary (server/bin/amethyst)
	cd server && go build -ldflags "-X main.version=$(VERSION)" -o bin/amethyst ./cmd/amethyst

clean: ## Remove build output
	rm -rf web/dist server/bin
