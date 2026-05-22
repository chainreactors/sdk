package cyberhub

import (
	"time"
)

// ExportFilter 通用导出筛选选项
type ExportFilter struct {
	Names        []string // 按名称过滤
	Tags         []string // 标签筛选（多个标签为 OR 关系）
	Sources      []string // 来源筛选（多个来源为 OR 关系）
	Severities   []string // 严重程度过滤
	POCType      string   // POC 类型过滤
	Statuses     []string // 生命周期状态过滤
	ReviewStatus string   // 审核状态过滤

	CreatedAfter  *time.Time // 创建时间起始
	CreatedBefore *time.Time // 创建时间截止
	UpdatedAfter  *time.Time // 更新时间起始
	UpdatedBefore *time.Time // 更新时间截止

	Limit int // 最大返回数量（映射为 page=1&page_size=limit）
}

// NewExportFilter 创建空的筛选器
func NewExportFilter() *ExportFilter {
	return &ExportFilter{}
}

func (f *ExportFilter) WithTags(tags ...string) *ExportFilter {
	f.Tags = tags
	return f
}

func (f *ExportFilter) WithSources(sources ...string) *ExportFilter {
	f.Sources = sources
	return f
}

func (f *ExportFilter) WithCreatedAfter(t time.Time) *ExportFilter {
	f.CreatedAfter = &t
	return f
}

func (f *ExportFilter) WithCreatedBefore(t time.Time) *ExportFilter {
	f.CreatedBefore = &t
	return f
}

func (f *ExportFilter) WithUpdatedAfter(t time.Time) *ExportFilter {
	f.UpdatedAfter = &t
	return f
}

func (f *ExportFilter) WithUpdatedBefore(t time.Time) *ExportFilter {
	f.UpdatedBefore = &t
	return f
}

func (f *ExportFilter) WithLimit(limit int) *ExportFilter {
	f.Limit = limit
	return f
}

func (f *ExportFilter) WithNames(names ...string) *ExportFilter {
	f.Names = names
	return f
}

func (f *ExportFilter) WithSeverities(severities ...string) *ExportFilter {
	f.Severities = severities
	return f
}

func (f *ExportFilter) WithPOCType(pocType string) *ExportFilter {
	f.POCType = pocType
	return f
}

func (f *ExportFilter) WithStatuses(statuses ...string) *ExportFilter {
	f.Statuses = statuses
	return f
}

func (f *ExportFilter) WithReviewStatus(status string) *ExportFilter {
	f.ReviewStatus = status
	return f
}
