package spray

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestContextNormalizesNilContext(t *testing.T) {
	if NewContext().WithContext(nil).Context() == nil {
		t.Fatal("WithContext(nil) returned nil context")
	}

	var ctx *Context
	if ctx.Context() == nil {
		t.Fatal("nil receiver Context returned nil")
	}
	if ctx.WithContext(nil).Context() == nil {
		t.Fatal("nil receiver WithContext(nil) returned nil context")
	}
}

func TestContextPreservesCancelledContext(t *testing.T) {
	base, cancel := context.WithCancel(context.Background())
	cancel()

	if err := NewContext().WithContext(base).Context().Err(); err == nil {
		t.Fatal("cancelled context was not preserved")
	}
}

func TestExecuteHandlesTypedNilContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("spray typed nil"))
	}))
	defer server.Close()

	eng, err := NewEngine(nil)
	if err != nil {
		t.Fatal(err)
	}

	var ctx *Context
	resultCh, err := eng.Execute(ctx, NewCheckTask([]string{server.URL}))
	if err != nil {
		t.Fatalf("execute with typed nil context: %v", err)
	}
	for range resultCh {
	}
}
