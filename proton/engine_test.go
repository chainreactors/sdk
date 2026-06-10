package proton

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"sync/atomic"
	"testing"

	"github.com/chainreactors/sdk/pkg/types"
)

// ========================================
// 内嵌模板 YAML（CI 自包含，不依赖外部文件）
// ========================================

const tmplPrivateKey = `
id: private-key-detect
info:
  name: Private Key Detection
  severity: high
  tags: file,keys
file:
  - extensions:
      - all
    matchers:
      - type: word
        words:
          - "PRIVATE KEY"
`

const tmplAWSKey = `
id: aws-access-key
info:
  name: AWS Access Key
  severity: critical
  tags: cloud,aws
file:
  - extensions:
      - all
    matchers:
      - type: word
        words:
          - "AKIA"
          - "AGPA"
          - "ASIA"
        condition: or
    extractors:
      - type: regex
        name: aws-ak
        regex:
          - "(AKIA|AGPA|ASIA)[a-zA-Z0-9]{16}"
`

const tmplPasswordExtract = `
id: password-extract
info:
  name: Password Extraction
  severity: medium
  tags: file,credential
file:
  - extensions:
      - all
    extractors:
      - type: regex
        name: password
        regex:
          - "password\\s*=\\s*(\\S+)"
        group: 1
`

const tmplArrayYAML = `
- id: array-word
  info:
    name: Array Word Test
    severity: info
    tags: test
  file:
    - extensions:
        - all
      matchers:
        - type: word
          words:
            - "SECRET_TOKEN"

- id: array-regex
  info:
    name: Array Regex Test
    severity: low
    tags: test
  file:
    - extensions:
        - all
      extractors:
        - type: regex
          regex:
            - "api_key\\s*=\\s*(\\S+)"
          group: 1
`

const tmplExtFilter = `
id: yaml-only
info:
  name: YAML Only Check
  severity: info
file:
  - extensions:
      - .yaml
      - .yml
    matchers:
      - type: word
        words:
          - "secret"
`

const tmplRegexMatcher = `
id: regex-matcher
info:
  name: Regex Matcher
  severity: low
  tags: test
file:
  - extensions:
      - all
    matchers:
      - type: regex
        regex:
          - "token\\s*[:=]\\s*['\"][a-f0-9]{32}['\"]"
`

const tmplANDCondition = `
id: and-condition
info:
  name: AND Condition
  severity: medium
  tags: test
file:
  - extensions:
      - all
    matchers-condition: and
    matchers:
      - type: word
        words:
          - "password"
      - type: word
        words:
          - "username"
`

// ========================================
// 辅助函数
// ========================================

func writeTempDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		sub := filepath.Dir(name)
		if sub != "." {
			if err := os.MkdirAll(filepath.Join(dir, sub), 0755); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func writeTempTemplate(t *testing.T, yamlContent string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "template.yaml")
	if err := os.WriteFile(path, []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeTempTemplates(t *testing.T, templates map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range templates {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func mustEngine(t *testing.T, cfg *Config) *Engine {
	t.Helper()
	eng, err := NewEngine(cfg)
	if err != nil {
		t.Fatalf("engine init: %v", err)
	}
	return eng
}

func sortFindings(findings []*Finding) {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].TemplateID != findings[j].TemplateID {
			return findings[i].TemplateID < findings[j].TemplateID
		}
		return findings[i].FilePath < findings[j].FilePath
	})
}

// ========================================
// 模板加载
// ========================================

func TestEngine_LoadFromTemplatePath(t *testing.T) {
	tmplPath := writeTempTemplate(t, tmplPrivateKey)
	dir := writeTempDir(t, map[string]string{
		"secret.pem": "-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAK\n-----END RSA PRIVATE KEY-----\n",
		"clean.txt":  "nothing here\n",
	})

	eng := mustEngine(t, NewConfig().WithTemplatePaths(tmplPath))
	findings, err := eng.Scan(NewContext(), dir)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].TemplateID != "private-key-detect" {
		t.Fatalf("unexpected template ID: %s", findings[0].TemplateID)
	}
	if filepath.Base(findings[0].FilePath) != "secret.pem" {
		t.Fatalf("unexpected file: %s", findings[0].FilePath)
	}
}

