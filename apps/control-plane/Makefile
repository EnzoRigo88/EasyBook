# Makefile para EasyBook control-plane (Go)
# Uso: make <comando>

.PHONY: dev build test lint sqlc docker-build docker-run clean

# ── Development ───────────────────────────────────────────────────────────────

## dev: Arranca el servidor con hot-reload (usando air)
dev:
	@which air > /dev/null || go install github.com/air-verse/air@latest
	air -c .air.toml

## build: Compila el binario
build:
	go build -o bin/server ./cmd/server

## run: Compila y corre (sin hot-reload)
run: build
	./bin/server

# ── Testing ───────────────────────────────────────────────────────────────────

## test: Corre todos los tests
test:
	go test -v -race ./...

## test-cover: Corre tests con coverage
test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# ── Code quality ──────────────────────────────────────────────────────────────

## lint: Corre golangci-lint
lint:
	@which golangci-lint > /dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run

## fmt: Formatea el código
fmt:
	gofmt -w .
	goimports -w .

## vet: Análisis estático básico (viene con Go)
vet:
	go vet ./...

# ── Database ──────────────────────────────────────────────────────────────────

## sqlc: Genera código Go desde los archivos SQL
sqlc:
	@which sqlc > /dev/null || go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	sqlc generate

## db-migrate: Corre las migraciones (usando goose o prisma desde el root)
db-migrate:
	cd ../../ && pnpm db:migrate

# ── Docker ────────────────────────────────────────────────────────────────────

## docker-build: Construye la imagen Docker de producción
docker-build:
	docker build -t easybook-control-plane:latest .

## docker-run: Corre el contenedor localmente
docker-run: docker-build
	docker run --rm -p 3000:3000 \
		--env-file ../../.env \
		easybook-control-plane:latest

# ── Utilities ─────────────────────────────────────────────────────────────────

## clean: Limpia binarios y cache
clean:
	rm -rf bin/ coverage.out tmp/

## deps: Descarga e instala dependencias
deps:
	go mod download
	go mod tidy

## help: Muestra este mensaje de ayuda
help:
	@grep -E '^## ' Makefile | sed 's/## //'
