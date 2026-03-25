package cyberhub

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/chainreactors/fingers/common"
)

// FingerprintScanRequest describes a CyberHub fingerprint scan request.
type FingerprintScanRequest struct {
	RuleIDs    []uint `json:"rule_ids,omitempty"`
	RawContent string `json:"raw_content,omitempty"`
	Engine     string `json:"engine,omitempty"`
	TargetURL  string `json:"target_url"`
	Level      int    `json:"level,omitempty"`
	Timeout    int    `json:"timeout,omitempty"`
}

// FingerprintScanResponse is the flattened CyberHub fingerprint scan response.
type FingerprintScanResponse struct {
	ScanID  uint                   `json:"scan_id,omitempty"`
	Slug    string                 `json:"slug,omitempty"`
	Status  string                 `json:"status,omitempty"`
	Matches []FingerprintScanMatch `json:"matches,omitempty"`
}

// FingerprintScanMatch is a normalized fingerprint hit suitable for result correlation.
type FingerprintScanMatch struct {
	FingerprintID uint                `json:"fingerprint_id,omitempty"`
	AliasID       uint                `json:"alias_id,omitempty"`
	Name          string              `json:"name,omitempty"`
	Protocol      string              `json:"protocol,omitempty"`
	Version       string              `json:"version,omitempty"`
	Vendor        string              `json:"vendor,omitempty"`
	Product       string              `json:"product,omitempty"`
	Category      string              `json:"category,omitempty"`
	CPE           string              `json:"cpe,omitempty"`
	Tags          []string            `json:"tags,omitempty"`
	Error         string              `json:"error,omitempty"`
	Fingerprint   *FingerprintSummary `json:"fingerprint,omitempty"`
	Framework     *common.Framework   `json:"framework,omitempty"`
}

// FingerprintSummary keeps the minimal fingerprint payload returned by the scan API.
type FingerprintSummary struct {
	ID       uint          `json:"id,omitempty"`
	Name     string        `json:"name,omitempty"`
	Protocol string        `json:"protocol,omitempty"`
	Alias    *AliasSummary `json:"alias,omitempty"`
	Tags     []string      `json:"tags,omitempty"`
}

// AliasSummary keeps the minimal alias payload returned by the scan API.
type AliasSummary struct {
	ID       uint     `json:"id,omitempty"`
	Name     string   `json:"name,omitempty"`
	Vendor   string   `json:"vendor,omitempty"`
	Product  string   `json:"product,omitempty"`
	Category string   `json:"category,omitempty"`
	Version  string   `json:"version,omitempty"`
	Tags     []string `json:"tags,omitempty"`
}

type fingerprintListItem struct {
	ID uint `json:"id"`
}

type fingerprintListPage struct {
	Fingerprints []fingerprintListItem `json:"fingerprints"`
	Total        int                   `json:"total"`
	Page         int                   `json:"page"`
	PageSize     int                   `json:"page_size"`
}

// ScanFingerprints executes CyberHub fingerprint scan and normalizes the result for correlation use.
func (c *Client) ScanFingerprints(ctx context.Context, req *FingerprintScanRequest) (*FingerprintScanResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("scan request is required")
	}
	targetURL := strings.TrimSpace(req.TargetURL)
	if targetURL == "" {
		return nil, fmt.Errorf("target_url is required")
	}

	ruleIDs, err := c.resolveFingerprintRuleIDs(ctx, req.RuleIDs, req.RawContent)
	if err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"target_url": targetURL,
		"level":      normalizeScanLevel(req.Level),
		"timeout":    normalizeScanTimeout(req.Timeout),
	}
	if len(ruleIDs) > 0 {
		payload["rule_ids"] = ruleIDs
	}
	if trimmed := strings.TrimSpace(req.RawContent); trimmed != "" {
		payload["raw_content"] = trimmed
	}
	if trimmed := strings.TrimSpace(req.Engine); trimmed != "" {
		payload["engine"] = trimmed
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal scan request failed: %w", err)
	}

	endpoint := fmt.Sprintf("%s/fingerprints/scan", c.baseURL)
	var raw map[string]interface{}
	if err := c.doRequest(ctx, "POST", endpoint, bytes.NewReader(body), &raw); err != nil {
		return nil, fmt.Errorf("scan fingerprints failed: %w", err)
	}

	return parseFingerprintScanResponse(raw), nil
}

