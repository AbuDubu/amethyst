# Run from the repository root. `make help` lists targets.
.DEFAULT_GOAL := help
.PHONY: help web-install web-build db-up db-down db-reset migrate dev dev-web test check build clean

COMPOSE := docker compose -f deploy/compose.yaml
# Matches deploy/compose.yaml; development only.
export AMETHYST_DATABASE_URL ?= postgres://amethyst:amethyst@localhost:5432/amethyst?sslmode=disable
# Admin connection used by tests to create a throwaway database per test.
export AMETHYST_TEST_DATABASE_URL ?= postgres://amethyst:amethyst@localhost:5432/postgres?sslmode=disable

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
	$(COMPOSE) down --volumes

migrate: db-up ## Apply pending database migrations
	cd server && go run ./cmd/amethyst migrate

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
	cd server && go build -o bin/amethyst ./cmd/amethyst

clean: ## Remove build output
	rm -rf web/dist server/bin
