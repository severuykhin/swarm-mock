# Запуск и тестирование

- Локальный запуск: `go run ./cmd/server` из каталога [`main`](main:1).
- Переменные окружения: `SWARM_STUB_HTTP_ADDR` с дефолтом `127.0.0.1:11208`.
- Переменные окружения: `SWARM_STUB_HTTP_ADDR` с дефолтом `127.0.0.1:11208`,
  `SWARM_STUB_DATA_PATH` (путь к JSON-снапшоту),
  `SWARM_STUB_PERSIST_INTERVAL` (интервал фонового сброса, формат `time.ParseDuration`).
- Тестирование: `go test ./...` в каталоге [`main`](main:1).
