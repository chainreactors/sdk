package association

import (
	"sync"

	"github.com/chainreactors/neutron/templates"
)

// ========================================
// 指纹-POC 关联索引
// ========================================

// FingerPOCIndex 指纹-POC关联索引
// 用于快速查询指纹和POC之间的关联关系
type FingerPOCIndex struct {
	// fingerToPOCs: fingerName -> []pocID
	fingerToPOCs map[string][]string
	// pocToFingers: pocID -> []fingerName
	pocToFingers map[string][]string
	// fingerHasPOC: fingerName -> bool (用于快速判断)
	fingerHasPOC map[string]bool

	mu sync.RWMutex
}

// NewFingerPOCIndex 创建关联索引
func NewFingerPOCIndex() *FingerPOCIndex {
	return &FingerPOCIndex{
		fingerToPOCs: make(map[string][]string),
		pocToFingers: make(map[string][]string),
		fingerHasPOC: make(map[string]bool),
	}
}

// BuildFromTemplates 从POC模板列表构建索引
func (idx *FingerPOCIndex) BuildFromTemplates(templates []*templates.Template) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// 清空现有索引
	idx.fingerToPOCs = make(map[string][]string)
	idx.pocToFingers = make(map[string][]string)
	idx.fingerHasPOC = make(map[string]bool)

	for _, t := range templates {
		if t == nil {
			continue
		}
		pocID := t.Id
		for _, fingerName := range t.Fingers {
			// 添加到 fingerToPOCs
			idx.fingerToPOCs[fingerName] = append(idx.fingerToPOCs[fingerName], pocID)
			// 添加到 pocToFingers
			idx.pocToFingers[pocID] = append(idx.pocToFingers[pocID], fingerName)
			// 标记指纹有关联POC
			idx.fingerHasPOC[fingerName] = true
		}
	}
}

// GetPOCsByFinger 获取指纹关联的POC列表
func (idx *FingerPOCIndex) GetPOCsByFinger(fingerName string) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.fingerToPOCs[fingerName]
}

// GetFingersByPOC 获取POC关联的指纹列表
func (idx *FingerPOCIndex) GetFingersByPOC(pocID string) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.pocToFingers[pocID]
}

// HasAssociatedPOC 判断指纹是否有关联POC
func (idx *FingerPOCIndex) HasAssociatedPOC(fingerName string) bool {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.fingerHasPOC[fingerName]
}

// GetFingerHasPOCMap 获取指纹是否有POC的映射（用于批量筛选）
func (idx *FingerPOCIndex) GetFingerHasPOCMap() map[string]bool {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	result := make(map[string]bool, len(idx.fingerHasPOC))
	for k, v := range idx.fingerHasPOC {
		result[k] = v
	}
	return result
}

// GetAllFingerNames 获取所有有关联POC的指纹名称
func (idx *FingerPOCIndex) GetAllFingerNames() []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	names := make([]string, 0, len(idx.fingerToPOCs))
	for name := range idx.fingerToPOCs {
		names = append(names, name)
	}
	return names
}

// GetAllPOCIDs 获取所有有关联指纹的POC ID
func (idx *FingerPOCIndex) GetAllPOCIDs() []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	ids := make([]string, 0, len(idx.pocToFingers))
	for id := range idx.pocToFingers {
		ids = append(ids, id)
	}
	return ids
}

// Count 获取索引统计
func (idx *FingerPOCIndex) Count() (fingerCount, pocCount int) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.fingerToPOCs), len(idx.pocToFingers)
}

// GetPOCCountByFinger 获取指纹关联的POC数量
func (idx *FingerPOCIndex) GetPOCCountByFinger(fingerName string) int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.fingerToPOCs[fingerName])
}

// GetFingerCountByPOC 获取POC关联的指纹数量
func (idx *FingerPOCIndex) GetFingerCountByPOC(pocID string) int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.pocToFingers[pocID])
}

// Clear 清空索引
func (idx *FingerPOCIndex) Clear() {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.fingerToPOCs = make(map[string][]string)
	idx.pocToFingers = make(map[string][]string)
	idx.fingerHasPOC = make(map[string]bool)
}
