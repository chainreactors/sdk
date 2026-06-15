package spray

import (
	"context"
	"testing"
)

func TestNilReceiverContextDoesNotPanic(t *testing.T) {
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

func TestExecuteRejectsTypedNilContext(t *testing.T) {
	eng, err := NewEngine(nil)
	if err != nil {
		t.Fatal(err)
	}

	var ctx *Context
	_, err = eng.Execute(ctx, NewCheckTask([]string{"http://127.0.0.1:1"}))
	if err == nil {
		t.Fatal("expected error for typed nil context, got nil")
	}
}
