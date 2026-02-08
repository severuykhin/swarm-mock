package service

import "errors"

var (
	// ErrNotFound представляет отсутствие ключа в хранилище.
	ErrNotFound = errors.New("swarm: key not found")
)
