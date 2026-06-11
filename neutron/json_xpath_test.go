package neutron

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chainreactors/sdk/pkg/types"
	"gopkg.in/yaml.v3"
)

func TestJSONExtractorTemplateParsesYAML(t *testing.T) {
	raw := `id: json-extract
info:
  name: JSON Extractor Test
  severity: info
http:
  - method: GET
    path:
      - "{{BaseURL}}/api"
    matchers:
      - type: status
        status:
          - 200
    extractors:
      - type: json
        json:
          - ".data.version"
          - ".items[].name"
`
	var tpl types.Template
	if err := yaml.Unmarshal([]byte(raw), &tpl); err != nil {
		t.Fatalf("parse: %v", err)
	}

	requests := tpl.GetRequests()
	if len(requests) != 1 {
		t.Fatalf("expected 1 http request, got %d", len(requests))
	}
	if len(requests[0].Extractors) != 1 {
		t.Fatalf("expected 1 extractor, got %d", len(requests[0].Extractors))
	}
	ext := requests[0].Extractors[0]
	if ext.Type != "json" {
		t.Fatalf("expected type=json, got %q", ext.Type)
	}
	if len(ext.JSON) != 2 {
		t.Fatalf("expected 2 json expressions, got %d", len(ext.JSON))
	}
	if ext.JSON[0] != ".data.version" {
		t.Fatalf("expected first json expr '.data.version', got %q", ext.JSON[0])
	}
}

func TestJSONMatcherTemplateParsesYAML(t *testing.T) {
	raw := `id: json-match
info:
  name: JSON Matcher Test
  severity: info
http:
  - method: GET
    path:
      - "{{BaseURL}}/api"
    matchers:
      - type: json
        json:
          - ".status == \"ok\""
`
	var tpl types.Template
	if err := yaml.Unmarshal([]byte(raw), &tpl); err != nil {
		t.Fatalf("parse: %v", err)
	}

	requests := tpl.GetRequests()
	if len(requests) != 1 {
		t.Fatalf("expected 1 http request, got %d", len(requests))
	}
	if len(requests[0].Matchers) != 1 {
		t.Fatalf("expected 1 matcher, got %d", len(requests[0].Matchers))
	}
	m := requests[0].Matchers[0]
	if m.Type != "json" {
		t.Fatalf("expected type=json, got %q", m.Type)
	}
	if len(m.JSON) != 1 {
		t.Fatalf("expected 1 json expression, got %d", len(m.JSON))
	}
}

func TestXPathExtractorTemplateParsesYAML(t *testing.T) {
	raw := `id: xpath-extract
info:
  name: XPath Extractor Test
  severity: info
http:
  - method: GET
    path:
      - "{{BaseURL}}/"
    matchers:
      - type: status
        status:
          - 200
    extractors:
      - type: xpath
        xpath:
          - "//title"
        attribute: text
`
	var tpl types.Template
	if err := yaml.Unmarshal([]byte(raw), &tpl); err != nil {
		t.Fatalf("parse: %v", err)
	}

	requests := tpl.GetRequests()
	if len(requests) != 1 {
		t.Fatalf("expected 1 http request, got %d", len(requests))
	}
	ext := requests[0].Extractors[0]
	if ext.Type != "xpath" {
		t.Fatalf("expected type=xpath, got %q", ext.Type)
	}
	if len(ext.XPath) != 1 || ext.XPath[0] != "//title" {
		t.Fatalf("unexpected xpath: %v", ext.XPath)
	}
	if ext.Attribute != "text" {
		t.Fatalf("expected attribute=text, got %q", ext.Attribute)
	}
}

func TestXPathMatcherTemplateParsesYAML(t *testing.T) {
	raw := `id: xpath-match
info:
  name: XPath Matcher Test
  severity: info
http:
  - method: GET
    path:
      - "{{BaseURL}}/"
    matchers:
      - type: xpath
        xpath:
          - "//div[@class='error']"
`
	var tpl types.Template
	if err := yaml.Unmarshal([]byte(raw), &tpl); err != nil {
		t.Fatalf("parse: %v", err)
	}

	m := tpl.GetRequests()[0].Matchers[0]
	if m.Type != "xpath" {
		t.Fatalf("expected type=xpath, got %q", m.Type)
	}
	if len(m.XPath) != 1 || m.XPath[0] != "//div[@class='error']" {
		t.Fatalf("unexpected xpath: %v", m.XPath)
	}
}

func TestJSONExtractorExecute(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"version":"1.2.3","name":"test"}}`))
	}))
	defer server.Close()

	tpl := parseTemplateForTest(t, `id: json-extract-e2e
info:
  name: JSON Extract E2E
  severity: info
http:
  - method: GET
    path:
      - "{{BaseURL}}/"
    matchers:
      - type: status
        status:
          - 200
    extractors:
      - type: json
        json:
          - ".data.version"
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
		t.Fatal("expected json extractor template to match")
	}
	found := false
	for _, vals := range result.Extracts {
		for _, v := range vals {
			if v == "1.2.3" {
				found = true
			}
		}
	}
	for _, v := range result.OutputExtracts {
		if v == "1.2.3" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected json extractor to extract '1.2.3', got extracts=%v output=%v", result.Extracts, result.OutputExtracts)
	}
}

func TestJSONMatcherExecute(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","count":42}`))
	}))
	defer server.Close()

	tpl := parseTemplateForTest(t, `id: json-match-e2e
info:
  name: JSON Match E2E
  severity: info
http:
  - method: GET
    path:
      - "{{BaseURL}}/"
    matchers:
      - type: json
        json:
          - ".status == \"ok\""
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
		t.Fatal("expected json matcher to match .status == \"ok\"")
	}
}

func TestXPathExtractorExecute(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><head><title>TestPage</title></head><body></body></html>`))
	}))
	defer server.Close()

	tpl := parseTemplateForTest(t, `id: xpath-extract-e2e
info:
  name: XPath Extract E2E
  severity: info
http:
  - method: GET
    path:
      - "{{BaseURL}}/"
    matchers:
      - type: status
        status:
          - 200
    extractors:
      - type: xpath
        xpath:
          - "//title"
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
		t.Fatal("expected xpath extractor template to match")
	}
	found := false
	for _, vals := range result.Extracts {
		for _, v := range vals {
			if v == "TestPage" {
				found = true
			}
		}
	}
	for _, v := range result.OutputExtracts {
		if v == "TestPage" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected xpath extractor to extract 'TestPage', got extracts=%v output=%v", result.Extracts, result.OutputExtracts)
	}
}

func TestXPathMatcherExecute(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body><div class="error">something wrong</div></body></html>`))
	}))
	defer server.Close()

	tpl := parseTemplateForTest(t, `id: xpath-match-e2e
info:
  name: XPath Match E2E
  severity: info
http:
  - method: GET
    path:
      - "{{BaseURL}}/"
    matchers:
      - type: xpath
        xpath:
          - "//div[@class='error']"
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
		t.Fatal("expected xpath matcher to match //div[@class='error']")
	}
}