func normalizeScanLevel(level int) int {
	if level < 0 {
		return 0
	}
	if level > 2 {
		return 2
	}
	return level
}

func normalizeScanTimeout(timeout int) int {
	if timeout <= 0 {
		return 10
	}
	if timeout > 300 {
		return 300
	}
	return timeout
}

func (c *Client) resolveFingerprintRuleIDs(ctx context.Context, ruleIDs []uint, rawContent string) ([]uint, error) {
	if trimmed := strings.TrimSpace(rawContent); trimmed != "" {
		return normalizeRuleIDs(ruleIDs), nil
	}
	if len(ruleIDs) > 0 {
		return normalizeRuleIDs(ruleIDs), nil
	}

	page := 1
	pageSize := 1000
	seen := make(map[uint]struct{})
	collected := make([]uint, 0)
	for {
		params := url.Values{}
		params.Set("page", strconv.Itoa(page))
		params.Set("page_size", strconv.Itoa(pageSize))
		params.Set("status", "active")
		endpoint := fmt.Sprintf("%s/fingerprints?%s", c.baseURL, params.Encode())
		var listing fingerprintListPage
		if err := c.doRequest(ctx, "GET", endpoint, nil, &listing); err != nil {
			return nil, fmt.Errorf("list fingerprints failed: %w", err)
		}
		if len(listing.Fingerprints) == 0 {
			break
		}
		for _, item := range listing.Fingerprints {
			if item.ID == 0 {
				continue
			}
			if _, exists := seen[item.ID]; exists {
				continue
			}
			seen[item.ID] = struct{}{}
			collected = append(collected, item.ID)
		}
		if len(listing.Fingerprints) < pageSize {
			break
		}
		if listing.Total > 0 && len(collected) >= listing.Total {
			break
		}
		page++
	}
	return collected, nil
}

func normalizeRuleIDs(ruleIDs []uint) []uint {
	seen := make(map[uint]struct{}, len(ruleIDs))
	result := make([]uint, 0, len(ruleIDs))
	for _, ruleID := range ruleIDs {
		if ruleID == 0 {
			continue
		}
		if _, exists := seen[ruleID]; exists {
			continue
		}
		seen[ruleID] = struct{}{}
		result = append(result, ruleID)
	}
	return result
}

func parseFingerprintScanResponse(raw map[string]interface{}) *FingerprintScanResponse {
	response := &FingerprintScanResponse{
		ScanID: parsePositiveUint(raw["id"]),
		Slug:   getString(raw, "slug"),
		Status: getString(raw, "status"),
	}

	nestedData := getMap(raw, "data")
	rawResults := getSlice(getMap(nestedData, "fingerprint"), "results")
	if len(rawResults) == 0 {
		rawResults = getSlice(getMap(raw, "fingerprint"), "results")
	}
	if len(rawResults) > 0 {
		response.Matches = parseStandardFingerprintMatches(rawResults)
		return response
	}

	frameworks := raw["frameworks"]
	response.Matches = parseLegacyFingerprintMatches(frameworks)
	return response
}

