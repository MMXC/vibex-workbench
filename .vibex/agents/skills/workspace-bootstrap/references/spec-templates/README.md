# VibeX Spec-Templates

**自举工程的种子：L0–L5 层级化 spec 模板库。**

---

## 概述

`spec-templates/` 是 VibeX spec 体系的自举种子。
它用 specs 描述 spec 体系本身，打破"先有 spec 才能生成代码"的循环依赖。

## 层级心智：收敛与发散（L1–L5）

各层在纵向链路里承担的「职责形状」可以概括为：**奇数层偏收敛，偶数层偏发散**（L5 仍是面向落地的收敛）。编写或评审 spec 时用来判断「这些内容是否出现在对的层级」：

| 层级 | 形状 | 一句话 |
|:---:|---|---|
| **L1** goal | **收敛** | 把愿景、边界、成功标准收成一根「准绳」，不包含实现细节。 |
| **L2** skeleton | **发散** | 在总目标下展开**技术骨架**（模块划分、依赖、边界），达成「长什么样」的全景。 |
| **L3** module | **收敛** | 对 L2 骨架里的**每个模块**收敛职责、对外契约与边界，避免模块间含糊。 |
| **L4** feature | **发散** | 在每个模块内展开**功能与行为**（场景、验收、与外界的交互），允许产品线级发散。 |
| **L5** slice | **收敛实现** | 把每个功能收敛为**可交付切片**（文件/改动点/验证步骤），直接对齐代码与 agent 任务。 |

记忆口诀：**L1 收口径 → L2 展骨架 → L3 收模块 → L4 展功能 → L5 收到实现**。模板正文里的 `io_contract` / `boundary` 措辞应与该心智一致；不要把 L5 才该有的路径细节塞进 L1。

## 产品分层心智（壳 / 能力 / 实现）

规划页与分层评审时，除「收敛·发散」外，统一采用 **产品视角** 刻画 L3/L4/L5。**权威定义与拆分启发式**见 [`L0-meta-convention/L0-layer-contract-template.yaml`](./L0-meta-convention/L0-layer-contract-template.yaml) 中的 `content.product_layering_governance`。

| 层级 | 产品视角（摘要） |
|:---:|---|
| **L3** | 打开应用后用户可见的**主模块 / 主分区**（壳级 UI），不写单次点击后的完整闭环。 |
| **L4** | 从 L3 分区上的控件触发的**一条完整用户能力**（行为 + 验收）；多级弹窗或并列子流程宜**拆成多条 L4**。 |
| **L5** | 该能力的**实现切片**（文件、接口、生成规则、验证节点等）。 |

**可达性**：目标为少量点击（例如三步内）从主界面到达任意已声明能力。**Spec 插槽 / 抽屉**：仅占容器可在 L5 占位；一旦填入可点的具体能力，应另立 **L4** 描述端到端行为，再由 **L5** 承接实现细节。

## 新工作区初始化（打卡工作区）

从空仓库按同一套路拉升 spec 时，采用 **L0 `content.workspace_bootstrap_contract`** 与各层模板内的 **`content.workspace_bootstrap`** 占位块：

| 层 | 心智（摘要） |
|:---:|---|
| **L1** | 总目标收敛；交付 **主界面静态原型**，经 **原型槽**过关后进入 L2 |
| **L2** | 输入 L1+主原型 → **可初始化、可打包可运行**；多工程则 **多条 L2、单开发入口**（如 `make wails-dev`） |
| **L3** | 前端自 L1 拆 **可见区 MOD**；后端按 **分层**写规范+验证工具；后端 MOD **可滞后**、由 L4 反推 |
| **L4** | 前端=功能+**完整交互原型**；后端适配层与前端 FEAT **mixin/配对**，产出 **通用插槽契约** |
| **L5** | 每 L4 的落地切片；**红绿/强校验**；后端为业务代码 **嵌入** L4 插槽 |

权威条文见 [`L0-meta-convention/L0-layer-contract-template.yaml`](./L0-meta-convention/L0-layer-contract-template.yaml) 中 **`workspace_bootstrap_contract`**；拷贝 [`L1-goal`](./L1-goal/L1-goal-template.yaml)～[`L5-slice`](./L5-slice/L5-slice-template.yaml) 模板填空即可初始化。

**自动化（vibex-workbench 内）**：`/workspace` 页「**Agent 初始化 Spec（L1–L5）**」走 `workspace-bootstrap` skill execute（`.agents/skills/workspace-bootstrap/scripts/execute.py`），Go agent 工具为 **`workspace_specs_bootstrap`**（`confirm: true`）；仅在 skill 缺失时回退 `generators/spec_workspace_bootstrap.py`。打包运行需能解析到 vibex-workbench 根目录（如设置 `VIBEX_WORKBENCH_ROOT`）。

## 目录结构

