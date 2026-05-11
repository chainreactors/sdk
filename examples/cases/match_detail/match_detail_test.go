// Test-based 演示：直接用 go test 跑通整条链路，证明
// EnableMatchDetail() 之后 framework.MatchDetail 真的会被填进去。
//
// Run with:
//   go test ./examples/cases/match_detail -v
package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	fingersEngine "github.com/chainreactors/fingers/fingers"
	sdkfingers "github.com/chainreactors/sdk/fingers"
	"github.com/chainreactors/utils/httputils"
)

// TestMatchDetailUsage 演示获取 matcher 详情的最小调用流程：
//
//  1. 构造 Engine
//  2. ★ NewEngine() 之后立刻翻开 MatchDetail
//     (engine.Compile() 会重置每条 finger 的开关，所以必须这里调)
//  3. 跑 DetectContent，从 framework.MatchDetail 读 matcher 详情
//  4. match_url 取值：MatchDetail.SendData 的 "url=" > 当前请求 URL（兜底）
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

	// ★ STEP 1：必须在 NewEngine 之后调用
	if fe, _ := eng.GetFingersEngine(); fe != nil {
		fe.EnableMatchDetail()
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

	// ★ STEP 2：跑匹配
	frames, err := eng.Get().DetectContent(httputils.ReadRaw(resp))
	if err != nil {
		t.Fatal(err)
	}

	// ★ STEP 3：读 MatchDetail，match_url 用兜底 URL
	fallbackURL := srv.URL
	if resp.Request != nil && resp.Request.URL != nil {
		fallbackURL = resp.Request.URL.String()
	}
	for _, fw := range frames {
		d := fw.MatchDetail
		if d == nil {
			continue
		}
		matchURL := fallbackURL
		if u := extractURL(d.SendData); u != "" {
			matchURL = u
		}
		t.Logf("[%s] match_url=%s matcher_type=%s matcher_value=%s rule_index=%d",
			fw.Name, matchURL, d.MatcherType, d.MatcherValue, d.RuleIndex)
	}

	// —— 把演示也当一条回归断言 ——
	fw, ok := frames["demo-app"]
	if !ok {
		t.Fatalf("expected match for demo-app, got: %v", frames)
	}
	if fw.MatchDetail == nil {
		t.Fatal("MatchDetail is nil — EnableMatchDetail() must be called after NewEngine()")
	}
	if fw.MatchDetail.MatcherType == "" || fw.MatchDetail.MatcherValue == "" {
		t.Fatalf("expected non-empty matcher fields, got %+v", *fw.MatchDetail)
	}
}

// TestMatchDetailRequiresEnable 反向对照：不调用 EnableMatchDetail() 时
// MatchDetail 为 nil，证明这一步是必需的不是可有可无。
func TestMatchDetailRequiresEnable(t *testing.T) {
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

	frames, _ := eng.Get().DetectContent(httputils.ReadRaw(resp))
	if fw, ok := frames["demo-app-disabled"]; ok && fw.MatchDetail != nil {
		t.Fatalf("expected MatchDetail=nil without EnableMatchDetail(), got %+v", *fw.MatchDetail)
	}
}

// 注意：extractURL 在 main.go 里已经定义，本文件直接调用，不再重复。
// 同一 package main 下编译时两文件共享所有顶层符号。
