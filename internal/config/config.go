package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

const (
	// DefaultHTTPAddr определяет адрес HTTP-сервера по умолчанию.
	DefaultHTTPAddr = "127.0.0.1:11208"

	envHTTPAddr            = "SWARM_STUB_HTTP_ADDR"
	envPersistencePath     = "SWARM_STUB_DATA_PATH"
	envPersistenceInterval = "SWARM_STUB_PERSIST_INTERVAL"
)

// HTTPConfig содержит настройки HTTP-сервера.
type HTTPConfig struct {
	Addr string
}

// Config агрегирует настройки приложения.
type Config struct {
	HTTP        HTTPConfig
	Persistence PersistenceConfig
}

// PersistenceConfig описывает параметры сохранения состояния.
type PersistenceConfig struct {
	Path     string
	Interval time.Duration
}

// Load загружает конфигурацию из переменных окружения с дефолтами.
func Load() (Config, error) {
	addr := os.Getenv(envHTTPAddr)
	if addr == "" {
		addr = DefaultHTTPAddr
	}

	if err := validateAddr(addr); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}

	persistencePath := os.Getenv(envPersistencePath)
	if persistencePath == "" {
		persistencePath = filepath.Join("data", "swarm_stub_snapshot.json")
	}

	interval := time.Minute
	if rawInterval := os.Getenv(envPersistenceInterval); rawInterval != "" {
		parsed, err := time.ParseDuration(rawInterval)
		if err != nil {
			return Config{}, fmt.Errorf("config: invalid persistence interval: %w", err)
		}
		if parsed < 0 {
			return Config{}, fmt.Errorf("config: persistence interval must be non-negative")
		}
		interval = parsed
	}

	return Config{
		HTTP: HTTPConfig{Addr: addr},
		Persistence: PersistenceConfig{
			Path:     persistencePath,
			Interval: interval,
		},
	}, nil
}

func validateAddr(addr string) error {
	if addr == "" {
		return fmt.Errorf("http addr is empty")
	}

	if _, err := net.ResolveTCPAddr("tcp", addr); err != nil {
		return fmt.Errorf("invalid http addr %q: %w", addr, err)
	}

	return nil
}
