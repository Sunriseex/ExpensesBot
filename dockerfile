# Stage 1: сборка приложения
FROM golang:1.23.3-alpine AS builder

WORKDIR /app

# Копируем файлы модулей и скачиваем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код (включая директорию templates, если она у вас есть)
COPY . .

# Собираем приложение; CGO отключён для получения статического бинарника
RUN CGO_ENABLED=0 GOOS=linux go build -o expensesbot .

# Stage 2: минимальный образ для запуска
FROM alpine:latest

WORKDIR /root/

# Копируем собранный бинарник
COPY --from=builder /app/expensesbot .

# Копируем директорию шаблонов, если используется (например, для веб-интерфейса)
COPY --from=builder /app/templates ./templates

# (Опционально) копируем файл .env для локальной отладки; для продакшена переменные задаются через docker-compose
COPY --from=builder /app/.env .

EXPOSE 8080

CMD ["./expensesbot"]
