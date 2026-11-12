.PHONY: run db-up db-down sqlc test build

run:
	CGO_ENABLED=0 go run ./cmd/api

db-up:
	docker compose up -d

db-down:
	docker compose down

sqlc:
	CGO_ENABLED=0 go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.27.0 generate

test:
	CGO_ENABLED=0 go test -v ./...

build:
	CGO_ENABLED=0 go build -o bin/api ./cmd/api