func TestEngine_LoadFromTemplateDir(t *testing.T) {
	tmplDir := writeTempTemplates(t, map[string]string{
		"privkey.yaml":  tmplPrivateKey,
		"password.yaml": tmplPasswordExtract,
	})
	dir := writeTempDir(t, map[string]string{
		"config.env": "password = hunter2\n-----BEGIN PRIVATE KEY-----\n",
	})

	eng := mustEngine(t, NewConfig().WithTemplatePaths(tmplDir))
	findings, err := eng.Scan(NewContext(), dir)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	sortFindings(findings)

	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(findings))
	}
	ids := []string{findings[0].TemplateID, findings[1].TemplateID}
	sort.Strings(ids)
	if ids[0] != "password-extract" || ids[1] != "private-key-detect" {
		t.Fatalf("unexpected IDs: %v", ids)
	}
}

func TestEngine_LoadFromTemplateData(t *testing.T) {
	dir := writeTempDir(t, map[string]string{
		"app.conf": "SECRET_TOKEN=abc123\napi_key = my_secret_key_999\n",
	})

	eng := mustEngine(t, NewConfig().WithTemplateData([]byte(tmplArrayYAML)))
	findings, err := eng.Scan(NewContext(), dir)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	sortFindings(findings)

	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(findings))
	}
	ids := []string{findings[0].TemplateID, findings[1].TemplateID}
	sort.Strings(ids)
	if ids[0] != "array-regex" || ids[1] != "array-word" {
		t.Fatalf("unexpected IDs: %v", ids)
	}

	for _, f := range findings {
		if f.TemplateID == "array-regex" {
			if f.Result == nil || len(f.Result.OutputExtracts) == 0 {
				t.Fatal("expected extracted values for array-regex")
			}
			if f.Result.OutputExtracts[0] != "my_secret_key_999" {
				t.Fatalf("unexpected extract: %s", f.Result.OutputExtracts[0])
			}
		}
	}
}

func TestEngine_WithPrecompiledRules(t *testing.T) {
	dir := writeTempDir(t, map[string]string{
		"test.txt": "PRIVATE KEY found\n",
	})

	tmplPath := writeTempTemplate(t, tmplPrivateKey)
	rules, err := NewConfig().WithTemplatePaths(tmplPath).Load()
	if err != nil {
		t.Fatalf("load rules: %v", err)
	}

	eng := mustEngine(t, NewConfig().WithRules(rules...))
	findings, scanErr := eng.Scan(NewContext(), dir)
	if scanErr != nil {
		t.Fatalf("scan: %v", scanErr)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
}

// ========================================
// 匹配能力
// ========================================

func TestEngine_WordMatcher_AND(t *testing.T) {
	tmplPath := writeTempTemplate(t, tmplANDCondition)
	dir := writeTempDir(t, map[string]string{
		"both.txt":    "password=secret123\nusername=admin\n",
		"partial.txt": "password=secret123\nno user here\n",
	})

	eng := mustEngine(t, NewConfig().WithTemplatePaths(tmplPath))
	findings, err := eng.Scan(NewContext(), dir)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("AND condition: expected 1 finding, got %d", len(findings))
	}
	if !pathContains(findings[0].FilePath, "both.txt") {
		t.Fatalf("expected both.txt, got %s", findings[0].FilePath)
	}
}

