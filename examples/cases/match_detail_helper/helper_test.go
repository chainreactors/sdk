// helper_test.go 既是 helper.go 的回归测试，也是它最小的使用演示。
package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chainreactors/fingers/common"
	fingersEngine "github.com/chainreactors/fingers/fingers"
	sdkfingers "github.com/chainreactors/sdk/fingers"
)

func detectInlineForTest(t *testing.T, target, marker, fingerName string) []FingerMatch {
	t.Helper()
	finger := &fingersEngine.Finger{
		Name:     fingerName,
		Protocol: "http",
		Rules:    fingersEngine.Rules{{Regexps: &fingersEngine.Regexps{Body: []string{marker}}}},
	}
	eng, err := sdkfingers.NewEngine(
		sdkfingers.NewConfig().WithFingers(fingersEngine.Fingers{finger}),
	)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Get(target)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	results, err := eng.MatchHTTPWithDetail(resp)
	if err != nil {
		t.Fatal(err)
	}
	return FromMatchResults(results)
}

func TestFromMatchResults(t *testing.T) {
	framework := &common.Framework{
		Name:       "WordPress",
		Tags:       []string{"cms"},
		Attributes: &common.Attributes{Version: "6.4"},
	}
	out := FromMatchResults([]sdkfingers.MatchResult{{
		Framework:    framework,
		MatchURL:     "https://x.test/",
		MatcherType:  "word",
		MatcherValue: "wp-content",
		RuleIndex:    3,
		SendData:     "scope=currentpath method=GET url=https://x.test/",
	}})
	if len(out) != 1 {
		t.Fatalf("expected 1 match, got %d", len(out))
	}
	m := out[0]
	if m.Name != "WordPress" || m.Version != "6.4" {
		t.Fatalf("framework fields not mapped: %+v", m)
	}
	if m.MatchURL != "https://x.test/" || m.MatcherType != "word" || m.MatcherValue != "wp-content" || m.RuleIndex != 3 {
		t.Fatalf("detail fields not mapped: %+v", m)
	}
}

func TestFromMatchResult_NilFramework(t *testing.T) {
	m := FromMatchResult(sdkfingers.MatchResult{MatchURL: "https://x.test/"})
	if m.MatchURL != "https://x.test/" {
		t.Fatalf("MatchURL should survive nil framework, got %+v", m)
	}
}

// TestDetectFingersDetail_E2E 端到端：用 inline finger + httptest 跑通
// MatchHTTPWithDetail -> FingerMatch 的完整链路。
func TestDetectFingersDetail_E2E(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("HelperDemoMarker in body"))
	}))
	defer srv.Close()

	matches := detectInlineForTest(t, srv.URL, "HelperDemoMarker", "helper-demo")
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	m := matches[0]
	if m.Name != "helper-demo" {
		t.Fatalf("expected helper-demo, got %q", m.Name)
	}
	if m.MatchURL != srv.URL {
		t.Fatalf("MatchURL should fall back to request URL %q, got %q", srv.URL, m.MatchURL)
	}
	if m.MatcherType == "" || m.MatcherValue == "" {
		t.Fatalf("matcher fields empty: %+v", m)
	}
	t.Logf("OK: %+v", m)
}
