# Микросервис для сжатия изображений на Go

## Описание

Преобразует картинки в JPEG с заданным качеством и возвращает результат в формате base64.

- Качество по умолчанию задается в файле окружения, но можно передавать в запросе через поле quality
- Поддерживаемые форматы: JPEG, PNG, WebP, GIF
- Проверка MIME-типа по содержимому файла (защита от подделки расширений)
- Максимальный размер файла настраивается через переменные окружения

## API Endpoints

### 1. POST /compress

- Сжимает изображение и возвращает в base64.
- Параметры (multipart/form-data):

image (file, обязательный) - файл изображения
quality (int, опциональный) - качество JPEG от 1 до 100

- Пример запроса с curl:

```bash
curl -X POST http://localhost:7070/compress \
    -F "image=@/path/to/your/image.jpg" \
    -F "quality=75"
```

- Пример запроса с Postman:

```plain-text
POST http://localhost:7070/compress
Content-Type: multipart/form-data

Form data:
- image: [choose file]
- quality: 75
```

- Пример успешного ответа

```json
{
  "image": "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQAAAQABAAD...",
  "extension": "jpeg"
}
```

### 2. GET /health

- Проверка состояния сервиса.
- Пример запроса

```bash
curl http://localhost:7070/health
```

- Ответ

```json
{
  "status": "ok"
}
```

## Иницицализаця

1. Docker-контейнер

```bash
# Сборка образа
docker build -t image-processor .
# Запуск контейнера
docker run -p 7070:7070 --env-file .env image-processor
# Или через docker-compose
docker-compose up -d
```

2. Локально

```bash
# Установка зависимостей
go mod tidy

# Запуск в режиме разработки
go run main.go

# Сборка и запуск
go build -o image-processor ./image-processor
```

## Пример .env

```env
PORT=7070
HOST=0.0.0.0
MAX_FILE_SIZE=10485760
DEFAULT_QUALITY=80
ALLOW_ORIGIN=*
GIN_MODE=release
```
