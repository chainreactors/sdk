package fingers

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/chainreactors/fingers/alias"
	fingersEngine "github.com/chainreactors/fingers/fingers"
)

// ========================================
// 侵入性类型定义
// ========================================

// InvasivenessType 侵入性类型
type InvasivenessType string

const (
	// InvasivenessAll 不筛选侵入性
	InvasivenessAll InvasivenessType = ""
	// InvasivenessInvasive 侵入性（扫描非根路径）
	InvasivenessInvasive InvasivenessType = "invasive"
	// InvasivenessNonInvasive 非侵入性（仅扫描根路径或被动探测）
	InvasivenessNonInvasive InvasivenessType = "non_invasive"
)

// ========================================
// 指纹筛选器
// ========================================

// FingerprintFilter 指纹筛选器
type FingerprintFilter struct {
	// 基础筛选条件
	Keyword    string   // 关键词（匹配名称、描述）
	Protocol   string   // 协议类型 (http, tcp)
	Tags       []string // 标签列表（OR关系）
	Categories []string // 分类列表
	Vendor     string   // 厂商
	Product    string   // 产品
	Authors    []string // 作者列表
	Sources    []string // 来源列表
	Status     string   // 状态（远程筛选）
	Statuses   []string // 状态列表（远程筛选）

	// 新增筛选条件
	Invasiveness     InvasivenessType // 侵入性
	HasAssociatedPOC *bool            // 是否有关联POC

	// 关联索引（用于判断是否有关联POC）
	pocFingerIndex map[string]bool // fingerName -> hasPOC
	aliasIndex     map[string]*alias.Alias
	sourceIndex    map[string]string // fingerName -> source
}

// NewFingerprintFilter 创建指纹筛选器
func NewFingerprintFilter() *FingerprintFilter {
	return &FingerprintFilter{}
}

// ========================================
// Builder 模式方法
// ========================================

// WithKeyword 设置关键词
func (f *FingerprintFilter) WithKeyword(keyword string) *FingerprintFilter {
	f.Keyword = keyword
	return f
}

// WithProtocol 设置协议
func (f *FingerprintFilter) WithProtocol(protocol string) *FingerprintFilter {
	f.Protocol = protocol
	return f
}

// WithTags 设置标签
func (f *FingerprintFilter) WithTags(tags ...string) *FingerprintFilter {
	f.Tags = tags
	return f
}

// WithCategories 设置分类
func (f *FingerprintFilter) WithCategories(categories ...string) *FingerprintFilter {
	f.Categories = categories
	return f
}

// WithVendor 设置厂商
func (f *FingerprintFilter) WithVendor(vendor string) *FingerprintFilter {
	f.Vendor = vendor
	return f
}

// WithProduct 设置产品
func (f *FingerprintFilter) WithProduct(product string) *FingerprintFilter {
	f.Product = product
	return f
}

// WithAuthors 设置作者
func (f *FingerprintFilter) WithAuthors(authors ...string) *FingerprintFilter {
	f.Authors = authors
	return f
}

// WithSources 设置来源
func (f *FingerprintFilter) WithSources(sources ...string) *FingerprintFilter {
	f.Sources = sources
	return f
}

// WithStatus 设置状态筛选（远程筛选）
func (f *FingerprintFilter) WithStatus(status string) *FingerprintFilter {
	f.Status = status
	return f
}

// WithStatuses 设置状态列表筛选（远程筛选）
func (f *FingerprintFilter) WithStatuses(statuses ...string) *FingerprintFilter {
	f.Statuses = statuses
	return f
}

// WithInvasiveness 设置侵入性筛选
func (f *FingerprintFilter) WithInvasiveness(invasiveness InvasivenessType) *FingerprintFilter {
	f.Invasiveness = invasiveness
	return f
}

// WithHasAssociatedPOC 设置是否有关联POC
func (f *FingerprintFilter) WithHasAssociatedPOC(hasPOC bool) *FingerprintFilter {
	f.HasAssociatedPOC = &hasPOC
	return f
}

