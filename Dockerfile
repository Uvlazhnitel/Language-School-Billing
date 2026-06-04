FROM node:22-bookworm-slim AS frontend-builder

WORKDIR /app/frontend

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

FROM golang:1.24-bookworm AS backend-builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/langschool-web ./cmd/web
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/langschool-backupctl ./cmd/backupctl

FROM debian:bookworm-slim AS runtime

WORKDIR /app

RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates \
  && rm -rf /var/lib/apt/lists/*

COPY --from=backend-builder /out/langschool-web /usr/local/bin/langschool-web
COPY --from=backend-builder /out/langschool-backupctl /usr/local/bin/langschool-backupctl
COPY --from=backend-builder /app/frontend/dist ./frontend/dist
COPY --from=backend-builder /app/fonts ./fonts

EXPOSE 8080

CMD ["langschool-web"]
