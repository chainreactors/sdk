package cyberhub

import (
	"time"

	"github.com/chainreactors/fingers/alias"
	"github.com/chainreactors/fingers/fingers"
	"github.com/chainreactors/neutron/templates"
)

// ========================================
// 导出筛选器
// ========================================

// ExportFilter 通用导出筛选选项
type ExportFilter struct {
	// 标签筛选（多个标签为 OR 关系）
	Tags []string

	// 来源筛选（多个来源为 OR 关系）
	Sources []string

	// 生命周期状态筛选（多个状态为 OR 关系，POC / 指纹导出均会透传）
	// POC 留空时 SDK 默认仅导出 active 状态，保持向后兼容；
	// 指纹留空时走后端默认语义。
	// POC 显式指定后默认 active 行为被覆盖，可用于加载待审核 / 草稿 / 未启用规则。
	// 合法值：active / pending / draft / inactive / deprecated。
	Statuses []string

	// 审核流程状态筛选（POC / 指纹导出均会透传）。
	// 对 POC 而言，显式指定后默认 active 行为被覆盖。
	// 留空表示不按审核状态过滤；合法值：pending / approved / rejected / draft / none。
	ReviewStatus string

	// 是否拉取待审核草稿内容（POC / 指纹导出均会透传，对应后端 ?with_draft=true）。
	//
	// 默认 false 时后端返回 RawContent（已生效内容）；置 true 时优先返回
	// RawContentDraft（待审核草稿），这是把"全新待审核行"和"编辑型 pending
	// 行"的实际新内容取回来的唯一途径。注意此标志与 Statuses / ReviewStatus
	// 正交：filter 决定"返回哪些行"，Draft 决定"读哪一列"，需要同时设置。
	//
	// 仅 FingerprintHub 引擎的指纹会在 raw_content 中返回 draft；其他引擎指
	// 纹 raw_content 字段固定为空。
	Draft bool

	// 时间范围筛选
	CreatedAfter  *time.Time // 创建时间起始
	CreatedBefore *time.Time // 创建时间截止
	UpdatedAfter  *time.Time // 更新时间起始
	UpdatedBefore *time.Time // 更新时间截止

	// 数量限制
	Limit int // 最大返回数量（映射为 page=1&page_size=limit）
}

// NewExportFilter 创建空的筛选器
func NewExportFilter() *ExportFilter {
	return &ExportFilter{}
}

// WithTags 设置标签筛选
func (f *ExportFilter) WithTags(tags ...string) *ExportFilter {
	f.Tags = tags
	return f
}

// WithSources 设置来源筛选
func (f *ExportFilter) WithSources(sources ...string) *ExportFilter {
	f.Sources = sources
	return f
}

// WithStatuses 设置生命周期状态筛选（POC / 指纹导出均会透传）。
// POC 调用此方法将覆盖 SDK 默认仅导出 active 的行为，可用于加载待审核 / 草稿 / 未启用规则。
// 合法值：active / pending / draft / inactive / deprecated。
func (f *ExportFilter) WithStatuses(statuses ...string) *ExportFilter {
	f.Statuses = statuses
	return f
}

// WithReviewStatus 设置审核流程状态筛选（POC / 指纹导出均会透传）。
// POC 调用此方法将覆盖 SDK 默认仅导出 active 的行为。
// 留空表示不过滤；合法值：pending / approved / rejected / draft / none。
func (f *ExportFilter) WithReviewStatus(status string) *ExportFilter {
	f.ReviewStatus = status
	return f
}

// WithDraft 控制是否拉取待审核草稿内容（对应后端 ?with_draft=true）。
//
// 默认 false 时后端返回 RawContent（已生效内容）；置 true 时优先返回
// RawContentDraft。要拉到"全新待审核 POC / 指纹"或"编辑型 pending"的实
// 际新内容，必须显式调用 WithDraft(true) —— 此标志与 Statuses /
// ReviewStatus 解耦，filter 决定"返回哪些行"，Draft 决定"读哪一列"。
//
// 仅 FingerprintHub 引擎的指纹会在 raw_content 字段返回 draft 内容。
func (f *ExportFilter) WithDraft(draft bool) *ExportFilter {
	f.Draft = draft
	return f
}

// WithCreatedAfter 设置创建时间起始
func (f *ExportFilter) WithCreatedAfter(t time.Time) *ExportFilter {
	f.CreatedAfter = &t
	return f
}

// WithCreatedBefore 设置创建时间截止
func (f *ExportFilter) WithCreatedBefore(t time.Time) *ExportFilter {
	f.CreatedBefore = &t
	return f
}

// WithUpdatedAfter 设置更新时间起始
func (f *ExportFilter) WithUpdatedAfter(t time.Time) *ExportFilter {
	f.UpdatedAfter = &t
	return f
}

// WithUpdatedBefore 设置更新时间截止
func (f *ExportFilter) WithUpdatedBefore(t time.Time) *ExportFilter {
	f.UpdatedBefore = &t
	return f
}

// WithLimit 设置数量限制
func (f *ExportFilter) WithLimit(limit int) *ExportFilter {
	f.Limit = limit
	return f
}

// ========================================
// Cyberhub API 响应（简化版 - 匹配后端 ExportFinger）
// ========================================

// FingerprintResponse Cyberhub Export API 返回的指纹数据
// 直接嵌入 fingers.Finger，完全匹配后端的 ExportFinger 结构
type FingerprintResponse struct {
	*fingers.Finger `json:",inline" yaml:",inline"` // 嵌入所有 Finger 字段
	Alias           *alias.Alias     `json:"alias,omitempty" yaml:"alias,omitempty"` // Alias 数据
}

// FingerprintListResponse Cyberhub Export API 列表响应
type FingerprintListResponse struct {
	Fingerprints []FingerprintResponse `json:"fingerprints"`
	Total        int                   `json:"total"`
	Page         int                   `json:"page"`
	PageSize     int                   `json:"page_size"`
}

// POCResponse Cyberhub Export API 返回的 POC 数据
// 直接嵌入 templates.Template，完全匹配后端的 ExportPOC 结构
type POCResponse struct {
	*templates.Template `json:",inline" yaml:",inline"` // 嵌入所有 Template 字段
}

// POCListResponse Cyberhub Export API POC 列表响应
type POCListResponse struct {
	POCs     []POCResponse `json:"pocs"`
	Total    int           `json:"total"`
	Exported int           `json:"exported"`
}

// APIResponse Cyberhub 标准响应格式
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// ========================================
// 辅助方法
// ========================================

// GetFinger 获取 Finger 对象（直接返回嵌入的 Finger）
func (r *FingerprintResponse) GetFinger() *fingers.Finger {
	return r.Finger
}

// GetAlias 获取 Alias 对象
func (r *FingerprintResponse) GetAlias() *alias.Alias {
	return r.Alias
}

// 注：Cyberhub 后端的 Fingerprint.Status（active/pending/draft/inactive 等）
// 不在 export payload 里序列化，因此 FingerprintResponse 无法仅凭返回数据判断
// "是否激活"。如需按状态筛选，使用 ExportFilter.WithStatuses(...) 在请求侧过滤。
//
// Deprecated: ExportFingerprints 的响应体不包含 Fingerprint.Status，返回值只能表示旧版
// SDK 的历史假设，不能用于判断显式状态筛选后的真实生命周期状态。
func (r *FingerprintResponse) IsActive() bool {
	return true
}

// GetTemplate 获取 Template 对象（直接返回嵌入的 Template）
func (r *POCResponse) GetTemplate() *templates.Template {
	return r.Template
}
