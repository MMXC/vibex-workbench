// Package designkit implements workspace design gate files (.vibex/design, .vibex/prototypes)
// per FEAT-workspace-design-prototype-gate. Used by SvelteKit handlers and Wails App bindings.
package designkit

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	DesignPathRel     = ".vibex/design/DESIGN.md"
	PrototypesPathRel = ".vibex/prototypes"
	MaxSourceBytes    = 600_000
)

const designTemplate = `# VibeX 设计物料库（DESIGN）

> **门禁**：Agent 生成原型、页面与样式时必须遵循本节；禁止脱离本仓库真实栈与样式约定「自由发挥」导致 UI 漂移。

## 技术栈

- 前端框架：（填写，如 SvelteKit 5）
- 样式方案：（填写，如 CSS / Tailwind / 设计令牌文件路径）
- 组件库：（如有）

## 设计令牌与主题

- 主色 / 辅色 / 背景：
- 圆角、间距、字号阶梯：
- 暗色模式：（是/否，关键变量）

## 布局与组件约定

- 页面栅格、最大宽度、区域命名：
- 可复用组件目录：（如 ` + "`frontend/src/lib/components/...`" + `）
- **禁止**：未在此声明的随机色值、与设计稿无关的字体栈

## 原型物料输出

- 可交付 HTML 草模归档目录：` + "`.vibex/prototypes/`" + `
- 每个 spec 在 ` + "`prototype.file`" + ` 中引用相对工作区根的物料路径（推荐放在 ` + "`.vibex/prototypes/`" + `）

## 从现有页面剥离

- 使用 Workbench 原型槽内「从页面提取」生成初版物料；再由人工或 Agent 收紧为与生产代码一致的样式子集。

`

const readmeProto = `# 原型物料库

本目录存放 **与真实栈对齐** 的 HTML/静态片段，用于 spec 原型槽校验与评审。

- 由 ` + "`.vibex/design/DESIGN.md`" + ` 约束样式与组件边界。
- 命名建议：` + "`<feature>-<variant>.html`" + `，并在 manifest 中登记。

`

const manifestTemplate = `{
  "version": 1,
  "entries": []
}
`

var (
	scriptTagRe = regexp.MustCompile(`(?is)<script\b[^>]*>.*?</script>`)
	onAttrRe    = regexp.MustCompile(`(?i)\s+on\w+\s*=\s*(?:"[^"]*"|'[^']*'|[^\s>]+)`)
)

// WorkspaceRelSafe joins root with a relative workspace path and ensures result stays under root.
func WorkspaceRelSafe(root, rel string) (abs string, err error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if rel == "" {
		return "", fmt.Errorf("empty path")
	}
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("absolute source path not allowed")
	}
	clean := filepath.Clean(rel)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes workspace")
	}
	full := filepath.Join(rootAbs, clean)
	fullAbs, err := filepath.Abs(full)
	if err != nil {
		return "", err
	}
	relOut, err := filepath.Rel(rootAbs, fullAbs)
	if err != nil || relOut == ".." || strings.HasPrefix(relOut, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path outside workspace")
	}
	return fullAbs, nil
}

// Status describes design kit files on disk.
type Status struct {
	Ok                  bool   `json:"ok"`
	WorkspaceRoot       string `json:"workspaceRoot,omitempty"`
	DesignMdExists      bool   `json:"designMdExists"`
	PrototypesDirExists bool   `json:"prototypesDirExists"`
	DesignPath          string `json:"designPath"`
	PrototypesPath      string `json:"prototypesPath"`
	Error               string `json:"error,omitempty"`
}

