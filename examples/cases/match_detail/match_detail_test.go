package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	fingersEngine "github.com/chainreactors/fingers/fingers"
	sdkfingers "github.com/chainreactors/sdk/fingers"
)

func TestMatchDetailUsage(t *testing.T) {
	eng, err := sdkfingers.NewEngine(
		sdkfingers.NewConfig().WithFingers(fingersEngine.Fingers{{
			Name:     "demo-app",
			Protocol: "http",
			Rules: fingersEngine.Rules{{
				Regexps: &fingersEngine.Regexps{Body: []string{"DemoMarker"}},
			}},
		}}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := eng.EnableMatchDetail(); err != nil {
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

	frameworks, err := eng.MatchHTTP(resp)
	if err != nil {
		t.Fatal(err)
	}
	fw := frameworks["demo-app"]
	if fw == nil {
		t.Fatalf("expected demo-app, got %v", frameworks)
	}
	if fw.MatchDetail == nil {
		t.Fatal("expected MatchDetail")
	}
	if fw.MatchDetail.MatcherType != "body" || fw.MatchDetail.MatcherValue != "demomarker" {
		t.Fatalf("unexpected detail: %+v", *fw.MatchDetail)
	}
}
