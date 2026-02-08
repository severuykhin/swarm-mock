package http

import (
	stdhttp "net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	handlerpkg "main/internal/handler"
	"main/internal/service"
)

// NewRouter собирает HTTP-маршруты.
func NewRouter(log *zap.Logger, svc service.KVService) stdhttp.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(Logger(log))

	h := handlerpkg.New(log, svc)

	r.Route("/swarm", func(r chi.Router) {
		r.Get("/{k1Prefix}/{k1Suffix}/", h.GetK1)
		r.Get("/{k1Prefix}/{k1Suffix}/{k2}/", h.Get)
		r.Get("/{k1Prefix}/{k1Suffix}/{k2}/{k3}/", h.Get)
		r.Put("/{k1Prefix}/{k1Suffix}/{k2}/", h.Set)
		r.Put("/{k1Prefix}/{k1Suffix}/{k2}/{k3}/", h.Set)
		r.Delete("/{k1Prefix}/{k1Suffix}/{k2}/", h.Delete)
		r.Delete("/{k1Prefix}/{k1Suffix}/{k2}/{k3}/", h.Delete)
	})

	return r
}