// GetStatus returns whether DESIGN.md and prototypes dir exist.
func GetStatus(root string) Status {
	root = strings.TrimSpace(root)
	if root == "" {
		return Status{Ok: false, Error: "workspaceRoot required", DesignPath: DesignPathRel, PrototypesPath: PrototypesPathRel}
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return Status{Ok: false, Error: err.Error(), DesignPath: DesignPathRel, PrototypesPath: PrototypesPathRel}
	}
	if _, err := os.Stat(rootAbs); err != nil {
		return Status{Ok: false, Error: "workspace not found", DesignPath: DesignPathRel, PrototypesPath: PrototypesPathRel}
	}
	designAbs := filepath.Join(rootAbs, filepath.FromSlash(DesignPathRel))
	protosAbs := filepath.Join(rootAbs, filepath.FromSlash(PrototypesPathRel))
	_, derr := os.Stat(designAbs)
	_, perr := os.Stat(protosAbs)
	return Status{
		Ok:                  true,
		WorkspaceRoot:       rootAbs,
		DesignMdExists:      derr == nil,
		PrototypesDirExists: perr == nil,
		DesignPath:          strings.ReplaceAll(DesignPathRel, `\`, `/`),
		PrototypesPath:      strings.ReplaceAll(PrototypesPathRel, `\`, `/`),
	}
}

// ScaffoldResult is the outcome of scaffold (confirm=true).
type ScaffoldResult struct {
	Ok          bool               `json:"ok"`
	Written     []string           `json:"written,omitempty"`
	Skipped     []string           `json:"skipped,omitempty"`
	Error       string             `json:"error,omitempty"`
	GateBlocked bool               `json:"gateBlocked,omitempty"`
	GateFailure *GateFailureDetail `json:"gateFailure,omitempty"`
}

// Scaffold creates missing .vibex/design/DESIGN.md, prototypes README, manifest. Never overwrites DESIGN.md.
// When specYAML is non-empty and confirm is true, EvaluatePrototypeGateYAML must pass before writes.
func Scaffold(root string, confirm bool, specYAML string) ScaffoldResult {
	if !confirm {
		return ScaffoldResult{Ok: false, Error: "需确认：传入 confirm: true 后才会写入 .vibex/design 与 .vibex/prototypes"}
	}
	if strings.TrimSpace(specYAML) != "" {
		ok, det := EvaluatePrototypeGateYAML([]byte(specYAML))
		if !ok {
			return ScaffoldResult{
				Ok: false, Error: "prototype_gate_blocked", GateBlocked: true, GateFailure: det,
			}
		}
	}
	root = strings.TrimSpace(root)
	if root == "" {
		return ScaffoldResult{Ok: false, Error: "workspace_root required"}
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return ScaffoldResult{Ok: false, Error: err.Error()}
	}
	if st, err := os.Stat(rootAbs); err != nil || !st.IsDir() {
		return ScaffoldResult{Ok: false, Error: "workspace not found"}
	}

	designDir := filepath.Join(rootAbs, ".vibex", "design")
	protoDir := filepath.Join(rootAbs, ".vibex", "prototypes")
	designFile := filepath.Join(designDir, "DESIGN.md")
	readmeFile := filepath.Join(protoDir, "README.md")
	manifestFile := filepath.Join(protoDir, "manifest.json")

	var written, skipped []string
	if err := os.MkdirAll(designDir, 0755); err != nil {
		return ScaffoldResult{Ok: false, Error: err.Error()}
	}
	if err := os.MkdirAll(protoDir, 0755); err != nil {
		return ScaffoldResult{Ok: false, Error: err.Error()}
	}

	if _, err := os.Stat(designFile); os.IsNotExist(err) {
		if err := os.WriteFile(designFile, []byte(designTemplate), 0644); err != nil {
			return ScaffoldResult{Ok: false, Error: err.Error()}
		}
		written = append(written, ".vibex/design/DESIGN.md")
	} else {
		skipped = append(skipped, ".vibex/design/DESIGN.md")
	}

	if _, err := os.Stat(readmeFile); os.IsNotExist(err) {
		if err := os.WriteFile(readmeFile, []byte(readmeProto), 0644); err != nil {
			return ScaffoldResult{Ok: false, Error: err.Error()}
		}
		written = append(written, ".vibex/prototypes/README.md")
	} else {
		skipped = append(skipped, ".vibex/prototypes/README.md")
	}

	if _, err := os.Stat(manifestFile); os.IsNotExist(err) {
		if err := os.WriteFile(manifestFile, []byte(manifestTemplate), 0644); err != nil {
			return ScaffoldResult{Ok: false, Error: err.Error()}
		}
		written = append(written, ".vibex/prototypes/manifest.json")
	} else {
		skipped = append(skipped, ".vibex/prototypes/manifest.json")
	}

	return ScaffoldResult{Ok: true, Written: written, Skipped: skipped}
}

func stripScriptTags(html string) string {
	s := scriptTagRe.ReplaceAllString(html, "")
	s = onAttrRe.ReplaceAllString(s, "")
	return s
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}

func buildExtractedDocument(title, sourcePath, previewHTML, sourceExcerpt string) string {
	return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1"/>
<title>` + escapeHTML(title) + `</title>
<style>
  body { font-family: system-ui, Segoe UI, sans-serif; margin: 0; background: #0e1016; color: #e2e8f0; line-height: 1.45; }
  header { padding: 12px 16px; border-bottom: 1px solid #303746; background: rgba(28,32,42,.9); }
  h1 { font-size: 14px; margin: 0; color: #7aa2ff; }
  .meta { font-size: 11px; color: #94a3b8; margin-top: 6px; word-break: break-all; }
  main { padding: 16px; }
  .preview { border: 1px solid #303746; border-radius: 12px; padding: 12px; background: rgba(28,32,42,.5); min-height: 80px; }
  pre.src { margin-top: 16px; padding: 12px; border-radius: 10px; background: #0a0c10; border: 1px solid #242b38; overflow: auto; font-size: 11px; color: #a3abb9; white-space: pre-wrap; }
</style>
</head>
<body>
<header>
  <h1>VibeX 剥离原型 · 物料预览</h1>
  <div class="meta">源：` + escapeHTML(sourcePath) + `</div>
  <div class="meta">自上而下：可渲染预览（已剔 script / 行内事件）与源摘录；请按 .vibex/design/DESIGN.md 对齐真实栈样式。</div>
</header>
<main>
  <div class="preview">` + previewHTML + `</div>
  <h2 class="meta" style="margin:16px 0 8px">源摘录</h2>
  <pre class="src">` + escapeHTML(sourceExcerpt) + `</pre>
</main>
</body>
</html>`
}

var bodyInnerRe = regexp.MustCompile(`(?is)<body\b[^>]*>([\s\S]*)</body>`)

// ExtractResult is returned by Extract.
type ExtractResult struct {
	Ok          bool   `json:"ok"`
	WrittenPath string `json:"writtenPath,omitempty"`
	SpecSnippet string `json:"specSnippet,omitempty"`
	SourcePath  string `json:"sourcePath,omitempty"`
	Error       string `json:"error,omitempty"`
	GateBlocked bool               `json:"gateBlocked,omitempty"`
	GateFailure *GateFailureDetail `json:"gateFailure,omitempty"`
}

func sanitizeBasename(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	out := strings.TrimSuffix(b.String(), ".html")
	out = strings.TrimSuffix(strings.TrimSuffix(out, ".HTML"), ".HTM")
	return out
}

func slugFromSourceBase(path string, ext string) string {
	base := strings.TrimSuffix(filepath.Base(path), ext)
	var b strings.Builder
	lastDash := false
	for _, r := range base {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if b.Len() > 0 && !lastDash {
			b.WriteRune('-')
			lastDash = true
		}
	}
	s := strings.Trim(b.String(), "-")
	if s == "" {
		return "extracted"
	}
	if len(s) > 80 {
		s = s[:80]
	}
	return s
}

// Extract reads a workspace-relative source file and writes a sandbox-safe HTML under .vibex/prototypes.
func Extract(root string, sourceRel string, outBasename string, confirm bool, specYAML string) ExtractResult {
	if !confirm {
		return ExtractResult{Ok: false, Error: "需确认：传入 confirm: true 写入 .vibex/prototypes"}
	}
	if strings.TrimSpace(specYAML) != "" {
		ok, det := EvaluatePrototypeGateYAML([]byte(specYAML))
		if !ok {
			return ExtractResult{
				Ok: false, Error: "prototype_gate_blocked", GateBlocked: true, GateFailure: det,
			}
		}
	}
	root = strings.TrimSpace(root)
	sourceRel = strings.TrimSpace(sourceRel)
	if root == "" {
		return ExtractResult{Ok: false, Error: "workspace_root invalid"}
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil || sourceRel == "" {
		if sourceRel == "" {
			return ExtractResult{Ok: false, Error: "source_path required (relative to workspace)"}
		}
		return ExtractResult{Ok: false, Error: "workspace_root invalid"}
	}
	if _, err := os.Stat(rootAbs); err != nil {
		return ExtractResult{Ok: false, Error: "workspace_root invalid"}
	}

	sourceAbs, err := WorkspaceRelSafe(rootAbs, sourceRel)
	if err != nil {
		return ExtractResult{Ok: false, Error: "forbidden: path traversal"}
	}

	st, err := os.Stat(sourceAbs)
	if err != nil || !st.Mode().IsRegular() {
		return ExtractResult{Ok: false, Error: "source file not found"}
	}
	if st.Size() > MaxSourceBytes {
		return ExtractResult{Ok: false, Error: fmt.Sprintf("file too large (max %d bytes)", MaxSourceBytes)}
	}

	raw, err := os.ReadFile(sourceAbs)
	if err != nil {
		return ExtractResult{Ok: false, Error: err.Error()}
	}

	ext := strings.ToLower(filepath.Ext(sourceAbs))
	baseName := sanitizeBasename(outBasename)
	if baseName == "" {
		baseName = slugFromSourceBase(sourceAbs, ext)
	}

	outDirAbs := filepath.Join(rootAbs, filepath.FromSlash(PrototypesPathRel))
	if relCheck, err := filepath.Rel(rootAbs, outDirAbs); err != nil || relCheck == ".." || strings.HasPrefix(relCheck, "..") {
		return ExtractResult{Ok: false, Error: "internal path error"}
	}
	if err := os.MkdirAll(outDirAbs, 0755); err != nil {
		return ExtractResult{Ok: false, Error: err.Error()}
	}

	outName := baseName + "-extracted.html"
	outAbs := filepath.Join(outDirAbs, outName)
	sourceRelSlash := strings.ReplaceAll(filepath.ToSlash(sourceRel), `\`, `/`)

	previewHTML := ""
	if ext == ".html" || ext == ".htm" {
		cleaned := stripScriptTags(string(raw))
		sub := bodyInnerRe.FindStringSubmatch(cleaned)
		if len(sub) > 1 {
			previewHTML = strings.TrimSpace(sub[1])
		} else {
			previewHTML = strings.TrimSpace(cleaned)
		}
		if previewHTML == "" {
			previewHTML = `<p class="meta">（空主体）</p>`
		}
	} else {
		previewHTML = `<p class="meta">非 HTML 源文件：预览区仅展示占位；请以源摘录为准在物料中重建 markup。</p>`
	}

	excerpt := string(raw)
	if len(excerpt) > 120_000 {
		excerpt = excerpt[:120_000] + "\n… [truncated]"
	}

	doc := buildExtractedDocument(baseName, sourceRelSlash, previewHTML, excerpt)
	if err := os.WriteFile(outAbs, []byte(doc), 0644); err != nil {
		return ExtractResult{Ok: false, Error: err.Error()}
	}

	relOut := strings.ReplaceAll(filepath.ToSlash(filepath.Join(PrototypesPathRel, outName)), `\`, `/`)
	snippet := fmt.Sprintf("prototype:\n  file: %s\n  status: draft", relOut)
	return ExtractResult{Ok: true, WrittenPath: relOut, SpecSnippet: snippet, SourcePath: sourceRelSlash}
}
