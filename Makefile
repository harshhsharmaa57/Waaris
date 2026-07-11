SHELL := /bin/sh
GO_PACKAGES := ./platform/... ./services/enrollment/... ./services/heartbeat/... ./services/notification/... ./services/witness-coordination/...
COMPOSE := docker compose

.PHONY: bootstrap up down logs ps build test lint format format-check migrate-up migrate-down check

bootstrap:
	@cp -n .env.example .env 2>/dev/null || true
	npm --prefix apps/web-dashboard install

up:
	$(COMPOSE) up --build -d

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f

ps:
	$(COMPOSE) ps

build:
	GOTELEMETRY=off go build $(GO_PACKAGES)
	npm --prefix apps/web-dashboard run build

test:
	GOTELEMETRY=off go test $(GO_PACKAGES)
	npm --prefix apps/web-dashboard test

lint:
	@test -z "$$(gofmt -l platform services)"
	GOTELEMETRY=off go vet $(GO_PACKAGES)
	npm --prefix apps/web-dashboard run lint

format:
	gofmt -w platform services
	npm --prefix apps/web-dashboard run format

format-check:
	@test -z "$$(gofmt -l platform services)"
	npm --prefix apps/web-dashboard run format:check

migrate-up:
	$(COMPOSE) run --rm migrations

migrate-down:
	@echo "Down migrations are intentionally not automated; run a reviewed, explicit migration command."
	@false

check: format-check lint test build
