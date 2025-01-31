FROM golang:1.23.3-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o expense-bot

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/expense-bot .
COPY templates/ ./templates/

CMD ["./expense-bot"]
