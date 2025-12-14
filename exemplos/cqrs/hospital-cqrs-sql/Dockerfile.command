# Dockerfile for Command Service
FROM golang:1.25-alpine AS base

WORKDIR /app

# Instalar Air para live reload
RUN go install github.com/air-verse/air@latest

# Copiar go mod files
COPY go.mod go.sum ./
RUN go mod download

# Development stage
FROM base AS development
WORKDIR /app
CMD ["air", "-c", ".air.command.toml"]
