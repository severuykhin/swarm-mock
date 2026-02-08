package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/cxhub/swarm-mock/internal/service"
	"github.com/cxhub/swarm-mock/pkg/swarm"
)

// Handler реализует HTTP-обработчики Swarm.
type Handler struct {
	log *zap.Logger
	svc service.KVService
}

// New создает Handler.
func New(log *zap.Logger, svc service.KVService) *Handler {
	return &Handler{log: log, svc: svc}
}

// Get обрабатывает GET /swarm/{k1Prefix}/{k1Suffix}/{k2}/[{k3}/].
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	key := service.Key{
		K1: composeK1(chi.URLParam(r, "k1Prefix"), chi.URLParam(r, "k1Suffix")),
		K2: chi.URLParam(r, "k2"),
		K3: chi.URLParam(r, "k3"),
	}

	result, err := h.svc.Get(ctx, key)
	if err != nil {
		h.handleError(w, "get failed", err)
		return
	}

	writeBinary(w, http.StatusOK, result.Payload)
}

// GetK1 обрабатывает GET /swarm/{k1Prefix}/{k1Suffix}/?t=...
func (h *Handler) GetK1(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	queries := r.URL.Query()["t"]

	key := service.K1{
		Prefix: chi.URLParam(r, "k1Prefix"),
		Suffix: chi.URLParam(r, "k1Suffix"),
		Value:  composeK1(chi.URLParam(r, "k1Prefix"), chi.URLParam(r, "k1Suffix")),
	}

	result, err := h.svc.GetK1(ctx, key, queries)
	if err != nil {
		h.handleError(w, "getk1 failed", err)
		return
	}

	payload := packTuples(result)
	writeBinary(w, http.StatusOK, payload)
}

// Set обрабатывает PUT /swarm/... .
func (h *Handler) Set(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	key := service.Key{
		K1: composeK1(chi.URLParam(r, "k1Prefix"), chi.URLParam(r, "k1Suffix")),
		K2: chi.URLParam(r, "k2"),
		K3: chi.URLParam(r, "k3"),
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.log.Warn("failed to read body", zap.Error(err))
		writeStatus(w, http.StatusBadRequest)
		return
	}

	if len(body) == 0 {
		writeStatus(w, http.StatusBadRequest)
		return
	}

	tar := service.Tuple{K1: key.K1, K2: key.K2, K3: key.K3, Payload: body}

	fmt.Println(tar)

	if err := h.svc.Set(ctx, tar); err != nil {
		h.handleError(w, "set failed", err)
		return
	}

	writeStatus(w, http.StatusOK)
}

// Delete обрабатывает DELETE /swarm/... .
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	key := service.Key{
		K1: composeK1(chi.URLParam(r, "k1Prefix"), chi.URLParam(r, "k1Suffix")),
		K2: chi.URLParam(r, "k2"),
		K3: chi.URLParam(r, "k3"),
	}

	deleted, err := h.svc.Delete(ctx, key)
	if err != nil {
		h.handleError(w, "delete failed", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"deleted": deleted})
}

func (h *Handler) handleError(w http.ResponseWriter, msg string, err error) {
	status := service.StatusCode(err)
	if status >= 500 {
		h.log.Error(msg, zap.Error(err))
	} else {
		h.log.Warn(msg, zap.Error(err))
	}
	writeStatus(w, status)
}

func composeK1(prefix, suffix string) string {
	if suffix == "" {
		return prefix
	}
	return prefix + ":" + suffix
}

func writeStatus(w http.ResponseWriter, code int) {
	w.WriteHeader(code)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeBinary(w http.ResponseWriter, code int, payload []byte) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(code)
	if len(payload) > 0 {
		_, _ = w.Write(payload)
	}
}

func packTuples(tuples []service.Tuple) []byte {
	if len(tuples) == 0 {
		return nil
	}

	encoded := make([]swarm.Tuple, 0, len(tuples))
	for _, tuple := range tuples {
		encoded = append(encoded, swarm.Tuple{
			[]byte(tuple.K1),
			[]byte(tuple.K2),
			[]byte(tuple.K3),
			tuple.Payload,
		})
	}

	return swarm.PackTuplesList(encoded)
}
