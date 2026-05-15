---
name: spec-prototype-fragment
description: |
  根据 L3/L4/L5 spec 生成派生原型片段。
  核心：标注定位 → 脚本生成 → 脚本验证 → 写盘 → 人工确认。
  触发词：生成原型片段、反推原型、spec 原型片段、原型提取。
---

# spec-prototype-fragment

规范路径：与本目录 `agent.json` 成对放置于 `.vibex/agents/skills/spec-prototype-fragment/`。

**脚本目录**：`scripts/`（生成脚本 + 验证脚本）
**模板目录**：`ref/`（WC 模板、四态配置、示例）

---

## 工作流（6步）

### Step 0：MVP 补图（前置条件）

如果 spec 描述的功能在 MVP 中没有对应 UI 区域，**不得跳过，不得从零构建**。
必须先在 MVP 原型中绘制静态占位 + 标注：

1. 找到最接近的父级区块（如 `.wb-right`、`.wb-center`）
2. 在父级内合适位置插入静态 HTML 占位
3. 添加 `data-vibex-annot` / `data-vibex-title` 标注
4. 同步更新 `vibex-workbench-mvp.yaml`（如有需要）
5. 提交 MVP 变更后，再进入 Step 1

```
### 需要补充 MVP

**spec**：`{spec.name}`
**spec.title**：`{display.title}`

该 spec 在 MVP 中没有对应 UI 区域（L3 → `.wb-right`，L4/L5 → 对应父区块）。
请描述抽屉/面板应出现在哪个位置：
1. 作为 `.wb-right` 上方独立浮层？
2. 作为 `.wb-center` 内的 tab 视图？
3. 作为 `.wb-dock` 内的面板？
4. 其他位置（请描述）：________
```

补充 MVP 后：
- 更新 `.vibex/specs/L1-goal/vibex-workbench-mvp.yaml`（如有需要）
- 提交 MVP 补图变更
- 再基于补图后的 MVP 使用技能生成派生原型

### Step 1：标注定位

在顶层 MVP 原型（`.vibex/prototypes/vibex-workbench-mvp.html`）中，
根据 spec 找到对应的 `data-vibex-annot` / `data-vibex-title` 标注区域。

**如果 MVP 中没有该区域的标注**，必须回到 Step 0——不能继续，不能用 Path B 从零构建。

不确定时询问用户，确认后记录 selector（如 `.trail`、`.statusbar`）。

### Step 2：生成脚本

根据确认的 selector，调用或编写对应的生成脚本（详见 `scripts/`）：

| 场景 | 脚本 |
|------|------|
| 有现成脚本 | 直接调用（如 `scripts/wc-gen.py --selector .statusbar`） |
| 无现成脚本 | 参考 `ref/wc-template.md` 新写脚本 |

**脚本职责（硬编码约束）**：
- 复制完整 MVP HTML（不做裁剪）
- 替换指定 selector 区域为 `<wc-{spec-name}>` web component
- 所有样式用 `var(--xxx)` 引用 MVP CSS 变量
- 四态结构（idle/open/loading/done/error）在 shadow DOM 内部
- 执行后输出派生原型文件路径

### Step 3：脚本验证

调用 `scripts/validate.py` 对生成的 HTML 进行约束校验：

```bash
python scripts/validate.py prototypes/MOD-{name}.html
```

校验通过后继续，未通过则修复脚本后重跑。

### Step 4：回写 Spec

将生成的原型文件路径回写到 spec：

```yaml
prototype:
  file: ".vibex/prototypes/MOD-{name}.html"
  validates: []
  status: derived
```

### Step 5：人工确认

向用户输出：
- 原型文件路径
- 核心交互描述（四态流程）
- "请审阅，确认后我将更新 spec"

用户确认后，提交写盘修改。

---

## 脚本说明

### scripts/wc-gen.py（通用生成器）

参数化 web component 原型生成：

```bash
python scripts/wc-gen.py \
  --spec-path .vibex/specs/L3-module/MOD-build-panel.yaml \
  --selector ".build-group" \
  --spec-name "MOD-build-panel" \
  --output .vibex/prototypes/MOD-build-panel.html
```

功能：
1. 复制完整 MVP HTML
2. 注入 WC 定义脚本（四态 panels + 状态机）
3. 替换指定 selector 为 `<wc-*>` 元素
4. 输出派生原型

### scripts/validate.py（约束校验）

```bash
python scripts/validate.py prototypes/MOD-{name}.html
```

检查项：
- MVP 全量复制（wb-root 区块完整）
- WC 定义存在（`customElements.define`）
- shadow DOM 存在（`attachShadow`）
- 四态 panels 存在（`#ph-idle`/`#ph-open`/`#ph-done`/`#ph-error`）
- 初始 hidden（所有 phase panel 含 `class="hidden"`）
- 无绝对颜色（样式用 `var(--xxx)`）

---

## 原则

0. **MVP 是唯一事实来源**——区域必须先存在于 MVP 才能派生；不存在就先画 MVP，再画派生。禁止跳过 MVP 从零构建。
1. **永远从 MVP 出发**——复制完整 HTML，不裁剪任何区块
2. **脚本即约束**——校验逻辑内嵌在脚本里，不写在 markdown 里
3. **四态在 shadow DOM 内**——`<wc-*>` 自包含全部状态机逻辑
4. **CSS 变量透传**——所有颜色/字体用 `var(--xxx)` 引用 MVP 变量
5. **人工确认闭环**——脚本跑完必须用户审阅后才能写盘

---

## 目录结构

```
spec-prototype-fragment/
├── SKILL.md               # 本文件（workflow orchestrator，Step 0–5）
├── agent.json
├── scripts/
│   ├── wc-gen.py         # 参数化原型生成脚本
│   └── validate.py       # 约束校验脚本
└── ref/
    ├── wc-template.md    # WC 模板速查
    └── example-run.md    # 完整运行示例
```
