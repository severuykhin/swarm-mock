package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"main/pkg/swarm"
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

// Persistence описывает хранилище снапшотов состояния Stub.
type Persistence interface {
	Load(ctx context.Context) ([]Tuple, error)
	Save(ctx context.Context, tuples []Tuple) error
}

// StubOption конфигурирует создание Stub.
type StubOption func(*stubConfig) error

type stubConfig struct {
	persistence      Persistence
	loadCtx          context.Context
	autosaveInterval time.Duration
}

// Stub реализует KVService, используя in-memory хранилище.
type Stub struct {
	mu   sync.RWMutex
	data map[string]record

	persistence Persistence

	autosave struct {
		interval time.Duration
		mu       sync.Mutex
		started  bool
		wg       sync.WaitGroup
	}
}

type record struct {
	key     Key
	payload []byte
}

// NewStub создает Stub с указанными опциями.
func NewStub(opts ...StubOption) (*Stub, error) {
	cfg := stubConfig{
		loadCtx: context.Background(),
	}

	for _, opt := range opts {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	if cfg.loadCtx == nil {
		cfg.loadCtx = context.Background()
	}

	stub := &Stub{
		data:        make(map[string]record),
		persistence: cfg.persistence,
	}
	stub.autosave.interval = cfg.autosaveInterval

	if stub.persistence != nil {
		tuples, err := stub.persistence.Load(cfg.loadCtx)
		if err != nil {
			return nil, fmt.Errorf("service: load snapshot: %w", err)
		}
		for _, tuple := range tuples {
			stub.data[storageKey(tuple.K1, tuple.K2, tuple.K3)] = record{
				key:     Key{K1: tuple.K1, K2: tuple.K2, K3: tuple.K3},
				payload: copyBytes(tuple.Payload),
			}
		}
	}

	return stub, nil
}

// WithPersistence задаёт источник сохранения состояния для Stub.
func WithPersistence(p Persistence) StubOption {
	return func(cfg *stubConfig) error {
		cfg.persistence = p
		return nil
	}
}

// WithLoadContext задаёт контекст, используемый при загрузке состояния.
func WithLoadContext(ctx context.Context) StubOption {
	return func(cfg *stubConfig) error {
		cfg.loadCtx = ctx
		return nil
	}
}

// WithAutosaveInterval конфигурирует периодичность автосохранения.
func WithAutosaveInterval(interval time.Duration) StubOption {
	return func(cfg *stubConfig) error {
		if interval < 0 {
			return fmt.Errorf("service: autosave interval must be non-negative")
		}
		cfg.autosaveInterval = interval
		return nil
	}
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

// SaveSnapshot выполняет сохранение текущего состояния через Persistence.
func (s *Stub) SaveSnapshot(ctx context.Context) error {
	if s.persistence == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	tuples := s.snapshot()
	return s.persistence.Save(ctx, tuples)
}

// StartAutosave запускает периодическое сохранение состояния Stub.
func (s *Stub) StartAutosave(ctx context.Context, onError func(error)) {
	if s.persistence == nil || s.autosave.interval <= 0 {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}

	s.autosave.mu.Lock()
	if s.autosave.started {
		s.autosave.mu.Unlock()
		return
	}
	s.autosave.started = true
	s.autosave.mu.Unlock()

	s.autosave.wg.Add(1)
	go func() {
		defer s.autosave.wg.Done()
		ticker := time.NewTicker(s.autosave.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.SaveSnapshot(ctx); err != nil {
					if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
						continue
					}
					if onError != nil {
						onError(err)
					}
				}
			}
		}
	}()
}

// WaitAutosave блокирует до завершения фонового воркера автосохранения.
func (s *Stub) WaitAutosave() {
	s.autosave.mu.Lock()
	started := s.autosave.started
	s.autosave.mu.Unlock()
	if !started {
		return
	}
	s.autosave.wg.Wait()
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

func (s *Stub) snapshot() []Tuple {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tuples := make([]Tuple, 0, len(s.data))
	for _, rec := range s.data {
		tuples = append(tuples, Tuple{
			K1:      rec.key.K1,
			K2:      rec.key.K2,
			K3:      rec.key.K3,
			Payload: copyBytes(rec.payload),
		})
	}

	sort.Slice(tuples, func(i, j int) bool {
		if tuples[i].K1 != tuples[j].K1 {
			return tuples[i].K1 < tuples[j].K1
		}
		if tuples[i].K2 != tuples[j].K2 {
			return tuples[i].K2 < tuples[j].K2
		}
		return tuples[i].K3 < tuples[j].K3
	})

	return tuples
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
