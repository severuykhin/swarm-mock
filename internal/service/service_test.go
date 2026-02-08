package service

import "testing"

func TestComposeStub(t *testing.T) {
	svc := NewStub()

	if svc == nil {
		t.Fatal("expected stub service instance")
	}
}
