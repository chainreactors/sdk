package fingers

import (
	"net/http"
	"strings"

	"github.com/chainreactors/fingers/common"
	"github.com/chainreactors/utils/httputils"
)

// MatchResult is a flattened, easy-to-consume representation of a single
// fingerprint hit. It exposes the matcher metadata and the URL the match
// was attributed to, while keeping the original *common.Framework pointer
// for callers that need the full payload (attributes, tags, etc.).
type MatchResult struct {
	// Framework is the original framework returned by the underlying engine.
	Framework *common.Framework `json:"framework"`

	// MatchURL is the URL the match was attributed to. Resolution order:
	//   1. MatchDetail.SendData's "url=" segment (populated by active
	//      probing flows that supply an explicit probe URL).
	//   2. The fallback URL passed to MatchWithDetail / inferred from
	//      resp.Request.URL by MatchHTTPWithDetail (the common case for
	//      passive matching, which doesn't itself emit a URL).
	MatchURL string `json:"match_url,omitempty"`

	// Convenience copies of MatchDetail. Empty when matcher detail has not
	// been enabled or the engine did not produce matcher information.
	MatcherType  string `json:"matcher_type,omitempty"`
	MatcherValue string `json:"matcher_value,omitempty"`
	RuleIndex    int    `json:"rule_index"`
	SendData     string `json:"send_data,omitempty"`
}

// EnableMatchDetail toggles matcher detail collection on the underlying
// fingers engine. Idempotent and cheap on subsequent calls.
//
// MatchWithDetail / MatchHTTPWithDetail invoke this automatically on first
// use; call it explicitly if you also want plain Match() / MatchHTTP() to
// fill MatchDetail on the returned *common.Framework.
//
// Background: NewEngine internally runs Compile() on the loaded fingers,
// which resets each finger's per-rule EnableMatchDetail flag back to the
// engine default (false). EnableMatchDetail flips both the engine flag
// and every per-finger flag, ensuring matcher metadata is collected on
// subsequent matches.
func (e *Engine) EnableMatchDetail() {
	if e == nil || e.engine == nil {
		return
	}
	fe, err := e.GetFingersEngine()
	if err != nil || fe == nil || fe.MatchDetailEnabled {
		return
	}
	fe.EnableMatchDetail()
}

// MatchWithDetail is the detail-aware counterpart of Match. It runs the
// same passive content match and returns []MatchResult with matcher
// metadata flattened from common.Framework.MatchDetail.
//
// fallbackURL is used as MatchResult.MatchURL when MatchDetail.SendData
// does not contain a "url=" segment (the common case for passive
// matching). Pass the URL of the request whose response body you are
// matching; if you only have a raw byte slice with no associated URL,
// pass "".
func (e *Engine) MatchWithDetail(data []byte, fallbackURL string) ([]MatchResult, error) {
	if e == nil || e.engine == nil {
		return nil, nil
	}
	e.EnableMatchDetail()
	frames, err := e.engine.DetectContent(data)
	if err != nil {
		return nil, err
	}
	return flattenMatchResults(frames, fallbackURL), nil
}

// MatchHTTPWithDetail is the detail-aware counterpart of MatchHTTP. It
// auto-enables matcher detail and returns []MatchResult, using
// resp.Request.URL.String() as the MatchURL fallback when MatchDetail
// does not carry an explicit probe URL.
func (e *Engine) MatchHTTPWithDetail(resp *http.Response) ([]MatchResult, error) {
	if e == nil || e.engine == nil {
		return nil, nil
	}
	e.EnableMatchDetail()

	fallbackURL := ""
	if resp != nil && resp.Request != nil && resp.Request.URL != nil {
		fallbackURL = resp.Request.URL.String()
	}
	var data []byte
	if resp != nil {
		data = httputils.ReadRaw(resp)
	}
	frames, err := e.engine.DetectContent(data)
	if err != nil {
		return nil, err
	}
	return flattenMatchResults(frames, fallbackURL), nil
}

func flattenMatchResults(frames common.Frameworks, fallbackURL string) []MatchResult {
	out := make([]MatchResult, 0, len(frames))
	for _, f := range frames {
		if f == nil {
			continue
		}
		r := MatchResult{Framework: f, MatchURL: fallbackURL}
		if d := f.MatchDetail; d != nil {
			r.MatcherType = d.MatcherType
			r.MatcherValue = d.MatcherValue
			r.RuleIndex = d.RuleIndex
			r.SendData = d.SendData
			if u := extractMatchURL(d.SendData); u != "" {
				r.MatchURL = u
			}
		}
		out = append(out, r)
	}
	return out
}

// extractMatchURL extracts the value of the "url=" segment from a SendData
// string formatted as "scope=... method=... url=<URL>". The URL value may
// contain '=' characters (query params), so we take everything from "url="
// to end-of-string after locating the segment at a word boundary.
func extractMatchURL(sendData string) string {
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
