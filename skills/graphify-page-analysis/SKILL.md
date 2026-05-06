---
name: graphify-page-analysis
description: 结合代码图谱（graphifyy / CLI graphify）与工作区源码分析 Svelte/页面/CSS 依赖，支撑 Design Kit「从页面提取」前的结构化分析。
---

## 官方工具是什么（务必区分名称）

| 名称 | 含义 |
|------|------|
| **PyPI 包** | `graphifyy`（两个 y）— https://pypi.org/project/graphifyy/ |
| **命令行** | `graphify`（安装包后提供；也可用 `python -m graphify`） |
| **其它 PyPI 上叫 graphify* 的包** | 非同一作者，不要用混 |

README 摘要：

- **安装（需 Python ≥3.10）**：`pip install graphifyy && graphify install`；或用 `pipx install graphifyy` / `uv tool install graphifyy`（更易把 `graphify` 放进 PATH）。
- **Windows 额外一步**：`graphify install --platform windows`（官方表格）。
- **首次若提示找不到命令**：plain `pip` 可能未把脚本目录加入 PATH — 使用 `python -m graphify <子命令>`，或把对应 `Scripts`/`bin` 加入 PATH。
- **在工作区根目录生成图谱**（IDE 里是 `/graphify .`，终端等价）：  
  `graphify .`  
  或只扫前端：  
  `graphify ./frontend`
- **默认产物目录（约定名）**：工作区根下 **`graphify-out/`**，内含：
  - `graph.json` — 机器可读主图谱（后续 `graphify query … --graph graphify-out/graph.json`）
  - `GRAPH_REPORT.md` — 高层摘要 / god nodes / 建议问题（给人与 Agent 先读）
  - `graph.html` — 浏览器可视化
- **常用开关**：`--no-viz`（不要 HTML）、`--update`（只增量更新变更文件）、`--cluster-only`（只重跑聚类）。
- **忽略规则**：项目根 `.graphifyignore`（语法同 `.gitignore`），建议忽略 `node_modules/`、`dist/` 等。
- **代码本身**：本地 Tree-sitter，不走模型 API；文档/PDF 等才可能走 LLM（见上游 README）。

更全命令见 PyPI 文档「Common commands」与「Full command reference」。

## 何时使用

用户在原型槽 / Design Kit 场景需要：**先理解页面实际依赖与样式入口**，再决定是否调用 design-kit extract 或手写 prototypes。

## 输入

- `WORKSPACE_ROOT`：工作区绝对路径。
- `SOURCE_REL`：相对工作区的源文件路径（如 `frontend/src/routes/workbench/+page.svelte`）；**应由图谱或路由推导后再读**，不要长期依赖聊天里粘贴的过时路径。
- **图谱位置（按优先级检查）**：
  1. **`graphify-out/graph.json`**（graphify 默认输出）
  2. **`.vibex/graph/`**（团队若约定把产物拷到这里或脚本输出到此）
  3. 其它 `*.graph.json` / `graphify.json`（用户自定义）

## 流程（与上游一致 + VibeX 约束）

1. **图谱**
   - 若不存在上述图谱文件：在工作区根用 **`bash`** 执行（Windows 下 Agent 已用 Git Bash）：  
     `graphify . --no-viz` 或 `python -m graphify . --no-viz`  
     若权限/依赖失败，再提示用户本机安装：`pip install graphifyy && graphify install`。
   - **不要把图谱写到工作区外**。
   - 生成后优先 **`read_file graphify-out/GRAPH_REPORT.md`**（短），再必要时 **`read_file graphify-out/graph.json`**（可配合 `limit`），或对具体问题执行：  
     `graphify query "<自然语言问题>" --graph graphify-out/graph.json`
2. **精读源码**：用图谱或 query 结果定位到的 **`SOURCE_REL`**，再 `read_file`；用 `grep` / 图谱边追踪 import 与样式入口。
3. **输出结构化摘要**（Markdown）：
   - 页面职责与路由上下文  
   - CSS 入口（Tailwind、`<style>`、`app.css`、组件库）  
   - 与 Design Kit 对齐：`.vibex/design/DESIGN.md` 是否覆盖
4. **不写盘**：除非用户明确要求，不自动写 prototypes 或改 spec；可建议下一步用槽位工具条「剥离并写入」或 `spec_patch_apply`。

## 工具

优先：`read_file`、`bash`（仅 graphify / 短命令）、必要时上游自带的 **`graphify query`**。遵守工作区路径策略；`read_file` 的路径优先**相对工作区**。

## VibeX 与默认目录不一致时

上游默认 **`graphify-out/`**。若产品文档写 `.vibex/graph/`，二者择一即可：**要么** Agent 始终读 `graphify-out/`；**要么**团队增加一步把 `graphify-out` 复制/同步到 `.vibex/graph/`（脚本或文档约定），Agent 两者都检查即可。
