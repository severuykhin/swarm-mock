package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"sync"

	"scripts/swarm_stub/pkg/swarm"
)

// KVService описывает операции с KV-хранилищем Swarm.
type KVService interface {
	Get(ctx context.Context, keys Key) (Tuple, error)
	GetK1(ctx context.Context, key K1, k2 []string) ([]Tuple, error)
	Set(ctx context.Context, tuple Tuple) error
	Delete(ctx context.Context, keys Key) (bool, error)
}

// Tuple представляет запись Swarm.
type Tuple struct {
	K1      string
	K2      string
	K3      string
	Payload []byte
}

// Key содержит составные части ключа.
type Key struct {
	K1 string
	K2 string
	K3 string
}

// K1 описывает составной ключ первого уровня.
type K1 struct {
	Prefix string
	Suffix string
	Value  string
}

// ErrInvalidTuple представляет ошибки формата Tuple.
var ErrInvalidTuple = errors.New("swarm: invalid tuple payload")

// statusError связывает ошибку с HTTP-статусом.
type statusError struct {
	status int
	err    error
}

func (e *statusError) Error() string { return e.err.Error() }
func (e *statusError) Unwrap() error { return e.err }

// StatusCode возвращает HTTP-статус для ошибки сервиса.
func StatusCode(err error) int {
	var se *statusError
	if errors.As(err, &se) {
		return se.status
	}
	return http.StatusInternalServerError
}

func wrapStatus(err error, status int) error {
	return &statusError{status: status, err: err}
}

// Stub реализует KVService, используя in-memory хранилище.
type Stub struct {
	mu   sync.RWMutex
	data map[string]record
}

type record struct {
	key     Key
	payload []byte
}

// NewStub создает Stub.
func NewStub() *Stub {
	return &Stub{data: make(map[string]record)}
}

// Get возвращает Tuple либо ошибку ErrNotFound.
func (s *Stub) Get(ctx context.Context, keys Key) (Tuple, error) {
	if err := ctx.Err(); err != nil {
		return Tuple{}, contextError(err)
	}

	rec, ok := s.load(keys)
	if !ok {
		return Tuple{}, wrapStatus(ErrNotFound, http.StatusNotFound)
	}

	return Tuple{
		K1:      rec.key.K1,
		K2:      rec.key.K2,
		K3:      rec.key.K3,
		Payload: copyBytes(rec.payload),
	}, nil
}

// GetK1 возвращает список Tuple по ключу первого уровня.
func (s *Stub) GetK1(ctx context.Context, key K1, filter []string) ([]Tuple, error) {
	if err := ctx.Err(); err != nil {
		return nil, contextError(err)
	}

	var filterSet map[string]struct{}
	if len(filter) > 0 {
		filterSet = make(map[string]struct{}, len(filter))
		for _, item := range filter {
			filterSet[item] = struct{}{}
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	tuples := make([]Tuple, 0)
	for _, rec := range s.data {
		if rec.key.K1 != key.Value {
			continue
		}
		if len(filterSet) > 0 {
			if _, ok := filterSet[rec.key.K2]; !ok {
				continue
			}
		}
		tuples = append(tuples, Tuple{
			K1:      rec.key.K1,
			K2:      rec.key.K2,
			K3:      rec.key.K3,
			Payload: copyBytes(rec.payload),
		})
	}

	if len(tuples) == 0 {
		return nil, wrapStatus(ErrNotFound, http.StatusNotFound)
	}

	sort.Slice(tuples, func(i, j int) bool {
		if tuples[i].K2 == tuples[j].K2 {
			return tuples[i].K3 < tuples[j].K3
		}
		return tuples[i].K2 < tuples[j].K2
	})

	return tuples, nil
}

// Set сохраняет Tuple в хранилище.
func (s *Stub) Set(ctx context.Context, tuple Tuple) error {
	if err := ctx.Err(); err != nil {
		return contextError(err)
	}

	if tuple.K1 == "" {
		return wrapStatus(fmt.Errorf("K1 empty"), http.StatusBadRequest)
	}
	if tuple.K2 == "" {
		return wrapStatus(fmt.Errorf("K2 empty"), http.StatusBadRequest)
	}
	if len(tuple.Payload) == 0 {
		return wrapStatus(fmt.Errorf("payload empty"), http.StatusBadRequest)
	}

	fields, err := swarm.UnpackTuple(tuple.Payload)
	if err != nil {
		return wrapStatus(err, http.StatusBadRequest)
	}
	if len(fields) >= 3 {
		k1 := string(fields[swarm.TupleK1])
		k2 := string(fields[swarm.TupleK2])
		k3 := string(fields[swarm.TupleK3])
		if k1 != tuple.K1 || k2 != tuple.K2 || k3 != tuple.K3 {
			return wrapStatus(fmt.Errorf("tuple keys mismatch"), http.StatusBadRequest)
		}
	}

	rec := record{
		key:     Key{K1: tuple.K1, K2: tuple.K2, K3: tuple.K3},
		payload: copyBytes(tuple.Payload),
	}

	s.mu.Lock()
	s.data[storageKey(tuple.K1, tuple.K2, tuple.K3)] = rec
	s.mu.Unlock()

	return nil
}

// Delete удаляет Tuple по ключу.
func (s *Stub) Delete(ctx context.Context, keys Key) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, contextError(err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	storeKey := storageKey(keys.K1, keys.K2, keys.K3)
	if _, ok := s.data[storeKey]; !ok {
		return false, wrapStatus(ErrNotFound, http.StatusNotFound)
	}

	delete(s.data, storeKey)
	return true, nil
}

func (s *Stub) load(key Key) (record, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rec, ok := s.data[storageKey(key.K1, key.K2, key.K3)]
	return rec, ok
}

func storageKey(k1, k2, k3 string) string {
	return fmt.Sprintf("%s\x00%s\x00%s", k1, k2, k3)
}

func copyBytes(src []byte) []byte {
	if src == nil {
		return nil
	}
	dst := make([]byte, len(src))
	copy(dst, src)
	return dst
}

func contextError(err error) error {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return wrapStatus(err, http.StatusGatewayTimeout)
	case errors.Is(err, context.Canceled):
		return wrapStatus(err, http.StatusRequestTimeout)
	default:
		return wrapStatus(err, http.StatusInternalServerError)
	}
}
