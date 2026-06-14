package proton

import (
	"context"
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
	eng := mustEngine(t, NewConfig().
		WithCapacity(1).
		WithTemplatePaths(writeTempTemplate(t, tmplPrivateKey)))

	var ctx *Context
	resultCh, err := eng.Execute(ctx, NewScanDataTask([]byte("PRIVATE KEY\n"), "test.txt"))
	if err != nil {
		t.Fatalf("execute with typed nil context: %v", err)
	}
	for range resultCh {
	}
}