func parseStandardFingerprintMatches(rawResults []interface{}) []FingerprintScanMatch {
	matches := make([]FingerprintScanMatch, 0, len(rawResults))
	for _, item := range rawResults {
		payload, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		fingerprintMap := getMap(payload, "fingerprint")
		fingerprint := parseFingerprintSummary(fingerprintMap)
		frameworkMap := getMap(payload, "framework")
		framework := parseFramework(frameworkMap)

		match := FingerprintScanMatch{
			FingerprintID: parsePositiveUint(payload["fingerprint_id"]),
			Error:         getString(payload, "error"),
			Fingerprint:   fingerprint,
			Framework:     framework,
		}
		if match.FingerprintID == 0 && fingerprint != nil {
			match.FingerprintID = fingerprint.ID
		}
		if fingerprint != nil && fingerprint.Alias != nil {
			match.AliasID = fingerprint.Alias.ID
		}
		match.Name = firstNonEmptyString(
			getString(frameworkMap, "name"),
			getString(frameworkMap, "framework"),
			getString(frameworkMap, "product"),
			getString(getMap(fingerprintMap, "alias"), "name"),
			getString(fingerprintMap, "name"),
		)
		if match.Name == "" {
			continue
		}
		match.Protocol = firstNonEmptyString(getString(fingerprintMap, "protocol"))
		match.Version = firstNonEmptyString(
			getString(frameworkMap, "version"),
			getString(getMap(frameworkMap, "attributes"), "version"),
			getString(getMap(fingerprintMap, "alias"), "version"),
			getString(fingerprintMap, "version"),
		)
		match.Vendor = firstNonEmptyString(
			getString(frameworkMap, "vendor"),
			getString(getMap(frameworkMap, "attributes"), "vendor"),
			getString(getMap(fingerprintMap, "alias"), "vendor"),
			getString(fingerprintMap, "vendor"),
		)
		match.Product = firstNonEmptyString(
			getString(frameworkMap, "product"),
			getString(getMap(frameworkMap, "attributes"), "product"),
			getString(getMap(fingerprintMap, "alias"), "product"),
			getString(fingerprintMap, "product"),
		)
		match.Category = firstNonEmptyString(
			getString(frameworkMap, "category"),
			getString(getMap(fingerprintMap, "alias"), "category"),
			getString(fingerprintMap, "category"),
		)
		match.CPE = firstNonEmptyString(
			getString(frameworkMap, "cpe"),
			getString(getMap(frameworkMap, "attributes"), "cpe"),
			frameworkCPE(framework),
			getString(getMap(fingerprintMap, "alias"), "cpe"),
			getString(fingerprintMap, "cpe"),
		)
		match.Tags = normalizeStringSlice(
			getInterfaceSlice(frameworkMap["tags"]),
			getInterfaceSlice(fingerprintMap["tags"]),
			getInterfaceSlice(getMap(fingerprintMap, "alias")["tags"]),
		)
		matches = append(matches, match)
	}
	return matches
}

func parseLegacyFingerprintMatches(frameworks interface{}) []FingerprintScanMatch {
	switch typed := frameworks.(type) {
	case map[string]interface{}:
		matches := make([]FingerprintScanMatch, 0, len(typed))
		for name, payload := range typed {
			if match, ok := parseLegacyFrameworkItem(name, payload); ok {
				matches = append(matches, match)
			}
		}
		return matches
	case []interface{}:
		matches := make([]FingerprintScanMatch, 0, len(typed))
		for _, payload := range typed {
			if match, ok := parseLegacyFrameworkItem("", payload); ok {
				matches = append(matches, match)
			}
		}
		return matches
	default:
		if match, ok := parseLegacyFrameworkItem("", typed); ok {
			return []FingerprintScanMatch{match}
		}
		return nil
	}
}

func parseLegacyFrameworkItem(nameHint string, payload interface{}) (FingerprintScanMatch, bool) {
	switch typed := payload.(type) {
	case map[string]interface{}:
		aliasMap := getMap(typed, "alias")
		match := FingerprintScanMatch{
			FingerprintID: parsePositiveUint(typed["fingerprint_id"]),
			AliasID:       parsePositiveUint(firstNonNil(typed["alias_id"], aliasMap["id"])),
			Name:          firstNonEmptyString(nameHint, getString(typed, "name"), getString(typed, "framework"), getString(typed, "product")),
			Protocol:      firstNonEmptyString(getString(typed, "protocol")),
			Version:       firstNonEmptyString(getString(typed, "version")),
			Vendor:        firstNonEmptyString(getString(typed, "vendor"), getString(aliasMap, "vendor")),
			Product:       firstNonEmptyString(getString(typed, "product"), getString(aliasMap, "product")),
			Category:      firstNonEmptyString(getString(typed, "category"), getString(aliasMap, "category")),
			CPE:           firstNonEmptyString(getString(typed, "cpe"), getString(aliasMap, "cpe")),
			Tags:          normalizeStringSlice(getInterfaceSlice(typed["tags"]), getInterfaceSlice(aliasMap["tags"])),
		}
		if match.Name == "" {
			return FingerprintScanMatch{}, false
		}
		return match, true
	default:
		name := firstNonEmptyString(nameHint, fmt.Sprint(payload))
		if name == "" {
			return FingerprintScanMatch{}, false
		}
		return FingerprintScanMatch{Name: name}, true
	}
}

