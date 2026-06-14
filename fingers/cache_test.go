package fingers

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
)

func TestCachingSenderOnlyCachesSuccessfulResponses(t *testing.T) {
	calls := 0
	sender := cachingSender(func(_ []byte) ([]byte, bool) {
		calls++
		if calls == 1 {
			return nil, false
		}
		return []byte("ok"), true
	})

	if _, ok := sender([]byte("/probe")); ok {
		t.Fatal("first failed response should not be reported as success")
	}
	if calls != 1 {
		t.Fatalf("calls after first attempt = %d, want 1", calls)
	}

	resp, ok := sender([]byte("/probe"))
	if !ok || string(resp) != "ok" {
		t.Fatalf("second response = %q, %v; want ok,true", resp, ok)
	}
	if calls != 2 {
		t.Fatalf("failed response was cached; calls = %d, want 2", calls)
	}

	resp, ok = sender([]byte("/probe"))
	if !ok || string(resp) != "ok" {
		t.Fatalf("cached response = %q, %v; want ok,true", resp, ok)
	}
	if calls != 2 {
		t.Fatalf("successful response was not cached; calls = %d, want 2", calls)
	}
}

func TestPathCachedTransportSeparatesRequestVariants(t *testing.T) {
	base := &recordingRoundTripper{}
	transport := &pathCachedTransport{base: base, cache: make(map[string]*pathCachedEntry)}

	first := mustCachedRequest(t, http.MethodGet, "http://example.test/probe?a=1", "", "one")
	body := mustRoundTripBody(t, transport, first)
	if body != "GET /probe?a=1 one #1" {
		t.Fatalf("first body = %q", body)
	}

	firstAgain := mustCachedRequest(t, http.MethodGet, "http://example.test/probe?a=1", "", "one")
	body = mustRoundTripBody(t, transport, firstAgain)
	if body != "GET /probe?a=1 one #1" {
		t.Fatalf("cached body = %q", body)
	}

	differentQuery := mustCachedRequest(t, http.MethodGet, "http://example.test/probe?a=2", "", "one")
	body = mustRoundTripBody(t, transport, differentQuery)
	if body != "GET /probe?a=2 one #2" {
		t.Fatalf("query-variant body = %q", body)
	}

	differentHeader := mustCachedRequest(t, http.MethodGet, "http://example.test/probe?a=1", "", "two")
	body = mustRoundTripBody(t, transport, differentHeader)
	if body != "GET /probe?a=1 two #3" {
		t.Fatalf("header-variant body = %q", body)
	}

	postNoBody := mustCachedRequest(t, http.MethodPost, "http://example.test/probe?a=1", "", "one")
	body = mustRoundTripBody(t, transport, postNoBody)
	if body != "POST /probe?a=1 one #4" {
		t.Fatalf("method-variant body = %q", body)
	}

	if base.calls != 4 {
		t.Fatalf("base calls = %d, want 4", base.calls)
	}
}

func TestPathCachedTransportDoesNotCacheRequestsWithBody(t *testing.T) {
	base := &recordingRoundTripper{}
	transport := &pathCachedTransport{base: base, cache: make(map[string]*pathCachedEntry)}

	first := mustCachedRequest(t, http.MethodPost, "http://example.test/probe", "x=1", "")
	if body := mustRoundTripBody(t, transport, first); body != "POST /probe  #1" {
		t.Fatalf("first body request = %q", body)
	}
	second := mustCachedRequest(t, http.MethodPost, "http://example.test/probe", "x=1", "")
	if body := mustRoundTripBody(t, transport, second); body != "POST /probe  #2" {
		t.Fatalf("second body request = %q", body)
	}
	if base.calls != 2 {
		t.Fatalf("base calls = %d, want 2", base.calls)
	}
}

func TestPathCachedTransportReturnsIndependentHeaders(t *testing.T) {
	base := &recordingRoundTripper{}
	transport := &pathCachedTransport{base: base, cache: make(map[string]*pathCachedEntry)}

	req := mustCachedRequest(t, http.MethodGet, "http://example.test/probe", "", "")
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Header.Set("X-Call", "mutated")
	_ = resp.Body.Close()

	resp, err = transport.RoundTrip(mustCachedRequest(t, http.MethodGet, "http://example.test/probe", "", ""))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if got := resp.Header.Get("X-Call"); got != "1" {
		t.Fatalf("cached response header = %q, want 1", got)
	}
}

func TestPathCachedTransportConcurrentAccess(t *testing.T) {
	base := &recordingRoundTripper{}
	transport := &pathCachedTransport{base: base}

	var wg sync.WaitGroup
	errCh := make(chan error, 20)
	for i := 0; i < 20; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				req, err := http.NewRequest(
					http.MethodGet,
					fmt.Sprintf("http://example.test/probe?id=%d", i%5),
					nil,
				)
				if err != nil {
					errCh <- err
					return
				}
				req.Header.Set("X-Mode", fmt.Sprintf("mode-%d", j%3))
				resp, err := transport.RoundTrip(req)
				if err != nil {
					errCh <- err
					return
				}
				_, err = io.ReadAll(resp.Body)
				_ = resp.Body.Close()
				if err != nil {
					errCh <- err
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func mustCachedRequest(t *testing.T, method, rawURL, body, mode string) *http.Request {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, rawURL, reader)
	if err != nil {
		t.Fatal(err)
	}
	if mode != "" {
		req.Header.Set("X-Mode", mode)
	}
	return req
}

func mustRoundTripBody(t *testing.T, rt http.RoundTripper, req *http.Request) string {
	t.Helper()
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

type recordingRoundTripper struct {
	mu    sync.Mutex
	calls int
}

func (rt *recordingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.mu.Lock()
	rt.calls++
	call := rt.calls
	rt.mu.Unlock()

	body := fmt.Sprintf("%s %s %s #%d", req.Method, req.URL.RequestURI(), req.Header.Get("X-Mode"), call)
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Proto:      "HTTP/1.1",
		Header:     http.Header{"X-Call": {fmt.Sprint(call)}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}, nil
}
