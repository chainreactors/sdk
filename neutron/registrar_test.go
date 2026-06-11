package neutron

import (
	"testing"

	"github.com/chainreactors/neutron/operators"
	"github.com/chainreactors/sdk/pkg/types"
)

func TestRegisterCustomExtractorType(t *testing.T) {
	const testExtType operators.ExtractorType = 200
	called := false

	operators.RegisterExtractorType("test-custom", testExtType, nil,
		func(e *operators.Extractor, corpus string, data map[string]interface{}) map[string]struct{} {
			called = true
			return map[string]struct{}{"custom-value": {}}
		},
	)

	if called {
		t.Fatal("extract func should not be called at registration time")
	}
}

func TestRegisterCustomMatcherType(t *testing.T) {
	const testMatchType operators.MatcherType = 200
	called := false

	operators.RegisterMatcherType("test-custom-match", testMatchType, nil,
		func(m *operators.Matcher, corpus string, data map[string]interface{}) (bool, []string) {
			called = true
			return true, nil
		},
	)

	if called {
		t.Fatal("match func should not be called at registration time")
	}
}

func TestNewTypeAliasesExist(t *testing.T) {
	var _ types.SSLTemplateRequest
	var _ types.ExtractorType = types.XPathExtractor
	var _ types.MatcherType = types.XPathMatcher
	var _ types.ExtractorType = types.JSONExtractor
	var _ types.MatcherType = types.JSONMatcher

	if types.XPathExtractor == 0 {
		t.Fatal("XPathExtractor should be non-zero")
	}
	if types.XPathMatcher == 0 {
		t.Fatal("XPathMatcher should be non-zero")
	}
	if types.JSONExtractor == 0 {
		t.Fatal("JSONExtractor should be non-zero")
	}
	if types.JSONMatcher == 0 {
		t.Fatal("JSONMatcher should be non-zero")
	}
}
