package neutron

import (
	"context"
	"net/http"
	"net/http/httptest"
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("typed nil neutron marker"))
	}))
	defer server.Close()

	tpl := parseTemplateForTest(t, `id: typed-nil-context
info:
  name: typed nil context
  severity: info
http:
  - method: GET
    path:
      - "{{BaseURL}}/"
    matchers:
      - type: word
        words:
          - "typed nil neutron marker"
`)
	eng := &Engine{config: NewConfig()}
	eng.templates = eng.compileTemplates([]*types.Template{tpl})
	eng.SetCapacity(1)

	var ctx *Context
	resultCh, err := eng.Execute(ctx, NewExecuteTask(server.URL))
	if err != nil {
		t.Fatalf("execute with typed nil context: %v", err)
	}
	for range resultCh {
	}
}
