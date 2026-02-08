package logger

import (
	"context"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger описывает минимальный интерфейс логгера, используемый приложением.
type Logger interface {
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Sync() error
}

// New создает zap.Logger с настройками разработки по умолчанию.
func New() (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	cfg.OutputPaths = []string{"stdout"}
	cfg.ErrorOutputPaths = []string{"stderr"}
	cfg.DisableStacktrace = true

	return cfg.Build()
}

// WithContext добавляет в контекст Logger для последующего извлечения.
func WithContext(ctx context.Context, l *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, l)
}

// FromContext извлекает логгер из контекста, возвращая fallback при отсутствии.
func FromContext(ctx context.Context) *zap.Logger {
	if v := ctx.Value(loggerKey{}); v != nil {
		if l, ok := v.(*zap.Logger); ok {
			return l
		}
	}

	l, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	return l
}

type loggerKey struct{}

// Close безопасно закрывает логгер, подавляя ошибку Sync() на stderr/stdout.
func Close(l *zap.Logger) {
	if l == nil {
		return
	}

	if err := l.Sync(); err != nil && err != os.ErrClosed {
		// TODO(phase2): рассмотреть централизованную обработку ошибок Sync.
	}
}
