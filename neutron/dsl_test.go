package neutron

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chainreactors/sdk/pkg/types"
)

func TestDSLReplaceRegex(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("path=" + r.URL.Path))
	}))
	defer server.Close()

	tpl := parseTemplateForTest(t, `id: dsl-replace-regex
info:
  name: DSL replace_regex test
  severity: info
variables:
  full_path: "/static/ueditor.config.js"
  dir_path: '{{replace_regex(full_path, "/[^/]*$", "/")}}'
http:
  - method: GET
    path:
      - "{{BaseURL}}{{dir_path}}check"
    matchers:
      - type: word
        words:
          - "path=/static/check"
`)

	engine := &Engine{config: NewConfig()}
	compiled := engine.compileTemplates([]*types.Template{tpl})
	if len(compiled) != 1 {
		t.Fatalf("expected 1 compiled template, got %d", len(compiled))
	}

	result, err := compiled[0].Execute(server.URL, nil)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result == nil || !result.Matched {
		t.Fatal("expected replace_regex to produce /static/ from /static/ueditor.config.js")
	}
}

func TestDSLExtractorInTemplate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("version=3.14.159"))
	}))
	defer server.Close()

	tpl := parseTemplateForTest(t, `id: dsl-extractor
info:
  name: DSL Extractor Test
  severity: info
http:
  - method: GET
    path:
      - "{{BaseURL}}/"
    matchers:
      - type: dsl
        dsl:
          - "status_code == 200"
    extractors:
      - type: dsl
        dsl:
          - "body"
`)

	engine := &Engine{config: NewConfig()}
	compiled := engine.compileTemplates([]*types.Template{tpl})
	if len(compiled) != 1 {
		t.Fatalf("expected 1 compiled template, got %d", len(compiled))
	}

	result, err := compiled[0].Execute(server.URL, nil)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result == nil || !result.Matched {
		t.Fatal("expected DSL matcher to match status_code == 200")
	}
	if len(result.OutputExtracts) == 0 {
		t.Fatalf("expected DSL extractor to produce output extracts, got none")
	}
}
