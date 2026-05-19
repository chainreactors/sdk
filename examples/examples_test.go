package examples_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestExampleCommandsBuild(t *testing.T) {
	root := repoRoot(t)
	commands := []struct {
		name string
		args []string
	}{
		{"fingers", []string{"./examples/fingers/main.go"}},
		{"neutron", []string{"./examples/neutron/main.go"}},
		{"gogo", []string{"./examples/gogo/main.go"}},
		{"spray", []string{"./examples/spray/main.go"}},
		{"host_spray_sdk", []string{"./examples/spray/host_spray_sdk.go"}},
		{"filter", []string{"./examples/filter/main.go"}},
		{"sdk_usage", []string{"./examples/sdk_usage/main.go"}},
		{"engine_test", []string{"./examples/engine_test/main.go"}},
		{"test_all_engines", []string{"./examples/test_all_engines/main.go"}},
		{"match_detail_case", []string{"./examples/cases/match_detail"}},
		{"spray_crawl_finger_case", []string{"./examples/cases/spray_crawl_finger"}},
	}

	for _, tc := range commands {
		t.Run(tc.name, func(t *testing.T) {
			output := filepath.Join(t.TempDir(), tc.name)
			if runtime.GOOS == "windows" {
				output += ".exe"
			}
			args := append([]string{"build", "-o", output}, tc.args...)
			if out, err := runGo(t, root, 2*time.Minute, args...); err != nil {
				t.Fatalf("go %s failed: %v\n%s", strings.Join(args, " "), err, out)
			}
		})
	}
}

func TestExampleCommandUsageSmoke(t *testing.T) {
	root := repoRoot(t)
	commands := []struct {
		name string
		args []string
		want string
	}{
		{"fingers", []string{"run", "./examples/fingers/main.go"}, "Usage: fingers"},
		{"neutron", []string{"run", "./examples/neutron/main.go"}, "Usage: neutron"},
		{"gogo", []string{"run", "./examples/gogo/main.go"}, "Usage: gogo"},
		{"spray", []string{"run", "./examples/spray/main.go"}, "Usage: spray"},
		{"host_spray_sdk", []string{"run", "./examples/spray/host_spray_sdk.go"}, "Usage: host_spray_sdk"},
		{"match_detail_case", []string{"run", "./examples/cases/match_detail"}, "-target"},
		{"spray_crawl_finger_case", []string{"run", "./examples/cases/spray_crawl_finger"}, "-target"},
	}

	for _, tc := range commands {
		t.Run(tc.name, func(t *testing.T) {
			out, err := runGo(t, root, time.Minute, tc.args...)
			if err == nil {
				t.Fatalf("expected command to fail without required args\n%s", out)
			}
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) {
				t.Fatalf("expected exit error, got %T: %v\n%s", err, err, out)
			}
			if !strings.Contains(out, tc.want) {
				t.Fatalf("expected output to contain %q\n%s", tc.want, out)
			}
		})
	}
}

func TestFilterExampleSmoke(t *testing.T) {
	root := repoRoot(t)
	out, err := runGo(t, root, time.Minute, "run", "./examples/filter/main.go")
	if err != nil {
		t.Fatalf("filter example failed: %v\n%s", err, out)
	}
	for _, want := range []string{
		"Fingers ExportFilter:",
		"FullFingers.Filter: OK",
		"Templates.Filter (severity): OK",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q\n%s", want, out)
		}
	}
}

func TestSprayExampleChecksLocalHTTP(t *testing.T) {
	root := repoRoot(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprint(w, `<html><head><title>SDK CI</title></head><body>ok</body></html>`)
	}))
	defer srv.Close()

	out, err := runGo(
		t,
		root,
		2*time.Minute,
		"run",
		"./examples/spray/main.go",
		"-u",
		srv.URL,
		"-json",
		"-threads",
		"1",
		"-timeout",
		"3",
		"-mc",
		"200",
	)
	if err != nil {
		t.Fatalf("spray example failed: %v\n%s", err, out)
	}

	var payload struct {
		TotalURLs    int `json:"total_urls"`
		Processed    int `json:"processed"`
		MatchedCount int `json:"matched_count"`
		Results      []struct {
			URL    string `json:"url"`
			Status int    `json:"status"`
			Title  string `json:"title"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(extractJSONObject(t, out)), &payload); err != nil {
		t.Fatalf("failed to parse json output: %v\n%s", err, out)
	}
	if payload.TotalURLs != 1 || payload.Processed != 1 || payload.MatchedCount != 1 {
		t.Fatalf("unexpected counters: %+v\n%s", payload, out)
	}
	if len(payload.Results) != 1 {
		t.Fatalf("expected one result: %+v\n%s", payload, out)
	}
	if payload.Results[0].URL != srv.URL || payload.Results[0].Status != http.StatusOK {
		t.Fatalf("unexpected spray result: %+v\n%s", payload.Results[0], out)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find repo root from %s", dir)
		}
		dir = parent
	}
}

func runGo(t *testing.T, root string, timeout time.Duration, args ...string) (string, error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = root
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("go %s timed out\n%s", strings.Join(args, " "), out.String())
	}
	return out.String(), err
}

func extractJSONObject(t *testing.T, output string) string {
	t.Helper()

	start := strings.Index(output, "{")
	end := strings.LastIndex(output, "}")
	if start < 0 || end < start {
		t.Fatalf("output does not contain a json object\n%s", output)
	}
	return output[start : end+1]
}