func TestEngine_RegexMatcher(t *testing.T) {
	tmplPath := writeTempTemplate(t, tmplRegexMatcher)
	dir := writeTempDir(t, map[string]string{
		"match.txt":   "token = 'a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4'\n",
		"nomatch.txt": "token = short\n",
	})

	eng := mustEngine(t, NewConfig().WithTemplatePaths(tmplPath))
	findings, err := eng.Scan(NewContext(), dir)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if !pathContains(findings[0].FilePath, "match.txt") {
		t.Fatalf("expected match.txt, got %s", findings[0].FilePath)
	}
}

func TestEngine_AWSKeyDetection(t *testing.T) {
	tmplPath := writeTempTemplate(t, tmplAWSKey)
	dir := writeTempDir(t, map[string]string{
		"credentials": "aws_access_key_id = AKIAIOSFODNN7EXAMPLE\n",
		"clean.txt":   "no keys here\n",
	})

	eng := mustEngine(t, NewConfig().WithTemplatePaths(tmplPath))
	findings, err := eng.Scan(NewContext(), dir)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	f := findings[0]
	if f.Severity != "critical" {
		t.Fatalf("unexpected severity: %s", f.Severity)
	}
	if f.Result == nil || len(f.Result.OutputExtracts) == 0 {
		t.Fatal("expected extracted AWS key")
	}
	found := false
	for _, v := range f.Result.OutputExtracts {
		if v == "AKIAIOSFODNN7EXAMPLE" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected AKIAIOSFODNN7EXAMPLE in extracts, got: %v", f.Result.OutputExtracts)
	}
}

func TestEngine_ExtensionFilter(t *testing.T) {
	tmplPath := writeTempTemplate(t, tmplExtFilter)
	dir := writeTempDir(t, map[string]string{
		"config.yaml": "secret: my_token\n",
		"config.yml":  "secret: another_token\n",
		"config.json": "secret: should_not_match\n",
		"config.txt":  "secret: should_not_match\n",
	})

	eng := mustEngine(t, NewConfig().WithTemplatePaths(tmplPath))
	findings, err := eng.Scan(NewContext(), dir)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(findings) != 2 {
		t.Fatalf("expected 2 findings (yaml + yml only), got %d", len(findings))
	}
	for _, f := range findings {
		base := filepath.Base(f.FilePath)
		if base != "config.yaml" && base != "config.yml" {
			t.Fatalf("unexpected file matched: %s", base)
		}
	}
}

func TestEngine_NoFindings(t *testing.T) {
	tmplPath := writeTempTemplate(t, tmplPrivateKey)
	dir := writeTempDir(t, map[string]string{
		"clean1.txt": "just regular text\n",
		"clean2.txt": "nothing secret here\n",
	})

	eng := mustEngine(t, NewConfig().WithTemplatePaths(tmplPath))
	findings, err := eng.Scan(NewContext(), dir)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings, got %d", len(findings))
	}
}

func TestEngine_EmptyTemplates(t *testing.T) {
	eng := mustEngine(t, NewConfig())
	dir := writeTempDir(t, map[string]string{
		"secret.txt": "PRIVATE KEY\npassword=test\n",
	})

	findings, err := eng.Scan(NewContext(), dir)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings with no templates, got %d", len(findings))
	}
}

// ========================================
// ScanData 内存扫描
// ========================================

func TestEngine_ScanData(t *testing.T) {
	eng := mustEngine(t, NewConfig().WithTemplatePaths(writeTempTemplate(t, tmplPasswordExtract)))

	data := []byte("database_password = s3cret_v4lue\nother_line\n")
	findings := eng.ScanData(data, "virtual/config.env")

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Result.OutputExtracts[0] != "s3cret_v4lue" {
		t.Fatalf("unexpected extract: %s", findings[0].Result.OutputExtracts[0])
	}
}

func TestEngine_ScanData_NoMatch(t *testing.T) {
	eng := mustEngine(t, NewConfig().WithTemplatePaths(writeTempTemplate(t, tmplPrivateKey)))

	findings := eng.ScanData([]byte("nothing interesting here\n"), "virtual/clean.txt")
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings, got %d", len(findings))
	}
}

