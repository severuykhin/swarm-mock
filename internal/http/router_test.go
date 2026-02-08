package http

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"github.com/cxhub/swarm-mock/internal/service"
	"github.com/cxhub/swarm-mock/pkg/swarm"
)

func TestRouter(t *testing.T) {
	log := zap.NewNop()
	svc, err := service.NewStub()
	if err != nil {
		t.Fatalf("NewStub() error = %v", err)
	}

	r := NewRouter(log, svc)
	putBody := swarm.PackTuple([][]byte{[]byte("a:b"), []byte("c"), []byte("d"), []byte("value")})
	cases := []struct {
		name   string
		method string
		path   string
		body   []byte
	}{
		{"get", http.MethodGet, "/swarm/a:b/c/d/", nil},
		{"getNotFound", http.MethodGet, "/swarm/a:b/c/e/", nil},
		{"getK1", http.MethodGet, "/swarm/a:b/?t=c", nil},
		{"set", http.MethodPut, "/swarm/a:b/c/d/", putBody},
		{"delete", http.MethodDelete, "/swarm/a:b/c/d/", nil},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, bytes.NewReader(tt.body))
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code == 0 {
				t.Fatalf("expected status code for %s %s", tt.method, tt.path)
			}
		})
	}
}
