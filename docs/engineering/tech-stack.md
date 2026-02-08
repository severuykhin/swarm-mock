# Технологии и зависимости

- Язык: Go 1.22 (см. [`go.mod`](go.mod:1)).
- HTTP-маршрутизация: [`github.com/go-chi/chi/v5`](go.mod:6) (middleware, маршруты).
- Логирование: [`go.uber.org/zap`](go.mod:7) (structured logging).
- Пакет [`swarm`](pkg/swarm): утилиты упаковки/распаковки tuple-структур (используется в [`service.Set()`](internal/service/service.go:153) и [`handler.packTuples()`](internal/handler/handler.go:155)).
