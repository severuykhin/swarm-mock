package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

const snapshotVersion = 1

// JSONPersistence реализует сохранение состояния Stub в JSON-файл.
type JSONPersistence struct {
	path string
}

// NewJSONPersistence создаёт JSONPersistence по указанному пути.
func NewJSONPersistence(path string) *JSONPersistence {
	return &JSONPersistence{path: path}
}

// Load читает снапшот и восстанавливает tuples.
func (p *JSONPersistence) Load(ctx context.Context) ([]Tuple, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	file, err := os.Open(p.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("persistence open: %w", err)
	}
	defer file.Close()

	var snapshot snapshotFile
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&snapshot); err != nil {
		return nil, fmt.Errorf("persistence decode: %w", err)
	}

	tuples := make([]Tuple, 0, len(snapshot.Tuples))
	for _, entry := range snapshot.Tuples {
		payload, err := base64.StdEncoding.DecodeString(entry.Payload)
		if err != nil {
			return nil, fmt.Errorf("persistence decode payload %s/%s/%s: %w", entry.K1, entry.K2, entry.K3, err)
		}
		tuples = append(tuples, Tuple{
			K1:      entry.K1,
			K2:      entry.K2,
			K3:      entry.K3,
			Payload: payload,
		})
	}

	return tuples, nil
}

// Save сериализует tuples в снапшот и записывает на диск атомарно.
func (p *JSONPersistence) Save(ctx context.Context, tuples []Tuple) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	snapshot := snapshotFile{
		Version:   snapshotVersion,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	snapshot.Tuples = make([]snapshotTuple, 0, len(tuples))
	for _, tuple := range tuples {
		snapshot.Tuples = append(snapshot.Tuples, snapshotTuple{
			K1:      tuple.K1,
			K2:      tuple.K2,
			K3:      tuple.K3,
			Payload: base64.StdEncoding.EncodeToString(tuple.Payload),
		})
	}

	dir := filepath.Dir(p.path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("persistence mkdir: %w", err)
		}
	}

	tmp, err := os.CreateTemp(dir, filepath.Base(p.path)+".tmp-")
	if err != nil {
		return fmt.Errorf("persistence temp file: %w", err)
	}
	defer func() {
		_ = os.Remove(tmp.Name())
	}()

	encoder := json.NewEncoder(tmp)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(snapshot); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("persistence encode: %w", err)
	}

	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("persistence sync: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("persistence close: %w", err)
	}

	if err := os.Rename(tmp.Name(), p.path); err != nil {
		return fmt.Errorf("persistence rename: %w", err)
	}

	if dirFile, err := os.Open(dir); err == nil {
		_, _ = dirFile.Stat()
		_ = dirFile.Sync()
		_ = dirFile.Close()
	}

	return nil
}

type snapshotFile struct {
	Version   int             `json:"version"`
	UpdatedAt string          `json:"updated_at"`
	Tuples    []snapshotTuple `json:"tuples"`
}

type snapshotTuple struct {
	K1      string `json:"k1"`
	K2      string `json:"k2"`
	K3      string `json:"k3"`
	Payload string `json:"payload"`
}