func parseFingerprintSummary(raw map[string]interface{}) *FingerprintSummary {
	if len(raw) == 0 {
		return nil
	}
	summary := &FingerprintSummary{
		ID:       parsePositiveUint(raw["id"]),
		Name:     getString(raw, "name"),
		Protocol: getString(raw, "protocol"),
		Tags:     normalizeStringSlice(getInterfaceSlice(raw["tags"])),
	}
	if aliasMap := getMap(raw, "alias"); len(aliasMap) > 0 {
		summary.Alias = &AliasSummary{
			ID:       parsePositiveUint(aliasMap["id"]),
			Name:     getString(aliasMap, "name"),
			Vendor:   getString(aliasMap, "vendor"),
			Product:  getString(aliasMap, "product"),
			Category: getString(aliasMap, "category"),
			Version:  getString(aliasMap, "version"),
			Tags:     normalizeStringSlice(getInterfaceSlice(aliasMap["tags"])),
		}
	}
	return summary
}

func parseFramework(raw map[string]interface{}) *common.Framework {
	if len(raw) == 0 {
		return nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var framework common.Framework
	if err := json.Unmarshal(data, &framework); err != nil {
		return nil
	}
	return &framework
}

func frameworkCPE(framework *common.Framework) string {
	if framework == nil || framework.Attributes == nil {
		return ""
	}
	return framework.CPE()
}

func parsePositiveUint(value interface{}) uint {
	switch typed := value.(type) {
	case float64:
		if typed > 0 {
			return uint(typed)
		}
	case float32:
		if typed > 0 {
			return uint(typed)
		}
	case int:
		if typed > 0 {
			return uint(typed)
		}
	case int64:
		if typed > 0 {
			return uint(typed)
		}
	case json.Number:
		parsed, err := typed.Int64()
		if err == nil && parsed > 0 {
			return uint(parsed)
		}
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return 0
		}
		parsed, err := strconv.ParseUint(trimmed, 10, 64)
		if err == nil && parsed > 0 {
			return uint(parsed)
		}
	case uint:
		return typed
	case uint64:
		return uint(typed)
	}
	return 0
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func firstNonNil(values ...interface{}) interface{} {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func getMap(source map[string]interface{}, key string) map[string]interface{} {
	if source == nil {
		return map[string]interface{}{}
	}
	value, ok := source[key]
	if !ok || value == nil {
		return map[string]interface{}{}
	}
	typed, ok := value.(map[string]interface{})
	if !ok {
		return map[string]interface{}{}
	}
	return typed
}

func getSlice(source map[string]interface{}, key string) []interface{} {
	if source == nil {
		return nil
	}
	value, ok := source[key]
	if !ok {
		return nil
	}
	return getInterfaceSlice(value)
}

func getInterfaceSlice(value interface{}) []interface{} {
	if value == nil {
		return nil
	}
	typed, ok := value.([]interface{})
	if !ok {
		return nil
	}
	return typed
}

func getString(source map[string]interface{}, key string) string {
	if source == nil {
		return ""
	}
	value, ok := source[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func normalizeStringSlice(candidates ...[]interface{}) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0)
	for _, candidate := range candidates {
		for _, item := range candidate {
			text := strings.TrimSpace(fmt.Sprint(item))
			if text == "" {
				continue
			}
			if _, exists := seen[text]; exists {
				continue
			}
			seen[text] = struct{}{}
			result = append(result, text)
		}
	}
	return result
}