// SetPOCAssociationIndex 设置POC关联索引
func (f *FingerprintFilter) SetPOCAssociationIndex(index map[string]bool) *FingerprintFilter {
	f.pocFingerIndex = index
	return f
}

// SetAliasIndex 设置 Alias 索引
func (f *FingerprintFilter) SetAliasIndex(index map[string]*alias.Alias) *FingerprintFilter {
	f.aliasIndex = index
	return f
}

// SetSourceIndex 设置来源索引（仅当外部提供来源数据时有效）
func (f *FingerprintFilter) SetSourceIndex(index map[string]string) *FingerprintFilter {
	f.sourceIndex = index
	return f
}

// ========================================
// 筛选方法
// ========================================

// Apply 应用筛选器到指纹列表
func (f *FingerprintFilter) Apply(fingers fingersEngine.Fingers) fingersEngine.Fingers {
	if f.isLocalEmpty() {
		return fingers
	}

	var result fingersEngine.Fingers
	for _, finger := range fingers {
		if f.match(finger) {
			result = append(result, finger)
		}
	}
	return result
}

// isEmpty 检查筛选器是否为空
func (f *FingerprintFilter) isLocalEmpty() bool {
	return f.Keyword == "" &&
		f.Protocol == "" &&
		len(f.Tags) == 0 &&
		len(f.Categories) == 0 &&
		f.Vendor == "" &&
		f.Product == "" &&
		len(f.Authors) == 0 &&
		(len(f.Sources) == 0 || f.sourceIndex == nil) &&
		f.Invasiveness == "" &&
		f.HasAssociatedPOC == nil
}

// HasRemoteCriteria 判断是否包含需要远程筛选的条件
func (f *FingerprintFilter) HasRemoteCriteria() bool {
	return f.Status != "" || len(f.Statuses) > 0 || len(f.Sources) > 0
}

// match 判断单个指纹是否匹配筛选条件
func (f *FingerprintFilter) match(finger *fingersEngine.Finger) bool {
	// 关键词匹配
	if f.Keyword != "" {
		keyword := strings.ToLower(f.Keyword)
		if !strings.Contains(strings.ToLower(finger.Name), keyword) &&
			!strings.Contains(strings.ToLower(finger.Description), keyword) {
			return false
		}
	}

	// 协议匹配
	if f.Protocol != "" && !strings.EqualFold(finger.Protocol, f.Protocol) {
		return false
	}

	// 标签匹配（OR关系）
	if len(f.Tags) > 0 && !f.matchTags(finger.Tags) {
		return false
	}

	// 分类匹配（基于 Alias.category/type）
	if len(f.Categories) > 0 && !f.matchCategories(finger) {
		return false
	}

	// 厂商匹配
	if f.Vendor != "" && !f.matchVendor(finger) {
		return false
	}

	// 产品匹配
	if f.Product != "" && !f.matchProduct(finger) {
		return false
	}

	// 作者匹配
	if len(f.Authors) > 0 && !containsIgnoreCase(f.Authors, finger.Author) {
		return false
	}

	// 来源匹配（需要外部提供来源索引）
	if len(f.Sources) > 0 && f.sourceIndex != nil {
		source := f.sourceIndex[finger.Name]
		if source == "" || !sliceContainsIgnoreCase(f.Sources, source) {
			return false
		}
	}

	// 侵入性匹配
	if f.Invasiveness != "" && !f.matchInvasiveness(finger) {
		return false
	}

	// 关联POC匹配
	if f.HasAssociatedPOC != nil && f.pocFingerIndex != nil {
		hasPOC := f.pocFingerIndex[finger.Name]
		if *f.HasAssociatedPOC != hasPOC {
			return false
		}
	}

	return true
}

// matchTags 标签匹配（OR关系）
func (f *FingerprintFilter) matchTags(fingerTags []string) bool {
	for _, filterTag := range f.Tags {
		for _, fingerTag := range fingerTags {
			if strings.EqualFold(filterTag, fingerTag) {
				return true
			}
		}
	}
	return false
}

