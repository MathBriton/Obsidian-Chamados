# syntax=docker/dockerfile:1

# ============================================================
# Stage 1 — build
# Binário estático Go. Sem CGO graças ao driver SQLite pure-Go
# (modernc.org/sqlite), o que permite uma imagem final mínima.
# ============================================================
FROM golang:1.26-alpine AS build

WORKDIR /src

# Cache de dependências
COPY go.mod go.sum ./
RUN go mod download

# Código-fonte
COPY . .

# Build estático do servidor
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /out/server ./cmd/server

# ============================================================
# Stage 2 — runtime
# Imagem distroless mínima, non-root, apenas com o binário.
# ============================================================
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

# Binário e migrations (aplicadas no boot)
COPY --from=build /out/server /app/server
COPY --from=build /src/migrations /app/migrations

EXPOSE 8080
USER nonroot:nonroot

ENTRYPOINT ["/app/server"]
