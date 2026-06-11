package neutron

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/antchfx/htmlquery"
	"github.com/antchfx/xmlquery"
	"github.com/chainreactors/neutron/common"
	"github.com/chainreactors/neutron/operators"
	"github.com/itchyny/gojq"
)

func init() {
	operators.RegisterExtractorType("json", operators.JSONExtractor, compileJSONExtractor, extractJSON)
	operators.RegisterExtractorType("xpath", operators.XPathExtractor, nil, extractXPath)
	operators.RegisterMatcherType("json", operators.JSONMatcher, compileJSONMatcher, matchJSON)
	operators.RegisterMatcherType("xpath", operators.XPathMatcher, nil, matchXPath)
}

func compileJSONExtractor(e *operators.Extractor) error {
	var compiled []*gojq.Code
	for _, query := range e.JSON {
		parsed, err := gojq.Parse(query)
		if err != nil {
			return fmt.Errorf("could not parse json: %s", query)
		}
		code, err := gojq.Compile(parsed)
		if err != nil {
			return fmt.Errorf("could not compile json: %s", query)
		}
		compiled = append(compiled, code)
	}
	e.SetCompiledData(compiled)
	return nil
}

func extractJSON(e *operators.Extractor, corpus string, _ map[string]interface{}) map[string]struct{} {
	results := make(map[string]struct{})
	var jsonObj interface{}
	if err := json.Unmarshal([]byte(corpus), &jsonObj); err != nil {
		return results
	}
	compiled, ok := e.GetCompiledData().([]*gojq.Code)
	if !ok {
		return results
	}
	for _, k := range compiled {
		iter := k.Run(jsonObj)
		for {
			v, ok := iter.Next()
			if !ok {
				break
			}
			if _, ok := v.(error); ok {
				break
			}
			var result string
			if res, err := common.JSONScalarToString(v); err == nil {
				result = res
			} else if res, err := json.Marshal(v); err == nil {
				result = string(res)
			} else {
				result = common.ToString(v)
			}
			results[result] = struct{}{}
		}
	}
	return results
}

func extractXPath(e *operators.Extractor, corpus string, _ map[string]interface{}) map[string]struct{} {
	if strings.HasPrefix(corpus, "<?xml") {
		return extractXML(e, corpus)
	}
	return extractHTML(e, corpus)
}

func extractHTML(e *operators.Extractor, corpus string) map[string]struct{} {
	results := make(map[string]struct{})
	doc, err := htmlquery.Parse(strings.NewReader(corpus))
	if err != nil {
		return results
	}
	for _, k := range e.XPath {
		nodes, err := htmlquery.QueryAll(doc, k)
		if err != nil {
			continue
		}
		for _, node := range nodes {
			var value string
			if e.Attribute != "" {
				value = htmlquery.SelectAttr(node, e.Attribute)
			} else {
				value = htmlquery.InnerText(node)
			}
			results[value] = struct{}{}
		}
	}
	return results
}

func extractXML(e *operators.Extractor, corpus string) map[string]struct{} {
	results := make(map[string]struct{})
	doc, err := xmlquery.Parse(strings.NewReader(corpus))
	if err != nil {
		return results
	}
	for _, k := range e.XPath {
		nodes, err := xmlquery.QueryAll(doc, k)
		if err != nil {
			continue
		}
		for _, node := range nodes {
			var value string
			if e.Attribute != "" {
				value = node.SelectAttr(e.Attribute)
			} else {
				value = node.InnerText()
			}
			results[value] = struct{}{}
		}
	}
	return results
}

func compileJSONMatcher(m *operators.Matcher) error {
	var compiled []*gojq.Code
	for _, query := range m.JSON {
		parsed, err := gojq.Parse(query)
		if err != nil {
			return fmt.Errorf("could not parse json matcher: %s", query)
		}
		code, err := gojq.Compile(parsed)
		if err != nil {
			return fmt.Errorf("could not compile json matcher: %s", query)
		}
		compiled = append(compiled, code)
	}
	m.SetCompiledData(compiled)
	return nil
}

func matchJSON(m *operators.Matcher, corpus string, _ map[string]interface{}) (bool, []string) {
	var jsonObj interface{}
	if err := json.Unmarshal([]byte(corpus), &jsonObj); err != nil {
		return false, nil
	}
	compiled, ok := m.GetCompiledData().([]*gojq.Code)
	if !ok {
		return false, nil
	}
	var matchedItems []string
	for i, code := range compiled {
		iter := code.Run(jsonObj)
		v, ok := iter.Next()
		if !ok {
			if m.GetCondition() == operators.ANDCondition {
				return false, nil
			}
			continue
		}
		if _, isErr := v.(error); isErr {
			if m.GetCondition() == operators.ANDCondition {
				return false, nil
			}
			continue
		}
		if !isJQTruthy(v) {
			if m.GetCondition() == operators.ANDCondition {
				return false, nil
			}
			continue
		}
		result := common.ToString(v)
		matchedItems = append(matchedItems, result)
		if m.GetCondition() == operators.ORCondition && !m.MatchAll {
			return true, matchedItems
		}
		if len(compiled)-1 == i && !m.MatchAll {
			return true, matchedItems
		}
	}
	if len(matchedItems) > 0 && m.MatchAll {
		return true, matchedItems
	}
	return false, nil
}

func isJQTruthy(v interface{}) bool {
	switch val := v.(type) {
	case nil:
		return false
	case bool:
		return val
	case float64:
		return val != 0
	case string:
		return val != ""
	default:
		return true
	}
}

func matchXPath(m *operators.Matcher, corpus string, _ map[string]interface{}) (bool, []string) {
	if strings.HasPrefix(corpus, "<?xml") {
		return matchXPathXML(m, corpus)
	}
	return matchXPathHTML(m, corpus)
}

func matchXPathHTML(m *operators.Matcher, corpus string) (bool, []string) {
	doc, err := htmlquery.Parse(strings.NewReader(corpus))
	if err != nil {
		return false, nil
	}
	var matchedItems []string
	for i, xpath := range m.XPath {
		nodes, err := htmlquery.QueryAll(doc, xpath)
		if err != nil || len(nodes) == 0 {
			if m.GetCondition() == operators.ANDCondition {
				return false, nil
			}
			continue
		}
		for _, node := range nodes {
			matchedItems = append(matchedItems, htmlquery.InnerText(node))
		}
		if m.GetCondition() == operators.ORCondition && !m.MatchAll {
			return true, matchedItems
		}
		if len(m.XPath)-1 == i && !m.MatchAll {
			return true, matchedItems
		}
	}
	if len(matchedItems) > 0 && m.MatchAll {
		return true, matchedItems
	}
	return false, nil
}

func matchXPathXML(m *operators.Matcher, corpus string) (bool, []string) {
	doc, err := xmlquery.Parse(strings.NewReader(corpus))
	if err != nil {
		return false, nil
	}
	var matchedItems []string
	for i, xpath := range m.XPath {
		nodes, err := xmlquery.QueryAll(doc, xpath)
		if err != nil || len(nodes) == 0 {
			if m.GetCondition() == operators.ANDCondition {
				return false, nil
			}
			continue
		}
		for _, node := range nodes {
			matchedItems = append(matchedItems, node.InnerText())
		}
		if m.GetCondition() == operators.ORCondition && !m.MatchAll {
			return true, matchedItems
		}
		if len(m.XPath)-1 == i && !m.MatchAll {
			return true, matchedItems
		}
	}
	if len(matchedItems) > 0 && m.MatchAll {
		return true, matchedItems
	}
	return false, nil
}
