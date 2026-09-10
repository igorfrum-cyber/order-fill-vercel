.PHONY: verify lint test up down logs load-order-fill lan-https lan-https-down https https-down

verify:
	bash scripts/verify.sh

lint:
	bash scripts/verify-toolchain.sh
	npm run lint --prefix frontend
	@find backend -name go.mod -exec dirname {} \; | sort | while IFS= read -r module; do \
		echo "==> $$module"; \
		bash scripts/verify-go.sh "$$module" lint; \
	done

test:
	npm run test:load
	npm run test --prefix frontend
	@root=$$(pwd); \
	find backend -name go.mod -exec dirname {} \; | sort | while IFS= read -r module; do \
		echo "==> $$module"; \
		(cd "$$module" && \
			GOWORK=off \
			GOTOOLCHAIN="$${GOTOOLCHAIN:-auto}" \
			GOCACHE="$${GOCACHE:-$$root/.cache/go-build}" \
			GOMODCACHE="$${GOMODCACHE:-$$root/.cache/go-mod}" \
			go test ./...); \
	done

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
