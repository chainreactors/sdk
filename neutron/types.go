package neutron

import (
	"github.com/chainreactors/neutron/templates"
	"github.com/chainreactors/sdk/pkg/cyberhub"
)

// ========================================
// Config 配置
// ========================================

// Config Neutron SDK 配置
type Config struct {
	cyberhub.Config

	// 加载配置
	LocalPath string
	Templates []*templates.Template
}
