package fingers

import (
	"context"
	"testing"

	"github.com/chainreactors/sdk/pkg/types"
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
	eng := newDetailTestEngine(t, NewConfig(), &types.Finger{
		Name:     "typed-nil-app",
		Protocol: "http",
		Rules: types.FingerRules{{
			Regexps: &types.FingerRegexps{Body: []string{"TypedNilMarker"}},
		}},
	})

	var ctx *Context
	resultCh, err := eng.Execute(ctx, NewMatchTask(rawHTTP("TypedNilMarker")))
	if err != nil {
		t.Fatalf("execute with typed nil context: %v", err)
	}

	result := <-resultCh
	if result == nil || !result.Success() {
		t.Fatalf("expected successful result, got %#v err=%v", result, result.Error())
	}
}
