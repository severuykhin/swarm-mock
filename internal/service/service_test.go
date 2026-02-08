package service

import "testing"

func TestComposeStub(t *testing.T) {
	svc, err := NewStub()
	if err != nil {
		t.Fatalf("NewStub() error = %v", err)
	}

	if svc == nil {
		t.Fatal("expected stub service instance")
	}
}
