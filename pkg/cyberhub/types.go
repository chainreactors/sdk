package cyberhub

import (
	"github.com/chainreactors/fingers/alias"
	"github.com/chainreactors/fingers/fingers"
	"github.com/chainreactors/neutron/templates"
)

// ========================================
// Cyberhub API 响应（简化版 - 匹配后端 ExportFinger）
// ========================================

// FingerprintResponse Cyberhub Export API 返回的指纹数据
// 直接嵌入 fingers.Finger，完全匹配后端的 ExportFinger 结构
type FingerprintResponse struct {
	*fingers.Finger `json:",inline"` // 嵌入所有 Finger 字段
	Alias           *alias.Alias `json:"alias,omitempty"` // Alias 数据
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

// IsActive 检查是否为激活状态（Export API 只返回 active 状态的指纹）
func (r *FingerprintResponse) IsActive() bool {
	return true // Export API 默认只导出 active 状态
}

// GetTemplate 获取 Template 对象（直接返回嵌入的 Template）
func (r *POCResponse) GetTemplate() *templates.Template {
	return r.Template
}
