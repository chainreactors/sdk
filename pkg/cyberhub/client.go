package cyberhub

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
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
}

// NewClient 创建 Cyberhub 客户端
func NewClient(baseURL, apiKey string, timeout time.Duration) *Client {
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
	}
}

// ExportFingerprints 导出所有指纹（使用 export API）
// withFingerprint: 是否返回完整的指纹规则数据
// source: 指纹来源过滤（可选，如 "github", "local" 等）
// filters: 筛选条件（可选，传 nil 表示不筛选）
func (c *Client) ExportFingerprints(ctx context.Context, withFingerprint bool, source string, filters ...*ExportFilter) ([]FingerprintResponse, error) {
	params := url.Values{}
	if withFingerprint {
		params.Set("with_fingerprint", "true")
	}
	if source != "" {
		params.Add("sources", source)
	}

	// 添加筛选参数
	applyFilterParams(params, firstFilter(filters))

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
// filters: 筛选条件（可选，传 nil 表示不筛选）
func (c *Client) ExportPOCs(ctx context.Context, tags []string, severities []string, pocType string, source string, filters ...*ExportFilter) ([]POCResponse, error) {
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
		params.Add("sources", source)
	}

	// 只导出激活状态的 POC
	params.Set("status", "active")

	// 添加筛选参数
	applyFilterParams(params, firstFilter(filters))

	endpoint := fmt.Sprintf("%s/pocs/export?%s", c.baseURL, params.Encode())

	var response POCListResponse
	if err := c.doRequest(ctx, "GET", endpoint, nil, &response); err != nil {
		return nil, fmt.Errorf("export pocs failed: %w", err)
	}

	return response.POCs, nil
}

// ExportPOCsByNames 按名称列表导出 POC
func (c *Client) ExportPOCsByNames(ctx context.Context, names []string) ([]POCResponse, error) {
	if len(names) == 0 {
		return nil, nil
	}

	params := url.Values{}
	for _, name := range names {
		params.Add("names", name)
	}
	params.Set("status", "active")

	endpoint := fmt.Sprintf("%s/pocs/export?%s", c.baseURL, params.Encode())

	var response POCListResponse
	if err := c.doRequest(ctx, "GET", endpoint, nil, &response); err != nil {
		return nil, fmt.Errorf("export pocs by names failed: %w", err)
	}

	return response.POCs, nil
}

// firstFilter 返回第一个非 nil 的筛选器
func firstFilter(filters []*ExportFilter) *ExportFilter {
	for _, filter := range filters {
		if filter != nil {
			return filter
		}
	}
	return nil
}

// applyFilterParams 将筛选条件应用到 URL 参数
func applyFilterParams(params url.Values, filter *ExportFilter) {
	if filter == nil {
		return
	}

	if len(filter.Tags) > 0 {
		existingTags := make(map[string]struct{})
		for _, tag := range params["tags"] {
			if tag == "" {
				continue
			}
			existingTags[tag] = struct{}{}
		}
		for _, tag := range filter.Tags {
			if tag == "" {
				continue
			}
			if _, exists := existingTags[tag]; exists {
				continue
			}
			params.Add("tags", tag)
			existingTags[tag] = struct{}{}
		}
	}

	if len(filter.Sources) > 0 {
		existingSources := make(map[string]struct{})
		for _, source := range params["sources"] {
			if source == "" {
				continue
			}
			existingSources[source] = struct{}{}
		}
		for _, source := range filter.Sources {
			if source == "" {
				continue
			}
			if _, exists := existingSources[source]; exists {
				continue
			}
			params.Add("sources", source)
			existingSources[source] = struct{}{}
		}
	}

	if filter.CreatedAfter != nil {
		params.Set("created_after", filter.CreatedAfter.Format(time.RFC3339))
	}

	if filter.CreatedBefore != nil {
		params.Set("created_before", filter.CreatedBefore.Format(time.RFC3339))
	}

	if filter.UpdatedAfter != nil {
		params.Set("updated_after", filter.UpdatedAfter.Format(time.RFC3339))
	}

	if filter.UpdatedBefore != nil {
		params.Set("updated_before", filter.UpdatedBefore.Format(time.RFC3339))
	}

	if filter.Limit > 0 {
		params.Set("page", "1")
		params.Set("page_size", strconv.Itoa(filter.Limit))
	}
}

type requestBodyProvider struct {
	data    []byte
	hasBody bool
}

func newRequestBodyProvider(body io.Reader) (*requestBodyProvider, error) {
	if body == nil {
		return &requestBodyProvider{hasBody: false}, nil
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("read request body failed: %w", err)
	}
	return &requestBodyProvider{data: data, hasBody: true}, nil
}

func (p *requestBodyProvider) Reader() io.Reader {
	if p == nil || !p.hasBody {
		return nil
	}
	return bytes.NewReader(p.data)
}

func readResponseBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()

	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer gzipReader.Close()
		reader = gzipReader
	}

	return io.ReadAll(reader)
}

// doRequest 执行 HTTP 请求（带重试）
func (c *Client) doRequest(ctx context.Context, method, endpoint string, body io.Reader, result interface{}) error {
	bodyProvider, err := newRequestBodyProvider(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, bodyProvider.Reader())
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}

	// 设置请求头
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	// 后端要求显式声明 gzip 才会返回压缩数据
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}

	bodyBytes, err := readResponseBody(resp)
	if err != nil {
		return fmt.Errorf("read response failed: %w", err)
	}

	// 检查 HTTP 状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// 解析标准响应格式: { code, message, data }
	var apiResp APIResponse
	if err := json.Unmarshal(bodyBytes, &apiResp); err != nil {
		return fmt.Errorf("parse response failed: %w", err)
	}

	// 检查业务状态码
	if apiResp.Code != 0 {
		return fmt.Errorf("api error: code=%d, message=%s", apiResp.Code, apiResp.Message)
	}

	// 解析 data 字段到目标结构
	dataBytes, err := json.Marshal(apiResp.Data)
	if err != nil {
		return fmt.Errorf("marshal data failed: %w", err)
	}

	if err := json.Unmarshal(dataBytes, result); err != nil {
		return fmt.Errorf("unmarshal data failed: %w", err)
	}

	return nil
}

// Close 关闭客户端
func (c *Client) Close() error {
	// HTTP client 不需要显式关闭
	return nil
}