// ========================================
// Execute 接口 (types.Engine)
// ========================================

func TestEngine_Execute_ScanTask(t *testing.T) {
	dir := writeTempDir(t, map[string]string{
		"key.pem": "-----BEGIN PRIVATE KEY-----\ndata\n-----END PRIVATE KEY-----\n",
	})

	eng := mustEngine(t, NewConfig().WithTemplatePaths(writeTempTemplate(t, tmplPrivateKey)))
	resultCh, err := eng.Execute(NewContext(), NewScanTask(dir))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	var count int
	for r := range resultCh {
		if !r.Success() {
			t.Fatalf("unexpected error: %v", r.Error())
		}
		f, ok := r.Data().(*Finding)
		if !ok {
			t.Fatalf("unexpected data type: %T", r.Data())
		}
		if f.TemplateID != "private-key-detect" {
			t.Fatalf("unexpected template: %s", f.TemplateID)
		}
		count++
	}
	if count != 1 {
		t.Fatalf("expected 1 result, got %d", count)
	}
}

func TestEngine_Execute_ScanDataTask(t *testing.T) {
	eng := mustEngine(t, NewConfig().WithTemplatePaths(writeTempTemplate(t, tmplPasswordExtract)))
	resultCh, err := eng.Execute(NewContext(), NewScanDataTask([]byte("password = test123\n"), "app.env"))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	var count int
	for r := range resultCh {
		if !r.Success() {
			t.Fatalf("unexpected error: %v", r.Error())
		}
		count++
	}
	if count != 1 {
		t.Fatalf("expected 1 result, got %d", count)
	}
}

func TestEngine_Execute_ValidationError(t *testing.T) {
	eng := mustEngine(t, NewConfig())

	if _, err := eng.Execute(NewContext(), NewScanTask("")); err == nil {
		t.Fatal("expected validation error for empty target")
	}
	if _, err := eng.Execute(NewContext(), NewScanDataTask(nil, "test.txt")); err == nil {
		t.Fatal("expected validation error for empty data")
	}
}

// ========================================
// ScanStream 流式 API
// ========================================

func TestEngine_ScanStream(t *testing.T) {
	dir := writeTempDir(t, map[string]string{
		"a.txt": "PRIVATE KEY\n",
		"b.txt": "PRIVATE KEY\n",
		"c.txt": "nothing\n",
	})

	eng := mustEngine(t, NewConfig().WithTemplatePaths(writeTempTemplate(t, tmplPrivateKey)))
	ch, err := eng.ScanStream(NewContext(), dir)
	if err != nil {
		t.Fatalf("scan stream: %v", err)
	}

	var count int
	for range ch {
		count++
	}
	if count != 2 {
		t.Fatalf("expected 2 streamed findings, got %d", count)
	}
}

// ========================================
// Context cancel
// ========================================

func TestEngine_ContextCancel(t *testing.T) {
	dir := writeTempDir(t, map[string]string{
		"a.txt": "PRIVATE KEY\n",
		"b.txt": "PRIVATE KEY\n",
		"c.txt": "PRIVATE KEY\n",
	})

	eng := mustEngine(t, NewConfig().WithTemplatePaths(writeTempTemplate(t, tmplPrivateKey)))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	resultCh, err := eng.Execute(NewContext().WithContext(ctx), NewScanTask(dir))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	var count int
	for range resultCh {
		count++
	}
	// cancelled context may produce 0 or partial results — just must not hang
	t.Logf("cancelled scan produced %d results (expected 0 or partial)", count)
}

// ========================================
// Stats callback
// ========================================

