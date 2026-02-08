package main

import (
	"context"
	"errors"
	stdhttp "net/http"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"scripts/swarm_stub/internal/config"
	transport "scripts/swarm_stub/internal/http"
	"scripts/swarm_stub/internal/logger"
	"scripts/swarm_stub/internal/service"
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

	svc := service.NewStub()
	h := transport.NewRouter(l, svc)

	server := &stdhttp.Server{
		Addr:         "127.0.0.1:11208",
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

	l.Info("http server stopped")
}
