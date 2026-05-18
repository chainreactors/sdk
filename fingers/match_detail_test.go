package fingers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	fingersEngine "github.com/chainreactors/fingers/fingers"
)

func TestEnableMatchDetailPassiveMatch(t *testing.T) {
	eng := newDetailTestEngine(t, &fingersEngine.Finger{
		Name:     "passive-app",
		Protocol: "http",
		Rules: fingersEngine.Rules{{
			Regexps: &fingersEngine.Regexps{Body: []string{"PassiveMarker"}},
		}},
	})

	frames, err := eng.Match(rawHTTP("PassiveMarker"))
	if err != nil {
		t.Fatal(err)
	}
	if fw := frames["passive-app"]; fw == nil || fw.MatchDetail != nil {
		t.Fatalf("MatchDetail should be nil before enabling, got %+v", fw)
	}

	if err := eng.EnableMatchDetail(); err != nil {
		t.Fatal(err)
	}
	frames, err = eng.Match(rawHTTP("PassiveMarker"))
	if err != nil {
		t.Fatal(err)
	}
	detail := frames["passive-app"].MatchDetail
	if detail == nil {
		t.Fatal("expected MatchDetail after EnableMatchDetail")
	}
	if detail.RuleIndex != 0 || detail.MatcherType != "body" || detail.MatcherValue != "passivemarker" {
		t.Fatalf("unexpected detail: %+v", *detail)
	}
	if detail.SendData != "" {
		t.Fatalf("passive match should not set send_data, got %q", detail.SendData)
	}
}

func TestEnableMatchDetailActiveHTTPMatch(t *testing.T) {
	eng := newDetailTestEngine(t, &fingersEngine.Finger{
		Name:        "active-app",
		Protocol:    "http",
		SendDataStr: "/probe",
		Rules: fingersEngine.Rules{{
			Regexps: &fingersEngine.Regexps{Body: []string{"ActiveMarker"}},
		}},
	})
	if err := eng.EnableMatchDetail(); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/probe" {
			_, _ = w.Write([]byte("ActiveMarker"))
			return
		}
		_, _ = w.Write([]byte("no marker"))
	}))
	defer srv.Close()

	results, err := eng.HTTPMatch(NewContext().WithLevel(1), []string{srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || len(results[0].Results) != 1 {
		t.Fatalf("expected one active match, got %+v", results)
	}
	detail := results[0].Results[0].Framework.MatchDetail
	if detail == nil {
		t.Fatal("expected active MatchDetail")
	}
	if detail.MatcherType != "body" || detail.MatcherValue != "activemarker" || detail.SendData != "/probe" {
		t.Fatalf("unexpected active detail: %+v", *detail)
	}
}

func TestEnableMatchDetailNilSafe(t *testing.T) {
	var nilEngine *Engine
	if err := nilEngine.EnableMatchDetail(); err != nil {
		t.Fatal(err)
	}
	if err := (&Engine{}).EnableMatchDetail(); err != nil {
		t.Fatal(err)
	}
}

func newDetailTestEngine(t *testing.T, finger *fingersEngine.Finger) *Engine {
	t.Helper()
	eng, err := NewEngine(NewConfig().WithFingers(fingersEngine.Fingers{finger}))
	if err != nil {
		t.Fatal(err)
	}
	return eng
}

func rawHTTP(body string) []byte {
	return []byte("HTTP/1.1 200 OK\r\nContent-Type: text/html\r\n\r\n" + body)
}
