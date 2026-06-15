#!/usr/bin/env bash
# ci-examples.sh — E2E functional tests for SDK examples.
# Compiles each example binary and runs 2-3 real scenarios per engine.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

PASS=0
FAIL=0
SKIP=0
ERRORS=()

log()  { printf "\033[1;34m==>\033[0m %s\n" "$*"; }
ok()   { printf "\033[1;32m  ✓\033[0m %s\n" "$*"; PASS=$((PASS + 1)); }
fail() { printf "\033[1;31m  ✗\033[0m %s -- %s\n" "$1" "$2"; FAIL=$((FAIL + 1)); ERRORS+=("$1: $2"); }
skip() { printf "\033[1;33m  ⊘\033[0m %s -- %s\n" "$1" "$2"; SKIP=$((SKIP + 1)); }

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

# ============================================================
# Phase 1: Build all example binaries
# ============================================================
log "Building example binaries..."
PACKAGES=$(go list -f '{{if eq .Name "main"}}{{.ImportPath}}{{end}}' ./examples/...)
for pkg in $PACKAGES; do
    name="${pkg#github.com/chainreactors/sdk/examples/}"
    bin="$TMPDIR/${name//\//_}"
    if ! go build -o "$bin" "$pkg" 2>&1; then
        fail "$name" "build failed"
    fi
done
log "All binaries built"

# ============================================================
# Helpers
# ============================================================
run() {
    local name="$1"; shift
    local bin="$TMPDIR/${name//\//_}"
    local output
    output=$("$bin" "$@" 2>&1) || true
    if echo "$output" | grep -qE "^(panic:|fatal error:|goroutine [0-9]+ )"; then
        fail "$name" "panic/fatal"
        echo "$output" | head -20
        return 1
    fi
    echo "$output"
}

assert() {
    local name="$1" output="$2" pattern="$3"
    if echo "$output" | grep -qF "$pattern"; then
        return 0
    fi
    fail "$name" "expected: $pattern"
    echo "  actual (first 5 lines):"
    echo "$output" | head -5 | sed 's/^/    /'
    return 1
}

assert_re() {
    local name="$1" output="$2" pattern="$3"
    if echo "$output" | grep -qE "$pattern"; then
        return 0
    fi
    fail "$name" "expected regex: $pattern"
    echo "  actual (first 5 lines):"
    echo "$output" | head -5 | sed 's/^/    /'
    return 1
}

echo ""
# ============================================================
# engines/fingers — HTTPMatch 主动探测
# ============================================================
log "engines/fingers"

# 1) 基本主动探测 + embed 数据
OUT=$(run engines_fingers -target "https://httpbin.org" -level 1 -timeout 10)
assert_re "fingers:basic" "$OUT" "httpbin.org" && ok "fingers: basic active probe"

# 2) JSON 输出
OUT=$(run engines_fingers -target "https://httpbin.org" -level 1 -timeout 10 -json)
assert "fingers:json" "$OUT" "Target" && ok "fingers: JSON output"

# 3) 多目标批量
OUT=$(run engines_fingers -target "https://httpbin.org,https://example.com" -level 1 -timeout 10)
assert_re "fingers:batch" "$OUT" "example.com" && ok "fingers: batch multi-target"

# ============================================================
# engines/gogo — 端口扫描 + 指纹
# ============================================================
log "engines/gogo"

# 1) 基本扫描
OUT=$(run engines_gogo -target 93.184.216.34 -ports 80 -threads 50 -version 0)
assert "gogo:basic" "$OUT" "80" && ok "gogo: scan port 80"

# 2) 带指纹识别
OUT=$(run engines_gogo -target 93.184.216.34 -ports 80 -threads 50 -version 1)
assert "gogo:finger" "$OUT" "80" && ok "gogo: scan with version detect"

# 3) JSON 输出
OUT=$(run engines_gogo -target 93.184.216.34 -ports 80 -threads 50 -json)
assert "gogo:json" "$OUT" "\"ip\"" && ok "gogo: JSON output"

# ============================================================
# engines/spray — HTTP 批量探测
# ============================================================
log "engines/spray"

# 1) 基本 Check + 状态码匹配
OUT=$(run engines_spray -u "http://example.com" -mc 200 -threads 2 -timeout 10)
assert "spray:check" "$OUT" "[200]" && ok "spray: check 200"

# 2) JSON 输出
OUT=$(run engines_spray -u "http://example.com" -mc 200 -json -threads 2 -timeout 10)
assert "spray:json" "$OUT" "\"matched_count\": 1" && ok "spray: JSON matched_count=1"

# 3) match 不存在的状态码 → 0 matched
OUT=$(run engines_spray -u "http://example.com" -mc 999 -threads 2 -timeout 10)
assert "spray:filter" "$OUT" "Matched: 0" && ok "spray: no match with mc=999"

# ============================================================
# engines/neutron — POC 漏洞扫描
# ============================================================
log "engines/neutron"

# 1) 加载 embed POC + 扫描目标
OUT=$(run engines_neutron -target "https://httpbin.org" -max 5 2>&1)
assert "neutron:basic" "$OUT" "Loaded" && ok "neutron: loaded embed POCs"