```
spec-templates/
├── README.md                        ← 本文件
├── L0-meta-convention/
│   ├── README.md
│   ├── L0-layer-contract-template.yaml      # L1–L5 每层 must_contain
│   ├── L0-directory-convention-template.yaml # 目录↔层级↔命名映射
│   └── L0-mixin-system-template.yaml        # mixin 定义/合并/循环检测
├── L1-goal/
│   ├── README.md
│   └── L1-goal-template.yaml                # 项目目标层
├── L2-skeleton/
│   ├── README.md
│   └── L2-skeleton-template.yaml            # 技术架构骨架
├── L3-module/
│   ├── README.md
│   └── L3-module-template.yaml              # 模块边界 + 公开 API
├── L4-feature/
│   ├── README.md
│   └── L4-feature-template.yaml             # 功能行为 + 验收标准
└── L5-slice/
    ├── README.md
    └── L5-slice-template.yaml               # 具体文件生成规格
```

## 对照 goskill 的启示

goskill（AIPMAndy/goskill）是一个用 Go+DSL 实现的技能框架，
它的 spec.yaml 采用纯 YAML 格式，核心特点：

| goskill 特色 | VibeX 对应 | 自举启示 |
|---|---|---|
| `context` 定义输入输出 | L1–L5 `io_contract` | **io_contract 必须放在 YAML frontmatter**，不要放 Markdown body |
| `validation` 场景测试 | L4 `test_scenarios` | TC 评分驱动 spec 迭代 |
| goa DSL 结构化表达 | L4 `behaviors` | behaviors 是代码生成的直接输入 |
| `generator` 自举 | gen.py 自举 | Stage 1 手工 → Stage 2 自举 → Stage 3 验证 |
| package 分层（cmd/internal/pkg） | L2/L3/L4 层级 | 模块边界在 L2 定义，L3/L4 展开 |

### 关键区别

- **goskill**：纯 YAML，100% 机器可解析，0% 人类友好
- **VibeX 方案C**：YAML frontmatter（机器解析）+ Markdown body（人类可读）
- **VibeX 优势**：开发者能直接读 Markdown，不依赖工具也能理解 spec
- **VibeX 要求**：YAML frontmatter 必须能被 `yaml.safe_load` 解析（禁止在 frontmatter 里放代码块）

## 自举路径

```
Stage 1（手工）：用 spec-templates 手工写 p-metaspec 的 L1–L5 specs
     ↓
Stage 2（自举）：gen.py 读取 spec-templates → 生成新的 *.Skeleton.svelte
     ↓
Stage 3（验证）：spec_bootstrap_pipeline.py TC-001/002/003 ≥ 0.8
```

Stage 1 的核心：手工把 `L2-generator-gen-mechanism.yaml` 等文件
迁移成 `.md` 格式（YAML frontmatter + Markdown body），
io_contract 必须进 frontmatter。

## 使用方法

### 创建新项目

```bash
# 1. 在 project-catalog.json 注册
# 2. 创建项目目录
mkdir -p projects/my-project/
# 3. 复制模板
cp spec-templates/L1-goal/L1-goal-template.yaml projects/my-project/SPEC.md
# 4. 填写模板（替换 {} 占位符）
```

### 验证 spec 格式

```bash
python3 ~/.hermes/spec-governance/scripts/validate_specs.py \
  projects/my-project/SPEC.md
# exit_code: 0=通过, 2=格式错误
```

## Canonical 槽位规则

所有 L1–L5 spec 都应在顶层包含以下 canonical 槽位，供图谱、列表、详情面板和 agent context 统一读取：

```yaml
display:
  title: "中文短标题"
  summary: "一句话说明"
  description: "更完整说明"

structure:
  parent: ""
  children: []
  dependencies: []
  impacted_files: []

io:
  input: []
  output: []
  boundary: ""

constraints:
  rules: []
  forbidden: []

prototype:
  file: ""
  validates: []
  status: none
```

填写规则：

- `display` 面向人类阅读，默认使用中文，由 agent 在创建或修改 spec 时同步维护。
- `structure` 描述图谱结构，`parent` 应与 `spec.parent` 保持一致，`impacted_files` 用于实现影响面展示。
- `io` 是新的输入输出权威字段；旧 `io_contract` 保留用于兼容和迁移，但 UI 优先读取 `io`。
- `constraints` 是新的约束权威字段；旧 `content.constraints` 保留用于兼容和迁移。
- `prototype.status` 可取 `none`、`draft`、`reviewed`、`implemented`。没有原型时写 `none`，不要省略字段。

缺省语义：

- 明确没有内容：写空数组、空字符串或 `status: none`，UI 显示“无”。
- 当前层级不适用：写空值，并在描述中说明“不适用”，UI 显示“不适用”。
- 应该补但尚未补：保留空值，UI 显示“待补充”。

### 迁移现有 YAML 文件

```python
# 从混合 YAML+Markdown → 方案C Markdown+YAML frontmatter
# 参考: L2-workflow 迁移脚本
```

## 模板更新规则

- L0 层变更 → 需要 Darwin 迭代验证（改动代价极高）
- L1–L5 层变更 → 通过 `spec-first` 流水线验证后再合并
- 模板路径固定：`~/.hermes/spec-governance/spec-templates/`
