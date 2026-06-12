package proton

import (
	"context"
	"os"
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

func writeTempTemplate(t *testing.T, yamlContent string) string {
	t.Helper()
	dir := t.TempDir()
	path := dir + "/template.yaml"
	if err := os.WriteFile(path, []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeTempTemplates(t *testing.T, templates map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range templates {
		if err := os.WriteFile(dir+"/"+name, []byte(content), 0644); err != nil {
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

func sortFindings(findings []Finding) {
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
	eng := mustEngine(t, NewConfig().WithTemplatePaths(tmplPath))

	findings := eng.ScanData([]byte("-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAK\n-----END RSA PRIVATE KEY-----\n"), "secret.pem")
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].TemplateID != "private-key-detect" {
		t.Fatalf("unexpected template ID: %s", findings[0].TemplateID)
	}
}

func TestEngine_LoadFromTemplateDir(t *testing.T) {
	tmplDir := writeTempTemplates(t, map[string]string{
		"privkey.yaml":  tmplPrivateKey,
		"password.yaml": tmplPasswordExtract,
	})

	eng := mustEngine(t, NewConfig().WithTemplatePaths(tmplDir))
	findings := eng.ScanData([]byte("password = hunter2\n-----BEGIN PRIVATE KEY-----\n"), "config.env")
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
	eng := mustEngine(t, NewConfig().WithTemplateData([]byte(tmplArrayYAML)))
	findings := eng.ScanData([]byte("SECRET_TOKEN=abc123\napi_key = my_secret_key_999\n"), "app.conf")
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
	tmplPath := writeTempTemplate(t, tmplPrivateKey)
	rules, err := NewConfig().WithTemplatePaths(tmplPath).Load()
	if err != nil {
		t.Fatalf("load rules: %v", err)
	}

	eng := mustEngine(t, NewConfig().WithRules(rules...))
	findings := eng.ScanData([]byte("PRIVATE KEY found\n"), "test.txt")
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
}

// ========================================
// 匹配能力
// ========================================

func TestEngine_WordMatcher_AND(t *testing.T) {
	eng := mustEngine(t, NewConfig().WithTemplatePaths(writeTempTemplate(t, tmplANDCondition)))

	findings := eng.ScanData([]byte("password=secret123\nusername=admin\n"), "both.txt")
	if len(findings) != 1 {
		t.Fatalf("AND condition: expected 1 finding, got %d", len(findings))
	}

	findings = eng.ScanData([]byte("password=secret123\nno user here\n"), "partial.txt")
	if len(findings) != 0 {
		t.Fatalf("AND condition partial: expected 0 findings, got %d", len(findings))
	}
}

func TestEngine_RegexMatcher(t *testing.T) {
	eng := mustEngine(t, NewConfig().WithTemplatePaths(writeTempTemplate(t, tmplRegexMatcher)))

	findings := eng.ScanData([]byte("token = 'a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4'\n"), "match.txt")
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}

	findings = eng.ScanData([]byte("token = short\n"), "nomatch.txt")
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings, got %d", len(findings))
	}
}

func TestEngine_AWSKeyDetection(t *testing.T) {
	eng := mustEngine(t, NewConfig().WithTemplatePaths(writeTempTemplate(t, tmplAWSKey)))

	findings := eng.ScanData([]byte("aws_access_key_id = AKIAIOSFODNN7EXAMPLE\n"), "credentials")
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

func TestEngine_NoFindings(t *testing.T) {
	eng := mustEngine(t, NewConfig().WithTemplatePaths(writeTempTemplate(t, tmplPrivateKey)))

	findings := eng.ScanData([]byte("just regular text\n"), "clean.txt")
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings, got %d", len(findings))
	}
}

func TestEngine_EmptyTemplates(t *testing.T) {
	eng := mustEngine(t, NewConfig())

	findings := eng.ScanData([]byte("PRIVATE KEY\npassword=test\n"), "secret.txt")
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings with no templates, got %d", len(findings))
	}
}

// ========================================
// Execute 接口 (types.Engine)
// ========================================

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

	if _, err := eng.Execute(NewContext(), NewScanDataTask(nil, "test.txt")); err == nil {
		t.Fatal("expected validation error for empty data")
	}
	if _, err := eng.Execute(NewContext(), NewScanDataTask([]byte("data"), "")); err == nil {
		t.Fatal("expected validation error for empty label")
	}
}

// ========================================
// Context cancel
// ========================================

func TestEngine_ContextCancel(t *testing.T) {
	eng := mustEngine(t, NewConfig().WithTemplatePaths(writeTempTemplate(t, tmplPrivateKey)))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	resultCh, err := eng.Execute(NewContext().WithContext(ctx), NewScanDataTask([]byte("PRIVATE KEY\n"), "test.txt"))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	var count int
	for range resultCh {
		count++
	}
	t.Logf("cancelled scan produced %d results (expected 0 or partial)", count)
}

// ========================================
// Stats callback
// ========================================

func TestEngine_StatsCallback(t *testing.T) {
	eng := mustEngine(t, NewConfig().WithTemplatePaths(writeTempTemplate(t, tmplPrivateKey)))

	var called int32
	ctx := NewContext().SetStatsHandler(func(s types.Stats) {
		atomic.AddInt32(&called, 1)
		if s.Engine != "proton" {
			t.Errorf("expected engine=proton, got %s", s.Engine)
		}
	})

	resultCh, err := eng.Execute(ctx, NewScanDataTask([]byte("PRIVATE KEY\n"), "secret.txt"))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	var count int
	for range resultCh {
		count++
	}
	if count != 1 {
		t.Fatalf("expected 1 finding, got %d", count)
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

	eng := mustEngine(t, NewConfig().WithTemplatePaths(tmplDir).WithTags("cloud"))
	findings := eng.ScanData([]byte("PRIVATE KEY\nAKIAIOSFODNN7EXAMPLE\npassword = test\n"), "all.txt")
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

	eng := mustEngine(t, NewConfig().WithTemplatePaths(tmplDir).WithExcludeTags("cloud"))
	findings := eng.ScanData([]byte("PRIVATE KEY\nAKIAIOSFODNN7EXAMPLE\n"), "all.txt")
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

	eng := mustEngine(t, NewConfig().WithTemplatePaths(tmplDir).WithIDs("password-extract"))
	findings := eng.ScanData([]byte("PRIVATE KEY\npassword = test\n"), "all.txt")
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

func TestEngine_ScanBlock(t *testing.T) {
	eng := mustEngine(t, NewConfig().WithTemplatePaths(writeTempTemplate(t, tmplPrivateKey)))

	findings := eng.ScanBlock([]byte("PRIVATE KEY"), "mem:pid1234")
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
}

// ========================================
// 流式接口
// ========================================

func TestEngine_NewLineWriter(t *testing.T) {
	eng := mustEngine(t, NewConfig().WithTemplatePaths(writeTempTemplate(t, tmplPasswordExtract)))

	var findings []Finding
	w := eng.NewLineWriter("stream:env", func(f Finding) {
		findings = append(findings, f)
	})

	w.Write([]byte("some_var=hello\n"))
	w.Write([]byte("database_password = s3cret\n"))
	w.Write([]byte("other_var=world\n"))
	w.Close()

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Result.OutputExtracts[0] != "s3cret" {
		t.Fatalf("unexpected extract: %s", findings[0].Result.OutputExtracts[0])
	}
}

func TestEngine_NewLineWriter_SplitChunks(t *testing.T) {
	eng := mustEngine(t, NewConfig().WithTemplatePaths(writeTempTemplate(t, tmplPrivateKey)))

	var findings []Finding
	w := eng.NewLineWriter("stream:chunked", func(f Finding) {
		findings = append(findings, f)
	})

	// 一行被拆成两个 chunk 写入
	w.Write([]byte("-----BEGIN PRIV"))
	w.Write([]byte("ATE KEY-----\n"))
	w.Close()

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding from split chunks, got %d", len(findings))
	}
}

func TestEngine_NewBlockWriter(t *testing.T) {
	eng := mustEngine(t, NewConfig().WithTemplatePaths(writeTempTemplate(t, tmplPrivateKey)))

	var findings []Finding
	w := eng.NewBlockWriter("mem:pid5678", func(f Finding) {
		findings = append(findings, f)
	})

	w.Write([]byte("some binary data PRIVATE KEY more binary data"))
	w.Close()

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
}
