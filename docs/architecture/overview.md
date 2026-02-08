# Архитектура и компоненты

- Входная точка [`main`](server/main.go:1) конфигурирует логгер, загружает конфигурацию, инициализирует сервис и HTTP-сервер с graceful shutdown.
- Конфигурация читается из окружения с валидацией адреса в [`config.Load()`](internal/config/config.go:26).
- Логирование централизовано через zap-фабрику [`logger.New()`](internal/logger/logger.go:10) с безопасным закрытием ресурсов.
- HTTP-транспорт собирается роутером [`http.NewRouter()`](internal/http/router.go:14), который подключает middleware `RequestID`, `RealIP` и кастомный логгер.
- Обработчики реализованы в [`handler.Handler`](internal/handler/handler.go:16); они нормализуют ключи Swarm, сериализуют/десериализуют полезную нагрузку и мапят ошибки сервисного слоя в HTTP-статусы.
- Бизнес-логика инкапсулирована в in-memory реализацию [`service.Stub`](internal/service/service.go:69), обеспечивающую потокобезопасный доступ к KV-данным и общую обработку ошибок.
- Персистентность реализована адаптером [`service.JSONPersistence`](internal/service/persistence.go:18): он сериализует состояния `Stub` в JSON-снапшот с атомарной заменой файла и вызывается по расписанию воркером, настроенным через [`config.PersistenceConfig`](internal/config/config.go:31).
