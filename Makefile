.PHONY: verify lint test up down logs load-order-fill lan-https lan-https-down https https-down

verify:
	bash scripts/verify.sh

lint:
	bash scripts/verify-toolchain.sh
	npm run lint --prefix frontend
	bash scripts/verify-go.sh services/api-service lint
	bash scripts/verify-go.sh services/document-service lint

test:
	npm run test --prefix frontend
	cd services/api-service && go test ./...
	cd services/document-service && go test ./...

COMPOSE := docker compose $(if $(wildcard .env),--env-file .env) -f deploy/docker-compose.yml

up:
	$(COMPOSE) up --build

down:
	$(COMPOSE) down --remove-orphans

logs:
	$(COMPOSE) logs -f

load-order-fill:
	node scripts/load-order-fill.mjs $(ARGS)

lan-https:
	bash scripts/lan-https.sh

lan-https-down:
	$(COMPOSE) $(if $(wildcard deploy/Caddyfile.lan),-f deploy/docker-compose.lan-https.yml) down --remove-orphans

https:
	bash scripts/prod-https.sh

https-down:
	PUBLIC_HOST=$${PUBLIC_HOST:-_} $(COMPOSE) -f deploy/docker-compose.https.yml down --remove-orphans
