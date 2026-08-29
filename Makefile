.PHONY: verify lint test up down logs

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

up:
	docker compose -f deploy/docker-compose.yml up --build

down:
	docker compose -f deploy/docker-compose.yml down

logs:
	docker compose -f deploy/docker-compose.yml logs -f