# 2) 列出 POC
OUT=$(run engines_neutron -list 2>&1)
assert "neutron:list" "$OUT" "Loaded" && ok "neutron: list mode"

# 3) 按 severity 过滤
OUT=$(run engines_neutron -list -severity critical 2>&1)
assert "neutron:filter" "$OUT" "Loaded" && ok "neutron: severity filter"

# ============================================================
# engines/proton — 敏感数据扫描
# ============================================================
log "engines/proton"

# 准备测试数据
cat > "$TMPDIR/secrets.txt" <<'EOF'
# config
AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
normal_line=nothing_here
GITHUB_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
EOF

# 1) 批量扫描
OUT=$(run engines_proton -input "$TMPDIR/secrets.txt")
assert "proton:batch" "$OUT" "Scan complete" && ok "proton: batch scan"

# 2) 流式扫描
OUT=$(run engines_proton -input "$TMPDIR/secrets.txt" -stream)
assert "proton:stream" "$OUT" "Stream scan complete" && ok "proton: streaming scan"

# 3) stdin 输入
OUT=$(echo "password=SuperSecret123" | run engines_proton -input -)
assert "proton:stdin" "$OUT" "Scan complete" && ok "proton: stdin scan"

# ============================================================
# engines/zombie — 弱口令爆破
# ============================================================
log "engines/zombie"

# 1) brute 模式（目标不可达，验证流程正确性）
OUT=$(run engines_zombie -target 127.0.0.1 -service ssh -port 1 -threads 1 -timeout 1 -mode brute)
ok "zombie: brute mode ran"

# 2) sniper 模式
OUT=$(run engines_zombie -target 127.0.0.1 -service redis -port 1 -threads 1 -timeout 1 -mode sniper)
ok "zombie: sniper mode ran"

# 3) pitchfork 模式
OUT=$(run engines_zombie -target 127.0.0.1 -service ssh -port 1 -threads 1 -timeout 1 -mode pitchfork -auths "root::123456")
ok "zombie: pitchfork mode ran"

# ============================================================
# sniper — 指纹→关联→攻击 workflow
# ============================================================
log "sniper"

# 1) 完整 workflow（embed 数据）— Step 1 always runs, Step 2 needs fingerprint hits
OUT=$(run sniper -target "http://example.com")
assert "sniper:workflow" "$OUT" "Step 1" && ok "sniper: workflow step 1"

# 2) 验证 Step 2 也能到达（如果 Step 1 识别到了指纹）
if echo "$OUT" | grep -qF "Step 2"; then
    ok "sniper: workflow step 2 (association lookup)"
else
    skip "sniper:step2" "no fingerprints matched (network or no active probe hit)"
fi

# ============================================================
# cases/association — 关联索引查询
# ============================================================
log "cases/association"

# 1) 内联 demo 全量
OUT=$(run cases_association)
assert "assoc:demo" "$OUT" "finger -> alias -> template" && \
assert "assoc:demo" "$OUT" "CVE-2022-0001" && ok "association: inline demo"

# 2) finger 查询
OUT=$(run cases_association -finger "apache tomcat")
assert "assoc:finger" "$OUT" "inline lookup" && \
assert "assoc:finger" "$OUT" "tomcat" && ok "association: finger query"

# 3) CVE 查询
OUT=$(run cases_association -cve "CVE-2021-44228")
assert "assoc:cve" "$OUT" "inline lookup" && ok "association: CVE query"

# ============================================================
# cases/provider_filter — Provider 加载 + 过滤
# ============================================================
log "cases/provider_filter"

# 1) 本地 filter API 验证
OUT=$(run cases_provider_filter)
assert "filter:export" "$OUT" "Fingers ExportFilter:" && \
assert "filter:local" "$OUT" "FullFingers.Filter: OK" && \
assert "filter:tpl" "$OUT" "Templates.Filter (severity): OK" && ok "provider_filter: all 3 checks"

# ============================================================
# cases/match_detail — matcher 详情
# ============================================================
log "cases/match_detail"

# 1) 对真实目标
OUT=$(run cases_match_detail -target "https://httpbin.org" 2>&1)
ok "match_detail: ran against httpbin"

# 2) 对 example.com
OUT=$(run cases_match_detail -target "https://example.com" 2>&1)
ok "match_detail: ran against example.com"

# ============================================================
# cases/spray_crawl_finger — 爬虫 + 深度指纹
# ============================================================
log "cases/spray_crawl_finger"

OUT=$(run cases_spray_crawl_finger -target "https://httpbin.org" 2>&1)
ok "spray_crawl_finger: ran against httpbin"

# ============================================================
# cases/active_match — 最小主动探测 case
# ============================================================
log "cases/active_match"

OUT=$(run cases_active_match "https://httpbin.org" 2>&1)
ok "active_match: ran with embed"

# ============================================================
# Summary
# ============================================================
echo ""
log "Results: $PASS passed, $FAIL failed, $SKIP skipped"

if [ "$FAIL" -gt 0 ]; then
    echo ""
    echo "Failed:"
    for e in "${ERRORS[@]}"; do
        echo "  - $e"
    done
    exit 1
fi
