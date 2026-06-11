package neutron

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chainreactors/sdk/pkg/types"
)

func TestExecuteWithTransportAndPathPrefix(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("path=" + r.URL.Path))
	}))
	defer server.Close()

	tpl := parseTemplateForTest(t, `id: rooturl-prefix
info:
  name: RootURL Prefix Test
  severity: info
http:
  - method: GET
    path:
      - "{{RootURL}}/health"
    matchers:
      - type: word
        words:
          - "path=/app/v1/health"
`)

	engine := &Engine{config: NewConfig()}
	compiled := engine.compileTemplates([]*types.Template{tpl})
	if len(compiled) != 1 {
		t.Fatalf("expected 1 compiled template, got %d", len(compiled))
	}

	result, err := compiled[0].ExecuteWithTransportAndPathPrefix(
		server.URL+"/app/v1/", nil, http.DefaultTransport, "/app/v1",
	)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result == nil || !result.Matched {
		t.Fatal("expected RootURL with PathPrefix to produce /app/v1/health")
	}
}

func TestExecuteWithTransportPreservesRedirectPolicy(t *testing.T) {
	redirectCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/redirect") {
			redirectCount++
			http.Redirect(w, r, "/final", http.StatusFound)
			return
		}
		_, _ = w.Write([]byte("final"))
	}))
	defer server.Close()

	tpl := parseTemplateForTest(t, `id: transport-redirect
info:
  name: Transport Redirect Test
  severity: info
http:
  - method: GET
    path:
      - "{{BaseURL}}/redirect"
    redirects: true
    max-redirects: 3
    matchers:
      - type: word
        words:
          - "final"
`)

	engine := &Engine{config: NewConfig()}
	compiled := engine.compileTemplates([]*types.Template{tpl})
	if len(compiled) != 1 {
		t.Fatalf("expected 1 compiled template, got %d", len(compiled))
	}

	result, err := compiled[0].ExecuteWithTransport(server.URL, nil, http.DefaultTransport)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result == nil || !result.Matched {
		t.Fatal("expected template to follow redirect and match 'final'")
	}
}
