package httpx

import (
	"net/http"
	"strings"
)

const (
	ProfileBrowser = "browser"

	browserUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
)

var browserProfileHeaders = map[string]string{
	"User-Agent":      browserUserAgent,
	"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8",
	"Accept-Language": "zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7",
	"Connection":      "keep-alive",
}

func ApplyBrowserProfileHeaders(header http.Header) {
	if header == nil {
		return
	}
	for key, value := range browserProfileHeaders {
		if strings.TrimSpace(header.Get(key)) == "" {
			header.Set(key, value)
		}
	}
}

type profileTransport struct {
	base http.RoundTripper
}

func (t *profileTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	ApplyBrowserProfileHeaders(req.Header)
	return base.RoundTrip(req)
}

func wrapProfileTransport(client *http.Client) {
	if client == nil {
		return
	}
	base := client.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	if _, ok := base.(*profileTransport); ok {
		return
	}
	client.Transport = &profileTransport{base: base}
}
