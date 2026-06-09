.PHONY: all build run test lint fmt tidy docker-up docker-down migrate-up migrate-down sqlc seed swagger deploy setup

BINARY := bin/api
CMD     := ./cmd/api

all: build

## Build the binary
build:
	go build -ldflags="-w -s" -o $(BINARY) $(CMD)

## Run the API locally (requires .env)
run:
	set -a && . ./.env && set +a && go run $(CMD)

## Run all tests
test:
	go test -race -cover ./...

## Lint (requires golangci-lint)
lint:
	golangci-lint run ./...

## Format all Go files
fmt:
	gofmt -w .
	goimports -w .

## Tidy modules
tidy:
	go mod tidy

## Start local Docker services (Postgres + Redis)
docker-up:
	docker compose up -d postgres redis

## Stop and remove local Docker services
docker-down:
	docker compose down -v

## Run the full stack in Docker
stack-up:
	docker compose up --build -d

## Apply all migrations
migrate-up:
	set -a && . ./.env && set +a && $(shell go env GOPATH)/bin/migrate -path migrations -database "$$DATABASE_URL" up

## Roll back the last migration
migrate-down:
	set -a && . ./.env && set +a && $(shell go env GOPATH)/bin/migrate -path migrations -database "$$DATABASE_URL" down 1

## Generate type-safe DB code from SQL queries
sqlc:
	sqlc generate

## Seed the database with grant data
seed:
	set -a && . ./.env && set +a && go run ./scripts/seed/main.go

## Generate Swagger docs (requires: go install github.com/swaggo/swag/cmd/swag@v1.16.4)
swagger:
	$(shell go env GOPATH)/bin/swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal

## Create a new migration file (usage: make migration NAME=add_xyz)
migration:
	migrate create -ext sql -dir migrations -seq $(NAME)

## Deploy to production server (usage: make deploy KEY=~/.ssh/your-key.pem)
deploy:
	chmod +x deploy/deploy.sh
	./deploy/deploy.sh $(KEY)

## One-time server bootstrap (usage: make setup KEY=~/.ssh/your-key.pem)
setup:
	chmod +x deploy/setup.sh
	ssh -i $(KEY) ubuntu@13.52.254.25 'bash -s' < deploy/setup.sh
