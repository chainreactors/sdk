// Test-based 演示：直接用 go test 跑通整条链路，证明
// MatchHTTPWithDetail 会返回 match_url 和 matcher 详情。
//
// Run with:
//
//	go test ./examples/cases/match_detail -v
package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	fingersEngine "github.com/chainreactors/fingers/fingers"
	sdkfingers "github.com/chainreactors/sdk/fingers"
)

// TestMatchDetailUsage 演示获取 matcher 详情的最小调用流程：
//
//  1. 构造 Engine
//  2. 调 MatchHTTPWithDetail(resp)
//  3. 直接读 MatchResult.MatchURL / MatcherType / MatcherValue
func TestMatchDetailUsage(t *testing.T) {
	// —— 演示用：inline finger + httptest。
	// 实际工程里换成 WithCyberhub / WithLocalFile 即可。
	finger := &fingersEngine.Finger{
		Name:     "demo-app",
		Protocol: "http",
		Rules: fingersEngine.Rules{
			{Regexps: &fingersEngine.Regexps{Body: []string{"DemoMarker"}}},
		},
	}
	eng, err := sdkfingers.NewEngine(
		sdkfingers.NewConfig().WithFingers(fingersEngine.Fingers{finger}),
	)
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("response with DemoMarker"))
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	// MatchHTTPWithDetail 会自动打开 MatchDetail，并返回扁平结果。
	results, err := eng.MatchHTTPWithDetail(resp)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Framework == nil || r.Framework.Name != "demo-app" {
		t.Fatalf("Framework not preserved: %+v", r)
	}
	if r.MatchURL != srv.URL {
		t.Fatalf("MatchURL should fall back to request URL %q, got %q", srv.URL, r.MatchURL)
	}
	if r.MatcherType == "" || r.MatcherValue == "" {
		t.Fatalf("expected matcher fields, got %+v", r)
	}
	t.Logf("[%s] match_url=%s matcher_type=%s matcher_value=%s rule_index=%d",
		r.Framework.Name, r.MatchURL, r.MatcherType, r.MatcherValue, r.RuleIndex)
}

// TestPlainMatchWithoutDetail 反向对照：plain MatchHTTP 仍保持兼容，
// 不会自动填 MatchDetail。
func TestPlainMatchWithoutDetail(t *testing.T) {
	finger := &fingersEngine.Finger{
		Name:     "demo-app-disabled",
		Protocol: "http",
		Rules: fingersEngine.Rules{
			{Regexps: &fingersEngine.Regexps{Body: []string{"DemoMarker2"}}},
		},
	}
	eng, _ := sdkfingers.NewEngine(
		sdkfingers.NewConfig().WithFingers(fingersEngine.Fingers{finger}),
	)
	// 故意不调 EnableMatchDetail()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("DemoMarker2"))
	}))
	defer srv.Close()
	resp, _ := http.Get(srv.URL)
	defer resp.Body.Close()

	frames, _ := eng.MatchHTTP(resp)
	if fw, ok := frames["demo-app-disabled"]; ok && fw.MatchDetail != nil {
		t.Fatalf("expected MatchDetail=nil for plain MatchHTTP, got %+v", *fw.MatchDetail)
	}
}