func TestEngine_StatsCallback(t *testing.T) {
	dir := writeTempDir(t, map[string]string{
		"secret.txt": "PRIVATE KEY\n",
	})

	eng := mustEngine(t, NewConfig().WithTemplatePaths(writeTempTemplate(t, tmplPrivateKey)))

	var called int32
	ctx := NewContext().SetStatsHandler(func(s types.Stats) {
		atomic.AddInt32(&called, 1)
		if s.Engine != "proton" {
			t.Errorf("expected engine=proton, got %s", s.Engine)
		}
	})

	findings, err := eng.Scan(ctx, dir)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if atomic.LoadInt32(&called) == 0 {
		t.Fatal("stats callback was not invoked")
	}
}

// ========================================
// Config filter
// ========================================

func TestEngine_ConfigFilter_Tags(t *testing.T) {
	tmplDir := writeTempTemplates(t, map[string]string{
		"privkey.yaml":  tmplPrivateKey,
		"aws.yaml":      tmplAWSKey,
		"password.yaml": tmplPasswordExtract,
	})
	dir := writeTempDir(t, map[string]string{
		"all.txt": "PRIVATE KEY\nAKIAIOSFODNN7EXAMPLE\npassword = test\n",
	})

	eng := mustEngine(t, NewConfig().WithTemplatePaths(tmplDir).WithTags("cloud"))
	findings, err := eng.Scan(NewContext(), dir)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding (only cloud tag), got %d", len(findings))
	}
	if findings[0].TemplateID != "aws-access-key" {
		t.Fatalf("unexpected template: %s", findings[0].TemplateID)
	}
}

func TestEngine_ConfigFilter_ExcludeTags(t *testing.T) {
	tmplDir := writeTempTemplates(t, map[string]string{
		"privkey.yaml": tmplPrivateKey,
		"aws.yaml":     tmplAWSKey,
	})
	dir := writeTempDir(t, map[string]string{
		"all.txt": "PRIVATE KEY\nAKIAIOSFODNN7EXAMPLE\n",
	})

	eng := mustEngine(t, NewConfig().WithTemplatePaths(tmplDir).WithExcludeTags("cloud"))
	findings, err := eng.Scan(NewContext(), dir)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding (cloud excluded), got %d", len(findings))
	}
	if findings[0].TemplateID != "private-key-detect" {
		t.Fatalf("unexpected template: %s", findings[0].TemplateID)
	}
}

func TestEngine_ConfigFilter_IDs(t *testing.T) {
	tmplDir := writeTempTemplates(t, map[string]string{
		"privkey.yaml":  tmplPrivateKey,
		"password.yaml": tmplPasswordExtract,
	})
	dir := writeTempDir(t, map[string]string{
		"all.txt": "PRIVATE KEY\npassword = test\n",
	})

	eng := mustEngine(t, NewConfig().WithTemplatePaths(tmplDir).WithIDs("password-extract"))
	findings, err := eng.Scan(NewContext(), dir)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].TemplateID != "password-extract" {
		t.Fatalf("unexpected template: %s", findings[0].TemplateID)
	}
}

// ========================================
// 基础属性
// ========================================

func TestEngine_Name(t *testing.T) {
	eng, _ := NewEngine(NewConfig())
	if eng.Name() != "proton" {
		t.Fatalf("unexpected name: %s", eng.Name())
	}
}

func TestEngine_Scanner_Stats(t *testing.T) {
	dir := writeTempDir(t, map[string]string{
		"a.txt": "PRIVATE KEY\n",
		"b.txt": "nothing\n",
	})

	eng := mustEngine(t, NewConfig().WithTemplatePaths(writeTempTemplate(t, tmplPrivateKey)))
	_, _ = eng.Scan(NewContext(), dir)

	s := eng.Scanner()
	if s == nil {
		t.Fatal("Scanner() returned nil after Init")
	}
	if s.Stats.Files == 0 {
		t.Fatal("expected Stats.Files > 0 after scan")
	}
}

// ========================================
// helpers
// ========================================

func pathContains(path, sub string) bool {
	return filepath.Base(path) == sub
}
