package cyberhub

import (
	"net/url"
	"strconv"
	"time"
)

// Query 链式查询构建器
type Query struct {
	params url.Values
}

// NewQuery 创建新的查询构建器
func NewQuery() *Query {
	return &Query{params: url.Values{}}
}

// Filter 添加过滤条件（多值为 OR 关系）
func (q *Query) Filter(key string, values ...string) *Query {
	for _, v := range values {
		if v != "" {
			q.params.Add(key, v)
		}
	}
	return q
}

// Set 设置单值参数（会覆盖已有值）
func (q *Query) Set(key, value string) *Query {
	if value != "" {
		q.params.Set(key, value)
	}
	return q
}

// Keyword 设置关键词搜索
func (q *Query) Keyword(keyword string) *Query {
	return q.Set("keyword", keyword)
}

// Tags 设置标签过滤
func (q *Query) Tags(tags ...string) *Query {
	return q.Filter("tags", tags...)
}

// Severities 设置严重程度过滤
func (q *Query) Severities(severities ...string) *Query {
	return q.Filter("severities", severities...)
}

// Type 设置类型过滤
func (q *Query) Type(pocType string) *Query {
	return q.Set("type", pocType)
}

// Source 设置来源过滤
func (q *Query) Source(source string) *Query {
	return q.Set("source", source)
}

// Status 设置状态过滤
func (q *Query) Status(status string) *Query {
	return q.Set("status", status)
}

// Names 设置名称列表过滤
func (q *Query) Names(names ...string) *Query {
	return q.Filter("names", names...)
}

// Limit 设置返回数量限制
func (q *Query) Limit(n int) *Query {
	if n > 0 {
		q.params.Set("limit", strconv.Itoa(n))
	}
	return q
}

// Page 设置分页
func (q *Query) Page(page, pageSize int) *Query {
	if page > 0 {
		q.params.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		q.params.Set("page_size", strconv.Itoa(pageSize))
	}
	return q
}

// CreatedAfter 设置创建时间起始
func (q *Query) CreatedAfter(t time.Time) *Query {
	q.params.Set("created_after", t.Format(time.RFC3339))
	return q
}

// CreatedBefore 设置创建时间截止
func (q *Query) CreatedBefore(t time.Time) *Query {
	q.params.Set("created_before", t.Format(time.RFC3339))
	return q
}

// UpdatedAfter 设置更新时间起始
func (q *Query) UpdatedAfter(t time.Time) *Query {
	q.params.Set("updated_after", t.Format(time.RFC3339))
	return q
}

// UpdatedBefore 设置更新时间截止
func (q *Query) UpdatedBefore(t time.Time) *Query {
	q.params.Set("updated_before", t.Format(time.RFC3339))
	return q
}

// WithFingerprint 设置是否返回完整指纹规则
func (q *Query) WithFingerprint(with bool) *Query {
	if with {
		q.params.Set("with_fingerprint", "true")
	}
	return q
}

// Encode 编码为 URL 查询字符串
func (q *Query) Encode() string {
	return q.params.Encode()
}

// Values 返回底层的 url.Values
func (q *Query) Values() url.Values {
	return q.params
}

// Merge 合并另一个 Query 的参数
func (q *Query) Merge(other *Query) *Query {
	if other == nil {
		return q
	}
	for key, values := range other.params {
		for _, v := range values {
			q.params.Add(key, v)
		}
	}
	return q
}

// Clone 克隆当前查询
func (q *Query) Clone() *Query {
	newParams := url.Values{}
	for key, values := range q.params {
		newParams[key] = append([]string{}, values...)
	}
	return &Query{params: newParams}
}
