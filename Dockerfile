# ---------- BUILDER ----------
FROM golang:latest AS builder

WORKDIR /build
# Копируем go.mod + go.sum и качаем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь проект
COPY . .

# Собираем бинарь
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o db_sync ./cmd/db_sync

# ---------- RUNTIME ----------
FROM alpine:latest

RUN adduser -D emaildb
USER emaildb

WORKDIR /app

# Копируем бинарь из builder
COPY --from=builder /build/db_sync .

CMD ["./db_sync"]

