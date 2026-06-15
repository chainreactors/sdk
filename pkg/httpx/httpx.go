// Package httpx is the SDK-layer HTTP client generator that turns proxy specs
// and profile presets into concurrency-safe *http.Client instances built on the
// shared, zero-global utils/httpx foundation.
package httpx

import (
	"net/http"
	"time"

	utilshttpx "github.com/chainreactors/utils/httpx"

	"github.com/chainreactors/sdk/pkg/types"
)

// Config describes the parameters for constructing an SDK HTTP client.
type Config struct {
	Timeout             time.Duration
	Proxy               []string
	FollowRedirects     bool
	InsecureSkipVerify  bool
	MaxIdleConns        int
	MaxIdleConnsPerHost int
	IdleConnTimeout     time.Duration
	DisableKeepAlives   bool
	Headers             map[string]string
}

// DefaultConfig returns a general-purpose preset: no redirects, skip TLS
// verification, sensible connection pool defaults.
func DefaultConfig() Config {
	return Config{
		Timeout:             10 * time.Second,
		FollowRedirects:     false,
		InsecureSkipVerify:  true,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}
}

// BrowserConfig returns a preset that mimics a mainstream browser: redirects
// enabled, browser-grade UA/Accept/Accept-Language headers injected on every
// request.
func BrowserConfig() Config {
	return Config{
		Timeout:             10 * time.Second,
		FollowRedirects:     true,
		InsecureSkipVerify:  true,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		Headers: map[string]string{
			"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36",
			"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8",
			"Accept-Language": "en-US,en;q=0.9",
			"Connection":      "keep-alive",
		},
	}
}

func (c Config) WithTimeout(d time.Duration) Config {
	c.Timeout = d
	return c
}

func (c Config) WithProxy(proxy ...string) Config {
	c.Proxy = proxy
	return c
}

func (c Config) WithRedirects(follow bool) Config {
	c.FollowRedirects = follow
	return c
}

func (c Config) WithHeaders(headers map[string]string) Config {
	c.Headers = headers
	return c
}

// DefaultClient constructs a new *http.Client with DefaultConfig.
func DefaultClient() *http.Client {
	c, _ := NewClient(DefaultConfig())
	return c
}

// NewClient constructs a new *http.Client from cfg. If Proxy is set but fails
// to resolve, the client is built without proxy (best-effort fallback).
func NewClient(cfg Config) (*http.Client, error) {
	client, err := newClientInner(cfg)
	if err != nil && len(cfg.Proxy) > 0 {
		cfg.Proxy = nil
		client, _ = newClientInner(cfg)
		err = nil
	}
	if client != nil && len(cfg.Headers) > 0 {
		client.Transport = &headerTransport{
			base:    client.Transport,
			headers: cfg.Headers,
		}
	}
	return client, err
}

func newClientInner(cfg Config) (*http.Client, error) {
	uc := utilshttpx.ClientConfig{
		Timeout:             cfg.Timeout,
		FollowRedirects:     cfg.FollowRedirects,
		InsecureSkipVerify:  cfg.InsecureSkipVerify,
		MaxIdleConns:        cfg.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.MaxIdleConnsPerHost,
		IdleConnTimeout:     cfg.IdleConnTimeout,
		DisableKeepAlives:   cfg.DisableKeepAlives,
	}
	if len(cfg.Proxy) > 0 {
		dialer, err := types.NewProxyDialer(cfg.Proxy)
		if err != nil {
			return nil, err
		}
		if dialer != nil {
			uc.DialContext = dialer.DialContext
		}
	}
	return utilshttpx.NewHTTPClient(uc), nil
}

// SetDefaultHeaders applies BrowserConfig headers to an http.Header, without
// overriding keys the caller has already set. Use this for requests sent
// through a raw *http.Client that was not created via NewClient.
func SetDefaultHeaders(header http.Header) {
	if header == nil {
		return
	}
	for key, value := range BrowserConfig().Headers {
		if header.Get(key) == "" {
			header.Set(key, value)
		}
	}
}

// headerTransport injects default headers into every outgoing request without
// overriding headers the caller has already set.
type headerTransport struct {
	base    http.RoundTripper
	headers map[string]string
}

func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for key, value := range t.headers {
		if req.Header.Get(key) == "" {
			req.Header.Set(key, value)
		}
	}
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}
