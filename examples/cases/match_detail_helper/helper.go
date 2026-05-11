// match_detail_helper 演示调用方如何把 SDK 的 MatchResult 转成自己的 DTO。
//
// 核心 matcher detail / match_url 逻辑已经在 sdk/fingers 公共 API 中：
//
//	eng.MatchHTTPWithDetail(resp) -> []fingers.MatchResult
//
// 这一份只保留应用层字段映射，不再让调用方复制 EnableMatchDetail、
// SendData 解析、fallback URL 等 SDK 内部细节。
package main

import (
	"fmt"
	"net/http"

	"github.com/chainreactors/fingers/common"
	sdkfingers "github.com/chainreactors/sdk/fingers"
)

func main() {
	fmt.Println("This example is exercised via `go test ./examples/cases/match_detail_helper`.")
}

// FingerMatch 是给上层的扁平、易序列化结构。
type FingerMatch struct {
	Name         string             `json:"name"`
	Version      string             `json:"version,omitempty"`
	Tags         []string           `json:"tags,omitempty"`
	Attributes   *common.Attributes `json:"attributes,omitempty"`
	MatchURL     string             `json:"match_url,omitempty"`
	MatcherType  string             `json:"matcher_type,omitempty"`
	MatcherValue string             `json:"matcher_value,omitempty"`
	RuleIndex    int                `json:"rule_index"`
	SendData     string             `json:"send_data,omitempty"`
}

// FromMatchResults 把 SDK MatchResult 拍平成调用方自己的结构。
func FromMatchResults(results []sdkfingers.MatchResult) []FingerMatch {
	out := make([]FingerMatch, 0, len(results))
	for _, r := range results {
		out = append(out, FromMatchResult(r))
	}
	return out
}

// FromMatchResult 转换单条 SDK MatchResult。
func FromMatchResult(r sdkfingers.MatchResult) FingerMatch {
	fm := FingerMatch{
		MatchURL:     r.MatchURL,
		MatcherType:  r.MatcherType,
		MatcherValue: r.MatcherValue,
		RuleIndex:    r.RuleIndex,
		SendData:     r.SendData,
	}
	if r.Framework == nil {
		return fm
	}
	fm.Name = r.Framework.Name
	fm.Tags = r.Framework.Tags
	fm.Attributes = r.Framework.Attributes
	if r.Framework.Attributes != nil {
		fm.Version = r.Framework.Attributes.Version
	}
	return fm
}

// DetectFingersDetail 演示单次 HTTP 被动匹配，返回调用方自己的 FingerMatch 列表。
func DetectFingersDetail(target, cyberhubURL, apiKey string) ([]FingerMatch, error) {
	cfg := sdkfingers.NewConfig()
	if cyberhubURL != "" {
		cfg.WithCyberhub(cyberhubURL, apiKey)
	}
	eng, err := sdkfingers.NewEngine(cfg)
	if err != nil {
		return nil, err
	}
	resp, err := http.Get(target)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	results, err := eng.MatchHTTPWithDetail(resp)
	if err != nil {
		return nil, err
	}
	return FromMatchResults(results), nil
}
