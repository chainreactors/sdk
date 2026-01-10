package neutron

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/chainreactors/neutron/templates"
)

// ========================================
// POC 筛选器
// ========================================

// AdvancedFilter 高级筛选条件（与后端保持一致）
type AdvancedFilter struct {
	ID       string      `json:"id,omitempty"`
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// POCFilter POC筛选器
type POCFilter struct {
	// 基础筛选条件
	Keyword    string   // 关键词（匹配ID、名称、描述）
	Tags       []string // 标签列表（OR关系）
	Severities []string // 严重程度列表（OR关系）
	Type       string   // POC类型
	Sources    []string // 来源列表
	Status     string   // POC状态（远程筛选）
	Statuses   []string // 多个POC状态（远程筛选）
	SourceIDs  []uint   // 源ID列表（远程筛选）

	// 高级筛选条件
	CVSSScoreMin      *float64         // 最小CVSS评分
	CVSSScoreMax      *float64         // 最大CVSS评分
	EPSSScoreMin      *float64         // 最小EPSS评分
	EPSSScoreMax      *float64         // 最大EPSS评分
	EPSSPercentileMin *float64         // 最小EPSS百分位
	EPSSPercentileMax *float64         // 最大EPSS百分位
	CVEID             string           // CVE ID（支持模糊匹配）
	CWEID             string           // CWE ID（支持模糊匹配）
	CPE               string           // CPE（支持模糊匹配）
	Authors           []string         // 作者列表
	Vendors           []string         // 厂商列表（metadata.vendor）
	Products          []string         // 产品列表（metadata.product）
	AdvancedFilters   []AdvancedFilter // 高级筛选（支持操作符）

	// 关联筛选
	HasFingers        *bool    // 是否有关联指纹
	AssociatedFingers []string // 关联的指纹名称列表（AND关系）
}

// NewPOCFilter 创建POC筛选器
func NewPOCFilter() *POCFilter {
	return &POCFilter{}
}

// ========================================
// Builder 模式方法
// ========================================

// WithKeyword 设置关键词
func (f *POCFilter) WithKeyword(keyword string) *POCFilter {
	f.Keyword = keyword
	return f
}

// WithTags 设置标签
func (f *POCFilter) WithTags(tags ...string) *POCFilter {
	f.Tags = tags
	return f
}

// WithSeverities 设置严重程度
func (f *POCFilter) WithSeverities(severities ...string) *POCFilter {
	f.Severities = severities
	return f
}

// WithType 设置类型
func (f *POCFilter) WithType(pocType string) *POCFilter {
	f.Type = pocType
	return f
}

// WithSources 设置来源
func (f *POCFilter) WithSources(sources ...string) *POCFilter {
	f.Sources = sources
	return f
}

// WithStatus 设置状态筛选（远程筛选）
func (f *POCFilter) WithStatus(status string) *POCFilter {
	f.Status = status
	return f
}

// WithStatuses 设置多个状态筛选（远程筛选）
func (f *POCFilter) WithStatuses(statuses ...string) *POCFilter {
	f.Statuses = statuses
	return f
}

// WithSourceIDs 设置源ID筛选（远程筛选）
func (f *POCFilter) WithSourceIDs(ids ...uint) *POCFilter {
	f.SourceIDs = ids
	return f
}

// WithCVSSScoreRange 设置CVSS评分范围
func (f *POCFilter) WithCVSSScoreRange(min, max float64) *POCFilter {
	f.CVSSScoreMin = &min
	f.CVSSScoreMax = &max
	return f
}

// WithCVSSScoreMin 设置最小CVSS评分
func (f *POCFilter) WithCVSSScoreMin(min float64) *POCFilter {
	f.CVSSScoreMin = &min
	return f
}

// WithCVSSScoreMax 设置最大CVSS评分
func (f *POCFilter) WithCVSSScoreMax(max float64) *POCFilter {
	f.CVSSScoreMax = &max
	return f
}

// WithEPSSScoreRange 设置EPSS评分范围
func (f *POCFilter) WithEPSSScoreRange(min, max float64) *POCFilter {
	f.EPSSScoreMin = &min
	f.EPSSScoreMax = &max
	return f
}

// WithEPSSScoreMin 设置最小EPSS评分
func (f *POCFilter) WithEPSSScoreMin(min float64) *POCFilter {
	f.EPSSScoreMin = &min
	return f
}

// WithEPSSScoreMax 设置最大EPSS评分
func (f *POCFilter) WithEPSSScoreMax(max float64) *POCFilter {
	f.EPSSScoreMax = &max
	return f
}

// WithEPSSPercentileRange 设置EPSS百分位范围
func (f *POCFilter) WithEPSSPercentileRange(min, max float64) *POCFilter {
	f.EPSSPercentileMin = &min
	f.EPSSPercentileMax = &max
	return f
}

// WithEPSSPercentileMin 设置最小EPSS百分位
func (f *POCFilter) WithEPSSPercentileMin(min float64) *POCFilter {
	f.EPSSPercentileMin = &min
	return f
}

// WithEPSSPercentileMax 设置最大EPSS百分位
func (f *POCFilter) WithEPSSPercentileMax(max float64) *POCFilter {
	f.EPSSPercentileMax = &max
	return f
}

// WithCVEID 设置CVE ID筛选
func (f *POCFilter) WithCVEID(cveID string) *POCFilter {
	f.CVEID = cveID
	return f
}

// WithCWEID 设置CWE ID筛选
func (f *POCFilter) WithCWEID(cweID string) *POCFilter {
	f.CWEID = cweID
	return f
}

// WithCPE 设置CPE筛选
func (f *POCFilter) WithCPE(cpe string) *POCFilter {
	f.CPE = cpe
	return f
}

// WithAuthors 设置作者筛选
func (f *POCFilter) WithAuthors(authors ...string) *POCFilter {
	f.Authors = authors
	return f
}

// WithVendors 设置厂商筛选（metadata.vendor）
func (f *POCFilter) WithVendors(vendors ...string) *POCFilter {
	f.Vendors = vendors
	return f
}

// WithProducts 设置产品筛选（metadata.product）
func (f *POCFilter) WithProducts(products ...string) *POCFilter {
	f.Products = products
	return f
}

// WithHasFingers 设置是否有关联指纹
func (f *POCFilter) WithHasFingers(hasFingers bool) *POCFilter {
	f.HasFingers = &hasFingers
	return f
}

// WithAssociatedFingers 设置关联的指纹名称
func (f *POCFilter) WithAssociatedFingers(fingers ...string) *POCFilter {
	f.AssociatedFingers = fingers
	return f
}

// WithAdvancedFilters 设置高级筛选条件
func (f *POCFilter) WithAdvancedFilters(filters ...AdvancedFilter) *POCFilter {
	f.AdvancedFilters = filters
	return f
}

// ========================================
// 筛选方法
// ========================================

// Apply 应用筛选器到POC列表
func (f *POCFilter) Apply(tpls []*templates.Template) []*templates.Template {
	if f.isLocalEmpty() {
		return tpls
	}

	var result []*templates.Template
	for _, t := range tpls {
		if f.match(t) {
			result = append(result, t)
		}
	}
	return result
}

// isEmpty 检查筛选器是否为空
func (f *POCFilter) isLocalEmpty() bool {
	return f.Keyword == "" &&
		len(f.Tags) == 0 &&
		len(f.Severities) == 0 &&
		f.Type == "" &&
		len(f.Sources) == 0 &&
		f.CVSSScoreMin == nil &&
		f.CVSSScoreMax == nil &&
		f.EPSSScoreMin == nil &&
		f.EPSSScoreMax == nil &&
		f.EPSSPercentileMin == nil &&
		f.EPSSPercentileMax == nil &&
		f.CVEID == "" &&
		f.CWEID == "" &&
		f.CPE == "" &&
		len(f.Authors) == 0 &&
		len(f.Vendors) == 0 &&
		len(f.Products) == 0 &&
		len(f.AdvancedFilters) == 0 &&
		f.HasFingers == nil &&
		len(f.AssociatedFingers) == 0
}

// HasRemoteCriteria 判断是否包含需要远程筛选的条件
func (f *POCFilter) HasRemoteCriteria() bool {
	return f.Status != "" || len(f.Statuses) > 0 || len(f.SourceIDs) > 0 || len(f.Sources) > 0 || f.Type != ""
}

// match 判断单个POC是否匹配筛选条件
func (f *POCFilter) match(t *templates.Template) bool {
	// 关键词匹配
	if f.Keyword != "" {
		keyword := strings.ToLower(f.Keyword)
		if !strings.Contains(strings.ToLower(t.Id), keyword) &&
			!strings.Contains(strings.ToLower(t.Info.Name), keyword) &&
			!strings.Contains(strings.ToLower(t.Info.Description), keyword) {
			return false
		}
	}

	// 标签匹配（OR关系）
	if len(f.Tags) > 0 && !f.matchTags(t) {
		return false
	}

	// 严重程度匹配（OR关系）
	if len(f.Severities) > 0 && !f.matchSeverity(t) {
		return false
	}

	// 类型匹配（metadata.type）
	if f.Type != "" && !f.matchType(t) {
		return false
	}

	// 来源匹配（metadata.source/source_name）
	if len(f.Sources) > 0 && !f.matchSource(t) {
		return false
	}

	// 作者匹配
	if len(f.Authors) > 0 && !f.matchAuthor(t) {
		return false
	}

	// 厂商匹配
	if len(f.Vendors) > 0 && !f.matchVendor(t) {
		return false
	}

	// 产品匹配
	if len(f.Products) > 0 && !f.matchProduct(t) {
		return false
	}

	// CVSS评分范围匹配
	if !f.matchCVSSScore(t) {
		return false
	}

	// EPSS评分范围匹配
	if !f.matchEPSSScore(t) {
		return false
	}

	// EPSS百分位范围匹配
	if !f.matchEPSSPercentile(t) {
		return false
	}

	// CVE ID匹配
	if f.CVEID != "" && !f.matchCVEID(t) {
		return false
	}

	// CWE ID匹配
	if f.CWEID != "" && !f.matchCWEID(t) {
		return false
	}

	// CPE匹配
	if f.CPE != "" && !f.matchCPE(t) {
		return false
	}

	// 关联指纹匹配
	if f.HasFingers != nil {
		hasFingers := len(t.Fingers) > 0
		if *f.HasFingers != hasFingers {
			return false
		}
	}

	// 指定关联指纹匹配（AND关系）
	if len(f.AssociatedFingers) > 0 && !f.matchAssociatedFingers(t) {
		return false
	}

	// 高级筛选
	if len(f.AdvancedFilters) > 0 && !f.matchAdvancedFilters(t) {
		return false
	}

	return true
}

// matchTags 标签匹配
func (f *POCFilter) matchTags(t *templates.Template) bool {
	templateTags := strings.Split(t.Info.Tags, ",")
	for _, filterTag := range f.Tags {
		for _, templateTag := range templateTags {
			if strings.EqualFold(strings.TrimSpace(filterTag), strings.TrimSpace(templateTag)) {
				return true
			}
		}
	}
	return false
}

// matchSeverity 严重程度匹配
func (f *POCFilter) matchSeverity(t *templates.Template) bool {
	for _, sev := range f.Severities {
		if strings.EqualFold(sev, t.Info.Severity) {
			return true
		}
	}
	return false
}

// matchType 类型匹配（metadata.type）
func (f *POCFilter) matchType(t *templates.Template) bool {
	metaType := metadataString(t.Info.Metadata, "type", "poc_type", "template_type")
	if metaType == "" {
		return true
	}
	return containsFold(metaType, f.Type)
}

// matchSource 来源匹配（metadata.source/source_name）
func (f *POCFilter) matchSource(t *templates.Template) bool {
	sources := metadataStrings(t.Info.Metadata, "source", "source_name")
	if len(sources) == 0 {
		return false
	}
	return anyContainsFold(sources, f.Sources)
}

// matchAuthor 作者匹配（OR关系）
func (f *POCFilter) matchAuthor(t *templates.Template) bool {
	if t.Info.Author == "" {
		return false
	}
	authors := splitCSV(t.Info.Author)
	return anyContainsFold(authors, f.Authors)
}

// matchVendor 厂商匹配（metadata.vendor）
func (f *POCFilter) matchVendor(t *templates.Template) bool {
	vendors := metadataStrings(t.Info.Metadata, "vendor")
	if len(vendors) == 0 {
		return false
	}
	return anyContainsFold(vendors, f.Vendors)
}

// matchProduct 产品匹配（metadata.product）
func (f *POCFilter) matchProduct(t *templates.Template) bool {
	products := metadataStrings(t.Info.Metadata, "product")
	if len(products) == 0 {
		return false
	}
	return anyContainsFold(products, f.Products)
}

// matchCVSSScore CVSS评分匹配
func (f *POCFilter) matchCVSSScore(t *templates.Template) bool {
	if f.CVSSScoreMin == nil && f.CVSSScoreMax == nil {
		return true
	}

	if t.Info.Classification == nil {
		return false
	}

	score := t.Info.Classification.CVSSScore
	if f.CVSSScoreMin != nil && score < *f.CVSSScoreMin {
		return false
	}
	if f.CVSSScoreMax != nil && score > *f.CVSSScoreMax {
		return false
	}
	return true
}

// matchEPSSScore EPSS评分匹配
func (f *POCFilter) matchEPSSScore(t *templates.Template) bool {
	if f.EPSSScoreMin == nil && f.EPSSScoreMax == nil {
		return true
	}

	if t.Info.Classification == nil {
		return false
	}

	score := t.Info.Classification.EPSSScore
	if f.EPSSScoreMin != nil && score < *f.EPSSScoreMin {
		return false
	}
	if f.EPSSScoreMax != nil && score > *f.EPSSScoreMax {
		return false
	}
	return true
}

// matchEPSSPercentile EPSS百分位匹配
func (f *POCFilter) matchEPSSPercentile(t *templates.Template) bool {
	if f.EPSSPercentileMin == nil && f.EPSSPercentileMax == nil {
		return true
	}
	if t.Info.Classification == nil {
		return false
	}

	score := t.Info.Classification.EPSSPercentile
	if f.EPSSPercentileMin != nil && score < *f.EPSSPercentileMin {
		return false
	}
	if f.EPSSPercentileMax != nil && score > *f.EPSSPercentileMax {
		return false
	}
	return true
}

// matchCVEID CVE ID匹配（支持模糊匹配）
func (f *POCFilter) matchCVEID(t *templates.Template) bool {
	if t.Info.Classification == nil || t.Info.Classification.CVEID == "" {
		return false
	}
	return strings.Contains(
		strings.ToUpper(t.Info.Classification.CVEID),
		strings.ToUpper(f.CVEID),
	)
}

// matchCWEID CWE ID匹配（支持模糊匹配）
func (f *POCFilter) matchCWEID(t *templates.Template) bool {
	if t.Info.Classification == nil || t.Info.Classification.CWEID == "" {
		return false
	}
	return strings.Contains(
		strings.ToUpper(t.Info.Classification.CWEID),
		strings.ToUpper(f.CWEID),
	)
}

// matchCPE CPE匹配（支持模糊匹配）
func (f *POCFilter) matchCPE(t *templates.Template) bool {
	if t.Info.Classification == nil || t.Info.Classification.CPE == "" {
		return false
	}
	return strings.Contains(
		strings.ToLower(t.Info.Classification.CPE),
		strings.ToLower(f.CPE),
	)
}

// matchAssociatedFingers 关联指纹匹配（AND关系）
func (f *POCFilter) matchAssociatedFingers(t *templates.Template) bool {
	for _, requiredFinger := range f.AssociatedFingers {
		found := false
		for _, templateFinger := range t.Fingers {
			if strings.EqualFold(requiredFinger, templateFinger) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (f *POCFilter) matchAdvancedFilters(t *templates.Template) bool {
	for _, advFilter := range f.AdvancedFilters {
		if !matchAdvancedFilter(t, advFilter) {
			return false
		}
	}
	return true
}

func matchAdvancedFilter(t *templates.Template, adv AdvancedFilter) bool {
	field := strings.ToLower(strings.TrimSpace(adv.Field))
	operator := strings.ToLower(strings.TrimSpace(adv.Operator))

	switch field {
	case "cvss_score":
		score := 0.0
		if t.Info.Classification == nil {
			return false
		}
		score = t.Info.Classification.CVSSScore
		return matchNumeric(score, operator, adv.Value)
	case "epss_score":
		if t.Info.Classification == nil {
			return false
		}
		return matchNumeric(t.Info.Classification.EPSSScore, operator, adv.Value)
	case "epss_percentile":
		if t.Info.Classification == nil {
			return false
		}
		return matchNumeric(t.Info.Classification.EPSSPercentile, operator, adv.Value)
	case "cve_id":
		if t.Info.Classification == nil {
			return false
		}
		return matchString(t.Info.Classification.CVEID, operator, adv.Value)
	case "cwe_id":
		if t.Info.Classification == nil {
			return false
		}
		return matchString(t.Info.Classification.CWEID, operator, adv.Value)
	case "cpe":
		if t.Info.Classification == nil {
			return false
		}
		return matchString(t.Info.Classification.CPE, operator, adv.Value)
	case "author":
		return matchString(t.Info.Author, operator, adv.Value)
	case "tags":
		return matchTagsValue(t.Info.Tags, operator, adv.Value)
	case "vendor":
		vendors := metadataStrings(t.Info.Metadata, "vendor")
		return matchStringList(vendors, operator, adv.Value)
	case "product":
		products := metadataStrings(t.Info.Metadata, "product")
		return matchStringList(products, operator, adv.Value)
	default:
		return true
	}
}

func matchNumeric(value float64, operator string, raw interface{}) bool {
	if operator == "" {
		operator = "gte"
	}

	switch operator {
	case "gte":
		if v, ok := toFloat64(raw); ok {
			return value >= v
		}
	case "lte":
		if v, ok := toFloat64(raw); ok {
			return value <= v
		}
	case "eq":
		if v, ok := toFloat64(raw); ok {
			return value == v
		}
	case "between":
		if min, max, ok := toFloatRange(raw); ok {
			return value >= min && value <= max
		}
	}
	return false
}

func matchString(target string, operator string, raw interface{}) bool {
	return matchStringList([]string{target}, operator, raw)
}

func matchStringList(targets []string, operator string, raw interface{}) bool {
	values := toStringSlice(raw)
	if len(values) == 0 {
		return false
	}
	if operator == "" {
		operator = "contains"
	}

	switch operator {
	case "eq":
		return anyEqualFold(targets, values)
	case "contains":
		return anyContainsFold(targets, values)
	default:
		return false
	}
}

func matchTagsValue(tags string, operator string, raw interface{}) bool {
	return matchStringList(splitCSV(tags), operator, raw)
}

func toFloat64(raw interface{}) (float64, bool) {
	switch val := raw.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case uint:
		return float64(val), true
	case uint64:
		return float64(val), true
	case string:
		parsed, err := strconv.ParseFloat(val, 64)
		if err == nil {
			return parsed, true
		}
	case json.Number:
		parsed, err := val.Float64()
		if err == nil {
			return parsed, true
		}
	}
	return 0, false
}

func toFloatRange(raw interface{}) (float64, float64, bool) {
	switch val := raw.(type) {
	case []float64:
		if len(val) >= 2 {
			return val[0], val[1], true
		}
	case []interface{}:
		if len(val) >= 2 {
			min, okMin := toFloat64(val[0])
			max, okMax := toFloat64(val[1])
			if okMin && okMax {
				return min, max, true
			}
		}
	case []string:
		if len(val) >= 2 {
			min, okMin := toFloat64(val[0])
			max, okMax := toFloat64(val[1])
			if okMin && okMax {
				return min, max, true
			}
		}
	}
	return 0, 0, false
}

func toStringSlice(raw interface{}) []string {
	switch val := raw.(type) {
	case string:
		if val == "" {
			return nil
		}
		return splitCSV(val)
	case []string:
		return val
	case []interface{}:
		var result []string
		for _, item := range val {
			if str, ok := item.(string); ok && str != "" {
				result = append(result, str)
			}
		}
		return result
	default:
		return nil
	}
}

func metadataString(meta map[string]interface{}, keys ...string) string {
	values := metadataStrings(meta, keys...)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func metadataStrings(meta map[string]interface{}, keys ...string) []string {
	if meta == nil {
		return nil
	}
	for _, key := range keys {
		value, ok := meta[key]
		if !ok || value == nil {
			continue
		}
		if strings, ok := value.([]string); ok {
			return strings
		}
		return toStringSlice(value)
	}
	return nil
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	var result []string
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			result = append(result, part)
		}
	}
	return result
}

func anyContainsFold(values []string, filters []string) bool {
	for _, filter := range filters {
		for _, value := range values {
			if containsFold(value, filter) {
				return true
			}
		}
	}
	return false
}

func anyEqualFold(values []string, filters []string) bool {
	for _, filter := range filters {
		for _, value := range values {
			if strings.EqualFold(value, filter) {
				return true
			}
		}
	}
	return false
}

func containsFold(value, filter string) bool {
	return strings.Contains(strings.ToLower(value), strings.ToLower(filter))
}
