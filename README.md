# URL Shortener

HTTP-сервис сокращает длинные HTTP/HTTPS-адреса в идентификаторы длиной 6–8 символов и перенаправляет по ним обратно. Данные хранятся в памяти процесса.

## Запуск

```bash
go run .
```

Сервер слушает `:8080`.

## Примеры

Сократить адрес:

```bash
curl -s -X POST http://localhost:8080/shorten \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://example.com/very/long/path"}'
```

Ответ:

```json
{"short_url":"abc123xx","original_url":"https://example.com/very/long/path"}
```

Перейти по короткому идентификатору (ответ `302` и заголовок `Location`):

```bash
curl -s -D - -o /dev/null http://localhost:8080/abc123xx
```

Неизвестный идентификатор возвращает `404`. Некорректный JSON или URL на `POST /shorten` возвращает `400` и JSON `{"error":"..."}`.

## Тесты и покрытие

```bash
go test -v ./...
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

HTML-отчёт:

```bash
go tool cover -html=coverage.out
```
