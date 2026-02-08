package swarm

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
)

var (
	mockSingleton   *MockClient
	mockSingletonMu sync.Mutex
)

type MockClient struct {
	data map[string]string
	mu   sync.RWMutex
}

type Client interface {
	Get(ctx context.Context, k1, k2, k3 string) (Tuple, error)
	Set(ctx context.Context, k1, k2, k3 string, value ...[]byte) error
	Delete(ctx context.Context, k1, k2, k3 string) (bool, error)
	Exists(ctx context.Context, k1, k2, k3 string) (bool, error)
	GetKeyValue(ctx context.Context, k1, k2, k3 string) (*Value, error)
	GetKeyValues(ctx context.Context, k1 string, k2 []string) (*Value, error)
	GetK1(ctx context.Context, k1 string, k2 []string) ([]Tuple, error)
	// MockData ничего не делает в обычном клиенте, в mock возвращает все данные
	MockData() map[string]string
}

var _ Client = (*MockClient)(nil)

func mockKey(k1, k2, k3 string) string {
	return fmt.Sprintf("%s/%s/%s", k1, k2, k3)
}

func (client *MockClient) Get(_ context.Context, k1, k2, k3 string) (Tuple, error) {

	client.mu.RLock()
	body, exists := client.data[mockKey(k1, k2, k3)]
	client.mu.RUnlock()
	if !exists {
		return nil, nil
	}
	return UnpackTuple([]byte(body))
}

func (client *MockClient) Set(_ context.Context, k1, k2, k3 string, value ...[]byte) error {

	client.mu.Lock()
	client.data[mockKey(k1, k2, k3)] = string(PackTuple(value))
	client.mu.Unlock()
	return nil
}

func (client *MockClient) Delete(_ context.Context, k1, k2, k3 string) (bool, error) {

	client.mu.Lock()
	_, exists := client.data[mockKey(k1, k2, k3)]
	delete(client.data, mockKey(k1, k2, k3))
	client.mu.Unlock()
	return exists, nil
}

func (client *MockClient) Exists(_ context.Context, k1, k2, k3 string) (bool, error) {

	client.mu.RLock()
	_, exists := client.data[mockKey(k1, k2, k3)]
	client.mu.RUnlock()
	return exists, nil
}

func (client *MockClient) GetKeyValue(ctx context.Context, k1, k2, k3 string) (*Value, error) {
	tuple, err := client.Get(ctx, k1, k2, k3)
	if err != nil {
		return nil, err
	}
	return &Value{value: tuple}, nil
}

func (client *MockClient) MockData() map[string]string {
	res := make(map[string]string)
	client.mu.RLock()
	for k, v := range client.data {
		res[k] = v
	}
	client.mu.RUnlock()
	return res
}

// GetK1 возвращает значение по k1 и списку k2. Если ключ не найден и не было ошибок то вернется nil, nil
func (client *MockClient) GetK1(_ context.Context, k1 string, k2 []string) ([]Tuple, error) {
	result := make([]Tuple, 0)

	k2Set := make(map[string]bool)
	for _, t := range k2 {
		k2Set[t] = true
	}

	prefix := k1 + "/"
	client.mu.RLock()
	defer client.mu.RUnlock()

	for k, body := range client.data {
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		keys := strings.Split(k, "/")

		if len(k2) > 0 && !k2Set[keys[TupleK2]] {
			continue
		}

		tuple := Tuple{Field(keys[TupleK1]), Field(keys[TupleK2]), Field(keys[TupleK3])}
		values, err := UnpackTuple([]byte(body))
		if err != nil {
			return nil, err
		}
		tuple = append(tuple, values...)
		result = append(result, tuple)
	}

	if len(result) == 0 {
		return nil, nil
	}

	sort.Slice(result, func(i, j int) bool { return bytes.Compare(result[i][1], result[j][1]) < 0 })

	return result, nil
}

func (client *MockClient) GetKeyValues(ctx context.Context, k1 string, k2 []string) (*Value, error) {
	tuples, err := client.GetK1(ctx, k1, k2)
	if err != nil {
		return nil, err
	}
	return &Value{values: tuples}, nil
}

func (client *MockClient) Reset() {
	client.mu.Lock()
	client.data = make(map[string]string)
	client.mu.Unlock()
}
