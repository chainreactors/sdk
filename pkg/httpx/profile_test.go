package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefaultConfigDoesNotInjectHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ua := r.Header.Get("User-Agent"); ua != "Go-http-client/1.1" && ua != "" {
			t.Errorf("default config injected unexpected UA: %q", ua)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewClient(DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
}

func TestBrowserConfigInjectsHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" || r.Header.Get("Accept") == "" || r.Header.Get("Accept-Language") == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewClient(BrowserConfig())
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestBrowserConfigDoesNotOverrideCallerHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "custom-agent" {
			t.Errorf("caller UA overridden: got %q", r.Header.Get("User-Agent"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewClient(BrowserConfig())
	if err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	req.Header.Set("User-Agent", "custom-agent")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
}

func TestProxyFallback(t *testing.T) {
	client, err := NewClient(DefaultConfig().WithProxy("socks5://invalid-proxy-that-does-not-exist:99999"))
	if err != nil {
		t.Fatalf("expected fallback to succeed, got error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client after proxy fallback")
	}
}

func TestWithBuilders(t *testing.T) {
	cfg := BrowserConfig().WithTimeout(5 * 1e9).WithProxy("socks5://127.0.0.1:1080").WithRedirects(false)
	if cfg.Timeout != 5*1e9 {
		t.Errorf("timeout = %v", cfg.Timeout)
	}
	if len(cfg.Proxy) != 1 || cfg.Proxy[0] != "socks5://127.0.0.1:1080" {
		t.Errorf("proxy = %v", cfg.Proxy)
	}
	if cfg.FollowRedirects {
		t.Error("redirects should be false")
	}
	if len(cfg.Headers) == 0 {
		t.Error("browser headers should be preserved")
	}
}
