FROM golang:1.23.4

WORKDIR /app

# Сначала копируем go.mod и go.sum,
# чтобы Docker мог закешировать скачивание зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Затем копируем весь остальной код
COPY . .

# Если бот слушает порт 8080 — делаем EXPOSE 8080 (по необходимости)
EXPOSE 8080

# Запускаем бота
CMD ["go", "run", "cmd/main.go"]
