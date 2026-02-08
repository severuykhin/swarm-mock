package config

import (
	"fmt"
	"net"
	"os"
)

const (
	// DefaultHTTPAddr определяет адрес HTTP-сервера по умолчанию.
	DefaultHTTPAddr = "127.0.0.1:11208"

	envHTTPAddr = "SWARM_STUB_HTTP_ADDR"
)

// HTTPConfig содержит настройки HTTP-сервера.
type HTTPConfig struct {
	Addr string
}

// Config агрегирует настройки приложения.
type Config struct {
	HTTP HTTPConfig
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

	return Config{
		HTTP: HTTPConfig{Addr: addr},
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
