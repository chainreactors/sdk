// match_detail_helper 演示一种把 common.MatchDetail 拍平成调用方友好结构
// (FingerMatch) 的写法，覆盖被动 + 主动两种链路的 match_url 兜底。
//
// 关键点和 examples/cases/match_detail 一致：
//  1. NewEngine() 之后必须调用 GetFingersEngine().EnableMatchDetail()。
//  2. common.Framework.MatchDetail 是数据源，FingerMatch 只是 ergonomics 封装。
//  3. match_url 取值优先级：
//       MatchDetail.SendData 中的 "url=" > 当前请求 URL / SprayResult.UrlString
//
// 这一份和 cases/match_detail/main.go 的区别只是封装风格，二选一即可。
// 用 go test ./examples/cases/match_detail_helper 运行演示。
package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/chainreactors/fingers/common"
	sdkfingers "github.com/chainreactors/sdk/fingers"
	"github.com/chainreactors/sdk/spray"
	"github.com/chainreactors/utils/httputils"
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
	RuleIndex    int                `json:"rule_index,omitempty"`
	SendData     string             `json:"send_data,omitempty"`
}

// EnableMatchDetail 翻开底层 fingers 引擎的 matcher 详情开关。
// 必须在 sdkfingers.NewEngine() 之后调用，因为 NewEngine 内部会触发
// engine.Compile()，把每条 finger 的 EnableMatchDetail 重置回 engine
// 字段的默认值 (false)。
func EnableMatchDetail(eng *sdkfingers.Engine) error {
	if eng == nil {
		return nil
	}
	fe, err := eng.GetFingersEngine()
	if err != nil {
		return err
	}
	if fe != nil {
		fe.EnableMatchDetail()
	}
	return nil
}

// FlattenMatches 把 common.Frameworks 拍平成 FingerMatch 切片。
// fallbackURL：MatchDetail.SendData 不含 url= 时回填用 (DetectFingers 传请求 URL,
// spray 传 SprayResult.UrlString)。
func FlattenMatches(frames common.Frameworks, fallbackURL string) []FingerMatch {
	out := make([]FingerMatch, 0, len(frames))
	for _, f := range frames {
		if f == nil {
			continue
		}
		fm := FingerMatch{
			Name:       f.Name,
			Tags:       f.Tags,
			Attributes: f.Attributes,
			MatchURL:   fallbackURL,
		}
		if f.Attributes != nil {
			fm.Version = f.Attributes.Version
		}
		if d := f.MatchDetail; d != nil {
			fm.MatcherType = d.MatcherType
			fm.MatcherValue = d.MatcherValue
			fm.RuleIndex = d.RuleIndex
			fm.SendData = d.SendData
			if u := ExtractURL(d.SendData); u != "" {
				fm.MatchURL = u
			}
		}
		out = append(out, fm)
	}
	return out
}

// ExtractURL 从 "scope=... method=... url=<...>" 中取 url= 后整段。
// 词边界判断避免 value 内出现 url= 子串时误匹配。
func ExtractURL(sendData string) string {
	const tag = "url="
	for start := 0; start < len(sendData); {
		i := strings.Index(sendData[start:], tag)
		if i < 0 {
			return ""
		}
		i += start
		if i == 0 || sendData[i-1] == ' ' {
			return strings.TrimSpace(sendData[i+len(tag):])
		}
		start = i + len(tag)
	}
	return ""
}

// DetectFingersDetail 演示 ① ：单次 HTTP 被动匹配，返回 FingerMatch 列表。
func DetectFingersDetail(target, cyberhubURL, apiKey string) ([]FingerMatch, error) {
	cfg := sdkfingers.NewConfig()
	if cyberhubURL != "" {
		cfg.WithCyberhub(cyberhubURL, apiKey)
	}
	eng, err := sdkfingers.NewEngine(cfg)
	if err != nil {
		return nil, err
	}
	if err := EnableMatchDetail(eng); err != nil {
		return nil, err
	}
	resp, err := http.Get(target)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	fallbackURL := target
	if resp.Request != nil && resp.Request.URL != nil {
		fallbackURL = resp.Request.URL.String()
	}
	frames, err := eng.Get().DetectContent(httputils.ReadRaw(resp))
	if err != nil {
		return nil, err
	}
	return FlattenMatches(frames, fallbackURL), nil
}

// SprayDetailResult spray 命中的资源 + 该资源上识别出的指纹列表。
type SprayDetailResult struct {
	URL     string        `json:"url"`
	Path    string        `json:"path"`
	Status  int           `json:"status"`
	Title   string        `json:"title"`
	Matches []FingerMatch `json:"matches"`
}

// SprayWithCrawlAndFingerDetail 演示 ② ：spray + 静态爬虫 + 指纹联动。
// max <= 0 表示不限；建议给个上限，否则 crawl 在大站会无限扩张。
func SprayWithCrawlAndFingerDetail(target string, seeds []string, depth, max int, cyberhubURL, apiKey string) ([]SprayDetailResult, error) {
	if depth >= 3 {
		depth = 2
	}
	if len(seeds) == 0 {
		seeds = []string{""}
	}
	cfg := spray.NewConfig()
	if cyberhubURL != "" && apiKey != "" {
		fEng, err := sdkfingers.NewEngine(sdkfingers.NewConfig().WithCyberhub(cyberhubURL, apiKey))
		if err != nil {
			return nil, err
		}
		if err := EnableMatchDetail(fEng); err != nil {
			return nil, err
		}
		if fEng.Get() != nil {
			cfg = cfg.WithFingersEngine(fEng)
		}
	}
	se := spray.NewEngine(cfg)
	if err := se.Init(); err != nil {
		return nil, err
	}
	defer se.Close()

	innerCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx := spray.NewContext().
		WithContext(innerCtx).
		SetThreads(50).
		SetTimeout(10).
		SetFinger(true).
		SetCrawlPlugin(true).
		SetCrawlDepth(depth)

	ch, err := se.Execute(ctx, spray.NewBruteTask(target, seeds))
	if err != nil {
		return nil, err
	}

	var out []SprayDetailResult
	for r := range ch {
		sr, ok := r.(*spray.Result)
		if !ok || !sr.Success() {
			continue
		}
		data := sr.SprayResult()
		if data == nil || len(data.Frameworks) == 0 {
			continue
		}
		out = append(out, SprayDetailResult{
			URL:     data.UrlString,
			Path:    data.Path,
			Status:  data.Status,
			Title:   data.Title,
			Matches: FlattenMatches(data.Frameworks, data.UrlString),
		})
		if max > 0 && len(out) >= max {
			cancel()
			break
		}
	}
	return out, nil
}
