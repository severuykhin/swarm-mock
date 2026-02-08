# Swarm Stub HTTP Service

Фаза 1: заглушка HTTP API, воспроизводящая маршруты клиента Swarm из [`docs/swarm-spec/spec.md`](../../docs/swarm-spec/spec.md:1).

## Быстрый старт

```bash
cd main
go run ./cmd/server
```

## Конфигурация

| Переменная | Назначение | Значение по умолчанию |
|------------|------------|------------------------|
| `SWARM_STUB_HTTP_ADDR` | Адрес HTTP сервера | `127.0.0.1:8080` |

## Структура

- [`cmd/server`](cmd/server/main.go:1) — точка входа, настройка логгера, конфигурации и graceful shutdown.
- [`internal/config`](internal/config/config.go:1) — загрузка конфигурации из окружения.
- [`internal/logger`](internal/logger/logger.go:1) — фабрика zap-логгера.
- [`internal/service`](internal/service/service.go:1) — интерфейсы и заглушка бизнес-логики.
- [`internal/http`](internal/http/router.go:1) — транспортный слой: роутер, middleware, обработчики.
- [`internal/handler`](internal/handler/handler.go:1) — конкретные HTTP-обработчики.

## Маршруты

Все маршруты отвечают кодом `200 OK` и статическими данными:

- `GET /swarm/{k1Prefix}/{k1Suffix}/{k2}/`
- `GET /swarm/{k1Prefix}/{k1Suffix}/{k2}/{k3}/`
- `GET /swarm/{k1Prefix}/{k1Suffix}/?t=...`
- `PUT /swarm/{k1Prefix}/{k1Suffix}/{k2}/`
- `PUT /swarm/{k1Prefix}/{k1Suffix}/{k2}/{k3}/`
- `DELETE /swarm/{k1Prefix}/{k1Suffix}/{k2}/`
- `DELETE /swarm/{k1Prefix}/{k1Suffix}/{k2}/{k3}/`

TODO-комментарии в коде отмечают места, где будет реализована реальная логика во второй фазе.

## Тесты

```bash
cd main
go test ./...
```
