package cyberhub

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ========================================
// Cyberhub API 客户端
// ========================================

// Client Cyberhub API 客户端
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	maxRetries int
}

// NewClient 创建 Cyberhub 客户端
func NewClient(baseURL, apiKey string, timeout time.Duration, maxRetries int) *Client {
	// 确保 baseURL 以 /api/v1 结尾
	baseURL = strings.TrimSuffix(baseURL, "/")
	if !strings.HasSuffix(baseURL, "/api/v1") {
		baseURL = baseURL + "/api/v1"
	}

	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		maxRetries: maxRetries,
	}
}

// ExportFingerprints 导出所有指纹（使用 export API）
// withFingerprint: 是否返回完整的指纹规则数据
// source: 指纹来源过滤（可选，如 "github", "local" 等）
func (c *Client) ExportFingerprints(ctx context.Context, withFingerprint bool, source string) ([]FingerprintResponse, error) {
	params := url.Values{}
	if withFingerprint {
		params.Set("with_fingerprint", "true")
	}
	if source != "" {
		params.Set("source", source)
	}

	endpoint := fmt.Sprintf("%s/fingerprints/export?%s", c.baseURL, params.Encode())

	var response FingerprintListResponse
	if err := c.doRequest(ctx, "GET", endpoint, nil, &response); err != nil {
		return nil, fmt.Errorf("export fingerprints failed: %w", err)
	}

	return response.Fingerprints, nil
}

// ExportPOCs 导出所有 POC（使用 export API）
// tags: 标签过滤（可选）
// severities: 严重程度过滤（可选）
// pocType: POC 类型过滤（可选）
// source: POC 来源过滤（可选，如 "github", "local" 等）
func (c *Client) ExportPOCs(ctx context.Context, tags []string, severities []string, pocType string, source string) ([]POCResponse, error) {
	params := url.Values{}

	// 添加标签过滤
	for _, tag := range tags {
		params.Add("tags", tag)
	}

	// 添加严重程度过滤
	for _, severity := range severities {
		params.Add("severities", severity)
	}

	// 添加类型过滤
	if pocType != "" {
		params.Set("type", pocType)
	}

	// 添加来源过滤
	if source != "" {
		params.Set("source", source)
	}

	// 只导出激活状态的 POC
	params.Set("status", "active")

	endpoint := fmt.Sprintf("%s/pocs/export?%s", c.baseURL, params.Encode())

	var response POCListResponse
	if err := c.doRequest(ctx, "GET", endpoint, nil, &response); err != nil {
		return nil, fmt.Errorf("export pocs failed: %w", err)
	}

	return response.POCs, nil
}

// doRequest 执行 HTTP 请求（带重试）
func (c *Client) doRequest(ctx context.Context, method, endpoint string, body io.Reader, result interface{}) error {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// 重试前等待
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * time.Second):
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
		if err != nil {
			return fmt.Errorf("create request failed: %w", err)
		}

		// 设置请求头
		req.Header.Set("X-API-Key", c.apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("http request failed: %w", err)
			continue
		}

		defer resp.Body.Close()

		// 处理 gzip 压缩的响应
		var reader io.Reader = resp.Body
		if resp.Header.Get("Content-Encoding") == "gzip" {
			gzipReader, err := gzip.NewReader(resp.Body)
			if err != nil {
				lastErr = fmt.Errorf("create gzip reader failed: %w", err)
				continue
			}
			defer gzipReader.Close()
			reader = gzipReader
		}

		// 读取响应体
		bodyBytes, err := io.ReadAll(reader)
		if err != nil {
			lastErr = fmt.Errorf("read response failed: %w", err)
			continue
		}

		// 检查 HTTP 状态码
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("http %d: %s", resp.StatusCode, string(bodyBytes))
			// 401 Unauthorized - 认证失败，不重试
			if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
				return lastErr
			}
			continue
		}

		// 解析标准响应格式: { code, message, data }
		var apiResp APIResponse
		if err := json.Unmarshal(bodyBytes, &apiResp); err != nil {
			lastErr = fmt.Errorf("parse response failed: %w", err)
			continue
		}

		// 检查业务状态码
		if apiResp.Code != 0 {
			lastErr = fmt.Errorf("api error: code=%d, message=%s", apiResp.Code, apiResp.Message)
			continue
		}

		// 解析 data 字段到目标结构
		dataBytes, err := json.Marshal(apiResp.Data)
		if err != nil {
			lastErr = fmt.Errorf("marshal data failed: %w", err)
			continue
		}

		if err := json.Unmarshal(dataBytes, result); err != nil {
			lastErr = fmt.Errorf("unmarshal data failed: %w", err)
			continue
		}

		return nil
	}

	return fmt.Errorf("request failed after %d attempts: %w", c.maxRetries+1, lastErr)
}

// Close 关闭客户端
func (c *Client) Close() error {
	// HTTP client 不需要显式关闭
	return nil
}
