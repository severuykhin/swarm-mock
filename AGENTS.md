# Основные правила работы с проектом
- Используй инструмент `bd` как таск-трекер для работы с задачами

# Конституция проекта Swarm Stub HTTP Service

## Назначение
- Репозиторий содержит заглушку HTTP API, имитирующую сервис Swarm KV для локальной разработки и интеграционного тестирования.
- Цель — воспроизвести ключевые маршруты и контракты клиента Swarm согласно спецификации [`spec.md`](docs/swarm-spec/spec.md:1), обеспечивая стабильные предсказуемые ответы.

## Контекст и границы
- Фаза разработки №1: возвращаются статические ответы и in-memory состояние без интеграции с реальным Swarm.
- Фаза №2 планирует замену заглушек на реальную бизнес-логику, указанную в TODO-комментариях исходного кода.
- Сервис предназначен для запуска локально и внутри автоматизированных пайплайнов как вспомогательный компонент.

## Архитектура и компоненты
- Входная точка [`main`](server/main.go:1) конфигурирует логгер, загружает конфигурацию, инициализирует сервис и HTTP-сервер с graceful shutdown.
- Конфигурация читается из окружения с валидацией адреса в [`config.Load()`](internal/config/config.go:26).
- Логирование централизовано через zap-фабрику [`logger.New()`](internal/logger/logger.go:10) с безопасным закрытием ресурсов.
- HTTP-транспорт собирается роутером [`http.NewRouter()`](internal/http/router.go:14), который подключает middleware `RequestID`, `RealIP` и кастомный логгер.
- Обработчики реализованы в [`handler.Handler`](internal/handler/handler.go:16); они нормализуют ключи Swarm, сериализуют/десериализуют полезную нагрузку и мапят ошибки сервисного слоя в HTTP-статусы.
- Бизнес-логика инкапсулирована в in-memory реализацию [`service.Stub`](internal/service/service.go:69), обеспечивающую потокобезопасный доступ к KV-данным и общую обработку ошибок.

## Технологии и зависимости
- Язык: Go 1.22 (см. [`go.mod`](go.mod:1)).
- HTTP-маршрутизация: `github.com/go-chi/chi/v5` (middleware, маршруты).
- Логирование: `go.uber.org/zap` (structured logging).
- Пакет [`swarm`](pkg/swarm): утилиты упаковки/распаковки tuple-структур (используется в [`service.Set()`](internal/service/service.go:153) и [`handler.packTuples()`](internal/handler/handler.go:155)).

## Принципы разработки
- Чистое разделение транспортного, обработчиков и бизнес-уровня для упрощения замены заглушек на реальную реализацию.
- Явное управление контекстом: все публичные методы проверки `ctx.Err()` с маппингом на корректные HTTP-статусы [`service.contextError()`](internal/service/service.go:233).
- Потокобезопасность: общий доступ к in-memory данным защищён `sync.RWMutex` в [`service.Stub`](internal/service/service.go:69).
- Строгая валидация ввода и ошибкоориентированное проектирование: ошибки оборачиваются в статусные типы [`service.statusError`](internal/service/service.go:47) для детерминированного ответа.
- Логирование с уровнями: WARN для ошибок клиента, ERROR для серверных с включением контекста.

## Практики качества
- Покрытие unit-тестами транспорта и сервисного слоя ([`router_test.go`](internal/http/router_test.go:1), [`service_test.go`](internal/service/service_test.go:1)).
- TODO-комментарии фиксируют будущие расширения — при реализации реальной логики из стадий 2 они служат ориентиром.
- Статический линтинг и стиль следуют стандартам Go (go fmt, go vet) — запускать перед коммитом.

## Запуск и тестирование
- Локальный запуск: `go run ./cmd/server` из каталога [`main`](main:1).
- Переменные окружения: `SWARM_STUB_HTTP_ADDR` с дефолтом `127.0.0.1:11208`.
- Тестирование: `go test ./...` в каталоге [`main`](main:1).

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd sync
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
