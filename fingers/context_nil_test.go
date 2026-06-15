package fingers

import (
	"context"
	"testing"

	"github.com/chainreactors/sdk/pkg/types"
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
	eng := newDetailTestEngine(t, NewConfig(), &types.Finger{
		Name:     "typed-nil-app",
		Protocol: "http",
		Rules: types.FingerRules{{
			Regexps: &types.FingerRegexps{Body: []string{"TypedNilMarker"}},
		}},
	})

	var ctx *Context
	_, err := eng.Execute(ctx, NewMatchTask(rawHTTP("TypedNilMarker")))
	if err == nil {
		t.Fatal("expected error for typed nil context, got nil")
	}
}
