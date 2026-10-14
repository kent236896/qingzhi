.PHONY: help infra-up infra-down infra-logs api-dev web-dev migrate-up migrate-down migrate-version check test test-api test-web compose-config

COMPOSE = docker compose --env-file .env -f deploy/docker-compose.yml

help:
	@echo "Windows: use scripts/*.ps1. See README.md."

infra-up:
	$(COMPOSE) up -d

infra-down:
	$(COMPOSE) down

infra-logs:
	$(COMPOSE) logs -f

api-dev:
	go run -C services/enterprise-api ./cmd/server

web-dev:
	pnpm --dir apps/admin-web dev

migrate-up:
	go run -C services/enterprise-api ./cmd/migrate up

migrate-down:
	go run -C services/enterprise-api ./cmd/migrate down

migrate-version:
	go run -C services/enterprise-api ./cmd/migrate version

check: test

test: test-api test-web

test-api:
	go test -C services/enterprise-api ./...
	go vet -C services/enterprise-api ./...

test-web:
	pnpm --dir apps/admin-web typecheck
	pnpm --dir apps/admin-web lint
	pnpm --dir apps/admin-web test
	pnpm --dir apps/admin-web build

compose-config:
	$(COMPOSE) --profile edge config
