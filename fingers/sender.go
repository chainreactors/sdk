package fingers

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/chainreactors/proxyclient"
)

// ========================================
// HTTPSender 接口和实现
// ========================================

// HTTPSender HTTP请求发送器接口
type HTTPSender interface {
	Send(url string) (*http.Response, error)
}

// DefaultHTTPSender 默认的HTTP发送器实现
type DefaultHTTPSender struct {
	client  *http.Client
	timeout time.Duration
	proxy   string
}

// NewDefaultHTTPSender 创建默认的HTTPSender
// 参数:
//   - timeout: 超时时间
//   - proxy: 代理地址（可选，如 "socks5://127.0.0.1:1080"）
func NewDefaultHTTPSender(timeout time.Duration, proxy string) *DefaultHTTPSender {
	if timeout <= 0 {
		timeout = 10 * time.Second // 默认10秒超时
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // 跳过证书验证
		},
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	// 如果配置了代理，使用proxyclient
	if proxy != "" {
		proxyURL, err := url.Parse(proxy)
		if err == nil {
			dial, err := proxyclient.NewClient(proxyURL)
			if err == nil {
				transport.DialContext = dial.DialContext
			}
		}
	}

	return &DefaultHTTPSender{
		timeout: timeout,
		proxy:   proxy,
		client: &http.Client{
			Timeout:   timeout,
			Transport: transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // 不自动跟随重定向
			},
		},
	}
}

// Send 发送HTTP请求
func (s *DefaultHTTPSender) Send(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	// 设置默认User-Agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request failed: %w", err)
	}

	return resp, nil
}

// SendWithMethod 使用指定方法发送HTTP请求
func (s *DefaultHTTPSender) SendWithMethod(url, method string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request failed: %w", err)
	}

	return resp, nil
}

// SetTimeout 设置超时时间
func (s *DefaultHTTPSender) SetTimeout(timeout time.Duration) {
	s.timeout = timeout
	s.client.Timeout = timeout
}

// SetClient 设置自定义HTTP客户端
func (s *DefaultHTTPSender) SetClient(client *http.Client) {
	s.client = client
}
