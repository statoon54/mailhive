.PHONY: build build-frontend run run-api run-worker test test-unit test-integration test-smtp test-all lint migrate-up migrate-down docker-up docker-down docker-dev docker-dev-down docker-dev-s3 docker-prod docker-prod-down docker-prod-logs docker-test-up docker-test-down tidy

# Variables
BINARY=bin/mailhive
# Version dérivée de git (tag le plus proche, sinon SHA court) ; surchargeable :
#   make build VERSION=v1.2.3
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS=-ldflags="-s -w -X main.version=$(VERSION)"

# Construction
build-frontend:
	cd frontend && npm run build
	rm -rf internal/frontend/dist
	cp -r frontend/dist internal/frontend/dist

build: build-frontend
	go build $(LDFLAGS) -o $(BINARY) ./cmd/mailhive

build-go:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/mailhive

# Exécution
run:
	go run ./cmd/mailhive

run-api:
	go run ./cmd/mailhive api

run-worker:
	go run ./cmd/mailhive worker

# Tests
test:
	go test ./... -v -count=1

test-unit:
	go test ./internal/domain/... ./internal/service/... ./internal/handler/... \
		./internal/worker/... ./internal/middleware/... ./internal/analysis/... -v -count=1

test-integration:
	go test ./internal/test/... -v -count=1 -timeout=120s

test-smtp:
	go test -tags=integration ./internal/adapter/mailer/... -v -count=1 -timeout=60s

test-all:
	go test ./... -v -count=1 -timeout=120s

# Lint
lint:
	golangci-lint run ./...

# Migrations
migrate-up:
	go run ./cmd/mailhive migrate

migrate-down:
	go run ./cmd/mailhive migrate-down

# Dépendances
tidy:
	go mod tidy

# Docker
docker-up:
	VERSION=$(VERSION) docker compose up --build -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

# Docker dev (Mailpit)
docker-dev:
	VERSION=$(VERSION) SMTP_MODE=mailpit docker compose --profile dev up --build -d

docker-dev-down:
	docker compose --profile dev down

docker-dev-logs:
	docker compose --profile dev logs -f

# Docker dev avec stockage des pièces jointes sur S3 (SeaweedFS local).
# La surcharge docker-compose.s3.yml active BLOB_BACKEND=s3 et fait attendre
# mailhive que la passerelle S3 soit saine (depends_on: service_healthy).
docker-dev-s3:
	VERSION=$(VERSION) SMTP_MODE=mailpit docker compose -f docker-compose.yml -f docker-compose.s3.yml --profile dev up --build -d

# Docker production : exécute l'image publiée (GHCR) avec pièces jointes sur S3.
# Nécessite les secrets en environnement (JWT_SECRET, ENCRYPTION_KEY, etc.) —
# voir docker-compose.prod.yml. Choix du tag : MAILHIVE_TAG=1.2.3 (semver sans « v »).
docker-prod:
	docker compose -f docker-compose.prod.yml up -d

docker-prod-down:
	docker compose -f docker-compose.prod.yml down

docker-prod-logs:
	docker compose -f docker-compose.prod.yml logs -f

# Docker test (Mailpit)
docker-test-up:
	docker compose -f docker-compose.test.yml up -d

docker-test-down:
	docker compose -f docker-compose.test.yml down
