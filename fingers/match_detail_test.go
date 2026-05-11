package fingers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chainreactors/fingers/common"
	fingersEngine "github.com/chainreactors/fingers/fingers"
)

// TestMatchHTTPWithDetail_HappyPath: 端到端验证客户视角的最短调用。
// 不需要调 EnableMatchDetail / GetFingersEngine / 解析 SendData。
func TestMatchHTTPWithDetail_HappyPath(t *testing.T) {
	eng := newTestEngine(t, "demo-app", "DemoMarker")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("body with DemoMarker"))
	}))
	defer srv.Close()
	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

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
		t.Fatalf("matcher fields empty: %+v", r)
	}
}

// TestMatchWithDetail_FallbackURL: 已有原始字节 + 传入 URL 的场景
func TestMatchWithDetail_FallbackURL(t *testing.T) {
	eng := newTestEngine(t, "raw-app", "RawMarker")
	raw := []byte("HTTP/1.1 200 OK\r\nContent-Type: text/html\r\n\r\nbody with RawMarker here")
	results, err := eng.MatchWithDetail(raw, "https://provided.example/path")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}
	if results[0].MatchURL != "https://provided.example/path" {
		t.Fatalf("MatchURL fallback not used; got %q", results[0].MatchURL)
	}
}

// TestEnableMatchDetail_AffectsPlainMatch: 显式调用 EnableMatchDetail 后,
// 即便走 plain Match() 也能拿到 MatchDetail
func TestEnableMatchDetail_AffectsPlainMatch(t *testing.T) {
	eng := newTestEngine(t, "plain-app", "PlainMarker")
	eng.EnableMatchDetail()
	frames, err := eng.Match([]byte("HTTP/1.1 200 OK\r\n\r\nPlainMarker"))
	if err != nil {
		t.Fatal(err)
	}
	fw, ok := frames["plain-app"]
	if !ok {
		t.Fatalf("expected plain-app, got: %v", frames)
	}
	if fw.MatchDetail == nil {
		t.Fatal("MatchDetail should be populated after EnableMatchDetail()")
	}
}

// TestPlainMatchWithoutEnable_NoMatchDetail: 反向对照,不调 EnableMatchDetail
// 时 plain Match 不应填充 MatchDetail (证明 EnableMatchDetail 是必需的)
func TestPlainMatchWithoutEnable_NoMatchDetail(t *testing.T) {
	eng := newTestEngine(t, "off-app", "OffMarker")
	frames, _ := eng.Match([]byte("HTTP/1.1 200 OK\r\n\r\nOffMarker"))
	if fw, ok := frames["off-app"]; ok && fw.MatchDetail != nil {
		t.Fatalf("MatchDetail should be nil without EnableMatchDetail(), got %+v", *fw.MatchDetail)
	}
}

// TestFlattenMatchResults_SendDataURLWins: SendData 含 url= 时优先用它
func TestFlattenMatchResults_SendDataURLWins(t *testing.T) {
	frames := common.Frameworks{
		"x": &common.Framework{
			Name: "x",
			MatchDetail: &common.MatchDetail{
				MatcherType:  "word",
				MatcherValue: "foo",
				SendData:     "scope=cp method=GET url=https://active.example/admin",
			},
		},
	}
	out := flattenMatchResults(frames, "https://fallback.example/")
	if len(out) != 1 || out[0].MatchURL != "https://active.example/admin" {
		t.Fatalf("expected MatchURL from SendData, got %+v", out)
	}
}

// TestExtractMatchURL: 词边界 / 含 = 的 url value / 缺失 / 子串误匹配
func TestExtractMatchURL(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"url=https://x.test/a", "https://x.test/a"},
		{"scope=cp method=GET url=https://x.test/p", "https://x.test/p"},
		{"url=https://x.test/api?next=https://y/foo=1", "https://x.test/api?next=https://y/foo=1"},
		{"scope=cp method=GET", ""},
		{"scope=anonymous_url=x method=GET", ""},
	}
	for _, c := range cases {
		if got := extractMatchURL(c.in); got != c.want {
			t.Errorf("extractMatchURL(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestDetailMethods_NilSafe(t *testing.T) {
	var nilEngine *Engine
	nilEngine.EnableMatchDetail()

	results, err := nilEngine.MatchWithDetail(nil, "https://fallback.example/")
	if err != nil || results != nil {
		t.Fatalf("nil MatchWithDetail = (%v, %v), want (nil, nil)", results, err)
	}

	results, err = nilEngine.MatchHTTPWithDetail(nil)
	if err != nil || results != nil {
		t.Fatalf("nil MatchHTTPWithDetail = (%v, %v), want (nil, nil)", results, err)
	}

	emptyEngine := &Engine{}
	results, err = emptyEngine.MatchHTTPWithDetail(nil)
	if err != nil || results != nil {
		t.Fatalf("empty MatchHTTPWithDetail = (%v, %v), want (nil, nil)", results, err)
	}
}

func newTestEngine(t *testing.T, name, marker string) *Engine {
	t.Helper()
	eng, err := NewEngine(NewConfig().WithFingers(fingersEngine.Fingers{{
		Name:     name,
		Protocol: "http",
		Rules:    fingersEngine.Rules{{Regexps: &fingersEngine.Regexps{Body: []string{marker}}}},
	}}))
	if err != nil {
		t.Fatal(err)
	}
	return eng
}
