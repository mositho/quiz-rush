.PHONY: setup setup-hooks dev dev-bg dev-down dev-logs dev-restart

COMPOSE_DEV = docker compose -f docker-compose.yml -f docker-compose.dev.yml

dev:
	$(COMPOSE_DEV) up --build

dev-bg:
	$(COMPOSE_DEV) up -d --build

dev-down:
	$(COMPOSE_DEV) down

dev-logs:
	$(COMPOSE_DEV) logs -f

dev-restart:
	make dev-down
	make dev-bg

setup: setup-hooks

setup-hooks:
	git config core.hooksPath .githooks