// matchCategories 分类匹配（Alias.category/type）
func (f *FingerprintFilter) matchCategories(finger *fingersEngine.Finger) bool {
	if f.aliasIndex == nil {
		return false
	}
	aliasData := f.aliasIndex[finger.Name]
	if aliasData == nil {
		return false
	}
	for _, category := range f.Categories {
		if strings.EqualFold(category, aliasData.Category) || strings.EqualFold(category, aliasData.Type) {
			return true
		}
	}
	return false
}

// matchVendor 厂商匹配（优先 Alias.vendor）
func (f *FingerprintFilter) matchVendor(finger *fingersEngine.Finger) bool {
	if f.Vendor == "" {
		return true
	}
	vendor := ""
	if f.aliasIndex != nil {
		if aliasData := f.aliasIndex[finger.Name]; aliasData != nil && aliasData.Vendor != "" {
			vendor = aliasData.Vendor
		}
	}
	if vendor == "" {
		vendor = finger.Attributes.Vendor
	}
	return containsFold(vendor, f.Vendor)
}

// matchProduct 产品匹配（优先 Alias.product）
func (f *FingerprintFilter) matchProduct(finger *fingersEngine.Finger) bool {
	if f.Product == "" {
		return true
	}
	product := ""
	if f.aliasIndex != nil {
		if aliasData := f.aliasIndex[finger.Name]; aliasData != nil && aliasData.Product != "" {
			product = aliasData.Product
		}
	}
	if product == "" {
		product = finger.Attributes.Product
	}
	return containsFold(product, f.Product)
}

// matchInvasiveness 判断侵入性
// 侵入性判断逻辑：分析Rule.SendDataStr中的HTTP请求路径
// 如果路径为"/"或为空，则为非侵入性
// 如果路径包含其他路径（如/admin、/config），则为侵入性
func (f *FingerprintFilter) matchInvasiveness(finger *fingersEngine.Finger) bool {
	isInvasive := AnalyzeInvasiveness(finger)

	if f.Invasiveness == InvasivenessInvasive {
		return isInvasive
	}
	if f.Invasiveness == InvasivenessNonInvasive {
		return !isInvasive
	}
	return true
}

// ========================================
// 侵入性分析函数
// ========================================

// AnalyzeInvasiveness 分析指纹的侵入性
// 返回 true 表示侵入性（扫描非根路径），false 表示非侵入性
func AnalyzeInvasiveness(finger *fingersEngine.Finger) bool {
	for _, rule := range finger.Rules {
		if rule.SendDataStr == "" {
			continue // 被动探测，非侵入性
		}

		// 解析SendDataStr中的HTTP路径
		path := ExtractHTTPPath(rule.SendDataStr)
		if path != "" && path != "/" {
			return true // 包含非根路径，为侵入性
		}
	}
	return false // 默认为非侵入性
}

// ExtractHTTPPath 从HTTP请求数据中提取路径
// SendDataStr格式示例：GET /admin HTTP/1.1\r\nHost: {{Hostname}}\r\n\r\n
func ExtractHTTPPath(sendData string) string {
	// 匹配 HTTP 请求行中的路径
	// 格式: METHOD PATH HTTP/1.x
	re := regexp.MustCompile(`^(GET|POST|PUT|DELETE|HEAD|OPTIONS|PATCH)\s+(\S+)\s+HTTP`)
	matches := re.FindStringSubmatch(sendData)
	if len(matches) >= 3 {
		pathStr := matches[2]
		// 解析路径（可能包含查询参数）
		if u, err := url.Parse(pathStr); err == nil {
			return u.Path
		}
		return pathStr
	}
	return ""
}

// ========================================
// 辅助函数
// ========================================

// containsIgnoreCase 忽略大小写检查切片是否包含字符串
func containsIgnoreCase(slice []string, s string) bool {
	for _, item := range slice {
		if containsFold(s, item) {
			return true
		}
	}
	return false
}

func sliceContainsIgnoreCase(slice []string, s string) bool {
	for _, item := range slice {
		if strings.EqualFold(item, s) {
			return true
		}
	}
	return false
}

func containsFold(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
