package main

import (
	"context"
	"errors"
	stdhttp "net/http"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/cxhub/swarm-mock/internal/config"
	transport "github.com/cxhub/swarm-mock/internal/http"
	"github.com/cxhub/swarm-mock/internal/logger"
	"github.com/cxhub/swarm-mock/internal/service"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	l, err := logger.New()
	if err != nil {
		panic(err)
	}
	defer logger.Close(l)

	cfg, err := config.Load()
	if err != nil {
		l.Fatal("failed to load config", zap.Error(err))
	}

	persistence := service.NewJSONPersistence(cfg.Persistence.Path)
	svc, err := service.NewStub(
		service.WithPersistence(persistence),
		service.WithLoadContext(ctx),
		service.WithAutosaveInterval(cfg.Persistence.Interval),
	)
	if err != nil {
		l.Fatal("failed to initialize service", zap.Error(err))
	}
	svc.StartAutosave(ctx, func(err error) {
		l.Warn("autosave failed", zap.Error(err))
	})

	h := transport.NewRouter(l, svc)

	server := &stdhttp.Server{
		Addr:         cfg.HTTP.Addr,
		Handler:      h,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		l.Info("http server starting", zap.String("addr", cfg.HTTP.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, stdhttp.ErrServerClosed) {
			l.Fatal("http server failed", zap.Error(err))
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := server.Shutdown(shutdownCtx); err != nil {
		l.Error("http server shutdown error", zap.Error(err))
	}

	if err := svc.SaveSnapshot(shutdownCtx); err != nil {
		l.Error("failed to persist state", zap.Error(err))
	}

	svc.WaitAutosave()

	l.Info("http server stopped")
}
