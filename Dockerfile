# Этап 1: Сборка приложения на Go
FROM golang:1.21-alpine AS builder

# Устанавливаем рабочую директорию внутри контейнера
WORKDIR /app

# Копируем файлы go.mod и go.sum
COPY go.mod go.sum ./

# Загружаем все зависимости. Зависимости будут кэшироваться, если файлы go.mod и go.sum не изменились
RUN go mod download

# Копируем исходный код из текущего каталога в рабочую директорию внутри контейнера
COPY . .

# Собираем приложение на Go
RUN go build -o note_value_api ./cmd/main.go

EXPOSE 3001

# Запускаем приложение на Go
CMD ["./note_value_api"]