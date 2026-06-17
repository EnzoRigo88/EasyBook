# ── Stage 1: Build ────────────────────────────────────────────────────────────
# Compilamos el binario en un contenedor con el toolchain de Go
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Instalar dependencias del OS (necesario para CGO con algunos drivers)
RUN apk add --no-cache git ca-certificates tzdata

# Copiar módulos primero (para aprovechar el cache de Docker)
# Si go.mod/go.sum no cambian, esta capa no se recompila
COPY go.mod go.sum ./
RUN go mod download

# Copiar el resto del código
COPY . .

# Compilar el binario
# CGO_ENABLED=0 → binario completamente estático (sin dependencias del OS)
# -ldflags="-w -s" → quitar debug symbols (binario más pequeño)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o /app/server ./cmd/server

# ── Stage 2: Runtime ──────────────────────────────────────────────────────────
# Imagen final extremadamente pequeña — solo el binario + certificados SSL
# scratch = imagen vacía de Docker (0MB base)
FROM scratch

# Copiar certificados CA para HTTPS calls (OpenAI API, Meta, etc.)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# Copiar datos de timezone para parsear fechas correctamente (America/Argentina/Buenos_Aires)
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
# Copiar el binario compilado
COPY --from=builder /app/server /server

# El puerto que expone la aplicación
EXPOSE 3000

# Correr como non-root (uid 65534 = nobody)
USER 65534:65534

ENTRYPOINT ["/server"]
