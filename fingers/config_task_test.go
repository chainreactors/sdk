package fingers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chainreactors/fingers/alias"
	fingersEngine "github.com/chainreactors/fingers/fingers"
	sdk "github.com/chainreactors/sdk/pkg"
)

func TestConfigSourceSelectionAndFiltering(t *testing.T) {
	cfg := NewConfig().
		WithFingers(fingersEngine.Fingers{
			{Name: "web-app", Protocol: "http"},
			{Name: "service-app", Protocol: "tcp"},
		}).
		WithAliases([]*alias.Alias{{Name: "web-app", Pocs: []string{"CVE-1"}}}).
		WithFilter(func(item *FullFinger) bool {
			return item != nil && item.Finger != nil && item.Finger.Protocol == "http"
		})

	if cfg.FullFingers.Len() != 1 {
		t.Fatalf("filtered fingers len = %d, want 1", cfg.FullFingers.Len())
	}
	item := cfg.FullFingers.Items["web-app"]
	if item == nil || item.Alias == nil || len(item.Alias.Pocs) != 1 {
		t.Fatalf("expected alias to be preserved after filter: %+v", item)
	}

	cfg.WithCyberhub("https://cyberhub.test", "key")
	if !cfg.IsRemoteEnabled() || cfg.Filename != "" || cfg.FullFingers.Len() != 0 {
		t.Fatalf("WithCyberhub did not reset local source: %+v", cfg)
	}

	cfg.WithLocalFile("fingers.yaml")
	if cfg.IsRemoteEnabled() || cfg.Filename != "fingers.yaml" || cfg.FullFingers.Len() != 0 {
		t.Fatalf("WithLocalFile did not reset remote source: %+v", cfg)
	}

	if err := NewConfig().WithCyberhub("https://cyberhub.test", "").Validate(); err == nil {
		t.Fatal("expected missing api key to fail validation")
	}
}

func TestExecuteMatchTaskUsesSDKResult(t *testing.T) {
	eng := newDetailTestEngine(t, NewConfig(), &fingersEngine.Finger{
		Name:     "execute-app",
		Protocol: "http",
		Rules: fingersEngine.Rules{{
			Regexps: &fingersEngine.Regexps{Body: []string{"ExecuteMarker"}},
		}},
	})

	resultCh, err := eng.Execute(NewContext(), NewMatchTask(rawHTTP("ExecuteMarker")))
	if err != nil {
		t.Fatal(err)
	}
	result := <-resultCh
	if result == nil || !result.Success() {
		t.Fatalf("expected successful result, got %#v err=%v", result, result.Error())
	}
	match := result.(*MatchResult)
	if !match.HasMatch() || match.Count() != 1 {
		t.Fatalf("expected one match, got %+v", match.Frameworks())
	}
	if match.Frameworks()["execute-app"] == nil {
		t.Fatalf("expected execute-app match, got %+v", match.Frameworks())
	}
}

func TestExecuteRejectsInvalidTaskAndContext(t *testing.T) {
	eng := newDetailTestEngine(t, NewConfig(), &fingersEngine.Finger{
		Name:     "task-app",
		Protocol: "http",
		Rules: fingersEngine.Rules{{
			Regexps: &fingersEngine.Regexps{Body: []string{"TaskMarker"}},
		}},
	})

	if _, err := eng.Execute(NewContext(), NewMatchTask(nil)); err == nil {
		t.Fatal("expected empty match task to fail validation")
	}
	if _, err := eng.Execute(fakeContext{ctx: context.Background()}, NewMatchTask(rawHTTP("TaskMarker"))); err == nil {
		t.Fatal("expected unsupported context type")
	}
	if _, err := eng.Execute(NewContext(), fakeTask{typ: "unknown"}); err == nil {
		t.Fatal("expected unsupported task type")
	}
}

func TestNewMatchTaskFromResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-SDK-Test", "ok")
		_, _ = w.Write([]byte("response-body"))
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	task := NewMatchTaskFromResponse(resp)
	if err := task.Validate(); err != nil {
		t.Fatal(err)
	}
	data := string(task.Data)
	if !containsAll(data, "X-Sdk-Test: ok", "response-body") {
		t.Fatalf("raw response missing expected content:\n%s", data)
	}
}

type fakeContext struct {
	ctx context.Context
}

func (f fakeContext) Context() context.Context {
	return f.ctx
}

type fakeTask struct {
	typ string
}

func (f fakeTask) Type() string {
	return f.typ
}

func (f fakeTask) Validate() error {
	return nil
}

var _ sdk.Context = fakeContext{}
var _ sdk.Task = fakeTask{}

func containsAll(s string, values ...string) bool {
	for _, value := range values {
		if !strings.Contains(s, value) {
			return false
		}
	}
	return true
}
