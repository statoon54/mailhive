# =============================================================================
# MailHive — Dockerfile multi-stage unique
# Produit une image Alpine légère avec le binaire Go embarquant le frontend React.
# =============================================================================

# ---------- Stage 1 : Build frontend ----------
FROM node:24-alpine AS frontend-builder

WORKDIR /app/frontend

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci

COPY frontend/ .
RUN npm run build

# ---------- Stage 2 : Build Go ----------
FROM golang:1.26-alpine AS go-builder

WORKDIR /app

# Dépendances Go
COPY go.mod go.sum ./
RUN go mod download

# Code source
COPY . .

# Injecter le build frontend dans le package embed
COPY --from=frontend-builder /app/frontend/dist ./internal/frontend/dist

# Version injectée au build (passée via --build-arg VERSION=... ; "dev" sinon).
ARG VERSION=dev

# Compiler le binaire unique
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X main.version=${VERSION}" -o /bin/mailhive ./cmd/mailhive

# ---------- Stage 3 : Image finale ----------
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=go-builder /bin/mailhive /app/mailhive

EXPOSE 8080

ENTRYPOINT ["/app/mailhive"]
CMD ["serve"]
