# SPEC.md — VibeX Spec-Governance 元规格体系

**本文件是 p-metaspec 项目的 L1 goal spec。**
**使用 L1-goal-template.yaml 格式重写，保留对现有 YAML spec 文件的引用。**

---

spec:
  version: "2.0"
  level: "1_project-goal"
  name: "p-metaspec"
  parent: null                    # L1 固定 null
  status: "confirmed"

meta:
  type: "project-goal"
  owner: "hermes"
  created: "2026-04-20"
  updated: "2026-04-23"

lifecycle:
  current: "doing"
  updated: "2026-04-23"
  history:
    - status: "proposal"
      at: "2026-04-20T09:00:00"
      by: "hermes"
      trigger: "user:manual"
      note: "初始提出 p-metaspec 元规格体系"
    - status: "draft"
      at: "2026-04-20T10:00:00"
      by: "hermes"
      trigger: "agent:auto"
      note: "完成 Darwin 迭代 Round 1-3"
    - status: "confirmed"
      at: "2026-04-22T14:00:00"
      by: "hermes"
      trigger: "user:manual"
      note: "评审通过，4个 spec 均达标（L2-generator 89/L2-workflow 81/L4-constraint 84/L4-iteration 84）"
    - status: "doing"
      at: "2026-04-23T00:00:00"
      by: "hermes"
      trigger: "agent:auto"
      note: "Stage 2：用 L1 goal 模板重写 SPEC.md，准备自举闭环"

# ── io_contract ─────────────────────────────────
io_contract:
  input: |
    用户/团队需要一套可自举的 spec 治理体系——
    能用 specs 描述 specs 自身，能驱动 spec_bootstrap_pipeline.py 生成代码。
  output: |
    L1 goal spec（含 mission、constraints、product_value_layers、l1_l2_lineage）
    + 四个核心 YAML spec 文件（L2-generator / L2-workflow / L4-constraint / L4-iteration）
    + spec_bootstrap_pipeline.py（自举流水线）
  boundary: |
    L1 只描述 spec 治理体系的"是什么"和"怎么做"
    L1 不描述具体项目 spec（那是其他 L1 goal 的职责）
    L1 不描述工具实现细节（那是 L3/L4 的职责）
  behavior: |
    1. 接收用户的产品意图描述
    2. 通过 Darwin 迭代改进每个 spec 的质量
    3. 验证 spec 间跨引用一致性
    4. 用 spec_bootstrap_pipeline.py 验证自举可行性
    5. 输出完整的 spec 治理体系

# ── content（按 L1-goal-template.yaml）────────────
content:
  mission: |
    VibeX Spec-Governance（元规格体系）是一套用 specs 描述 specs 自身的自举闭环系统。
    核心目标：让 VibeX 工作台能够用自身 specs 驱动自身代码生成，
    从而证明规格体系的完整性和可验证性。
    自举闭环是验证手段，不是目标——真正的目标是让任意项目都能建立可自举的 spec 治理体系。

  target_users:
    - name: "VibeX 开发团队"
      description: "需要用 spec 驱动方式管理 vibex-workbench 项目的开发者"
    - name: "AI Agent"
      description: "需要依据 spec 生成代码的 AI Agent（通过 spec_bootstrap_pipeline.py）"
    - name: "外部项目团队"
      description: "希望复用 VibeX spec 治理范式的其他项目团队"

  constraints:
    - id: "C1"
      name: "自举闭环"
      rule: |
        p-metaspec 的 specs 必须能驱动 spec_bootstrap_pipeline.py 生成自身代码；
        自举通过即规格体系有效性的主证明。
    - id: "C2"
      name: "从属链完整"
      rule: |
        每个 spec 必须声明 parent，找不到来源 = 违规，阻断生成。
    - id: "C3"
      name: "Darwin 迭代"
      rule: |
        每个 spec 必须通过 Darwin 迭代（7维度评分 + 最低分优先改进）达到目标分数。
    - id: "C4"
      name: "跨 Spec 一致性"
      rule: |
        跨 spec 的接口（exit_code、io_contract 字段名、CLI 参数）必须一致，
        不得存在冲突。
    - id: "C5"
      name: "io_contract 必填"
      rule: |
        每个 spec 的 content 层必须包含 input/output/boundary/behavior/changelog。

  l1_l2_lineage:
    summary: |
      L1（本文档）负责把 spec 治理体系的使命、约束、成功标准写清；
      L2（skeleton 层）负责把同一套意图落成技术架构——四个核心 spec + 自举流水线。
    what_l1_owns:
      - "spec 治理体系的使命与边界"
      - "Darwin 迭代质量标准"
      - "自举闭环验收标准"
      - "四个核心 spec 的定位与关系"
    what_l2_owns:
      - "四个核心 spec（L2-generator / L2-workflow / L4-constraint / L4-iteration）的文件路径与内容"
      - "spec_bootstrap_pipeline.py 的 5 阶段流水线"
      - "Darwin 迭代引擎的实现"
      - "跨 spec 一致性验证工具"
    traceability:
      - l1_concern: "自举闭环（C1）"
        in_l2: "L2-generator.content.bootstrap_strategy + spec_bootstrap_pipeline.py"
      - l1_concern: "Darwin 迭代质量标准（C3）"
        in_l2: "L2-generator.content.bootstrap_strategy + L2-workflow W1-W6"
      - l1_concern: "跨 spec 一致性（C4）"
        in_l2: "Round 8 闭环验证结论（L2-workflow 内部 + L4-iteration）"

  product_value_layers:
    summary: |
      p-metaspec 的三层价值：① spec 治理内核 ② 自举作为体系证明 ③ 流水线作为体验。
      验收维度分离：自举通过证明「规格语言 + 管线可信」；Darwin 分数证明「各 spec 质量达标」。
    kernel_governance:
      meaning: |
        内核层：四个核心 spec（L2-generator / L2-workflow / L4-constraint / L4-iteration）
        定义了完整的 spec 治理闭环。
      maps_to_constraints: ["C1", "C2", "C3", "C4"]
    proof_self_bootstrap:
      meaning: |
        证明层：spec_bootstrap_pipeline.py 运行后 TC-001/002/003 达标（≥0.8），
        证明 specs 能驱动代码生成。
      maps_to_constraints: ["C1"]
      milestone: "Stage 3 自举验证"
    experience_visualization:
      meaning: |
        体验层：Darwin 迭代引擎的可视化评估报告 +
        spec_bootstrap_pipeline.py 的 SSE 进度流 +
        跨 spec 一致性验证报告。
      maps_to_constraints: ["C3", "C4"]

  spec_lifecycle:
    summary: "每个 spec 必须包含 spec.status + lifecycle 字段"
    status_values:
      - proposal
      - draft
      - review
      - delicate
      - confirmed
      - todo
      - doing
      - conditional-done
      - done
      - abandoned

  io_contract_global:
    summary: "所有 spec（L1–L5）的 content 层必须包含以下字段"
    fields:
      - name: input
        description: "该 spec 的输入条件"
        required: true
      - name: output
        description: "该 spec 的输出结果"
        required: true
      - name: boundary
        description: "边界条件"
        required: true
      - name: behavior
        description: "该 spec 内部行为"
        required: true
      - name: changelog
        description: "变更日志"
        required: true

  routing_paths:
    spec_first_iteration:
      trigger: "用户描述一个需要通过 spec-first 方式实现的变更"
      route_to: "spec_bootstrap_pipeline.py"
      clarification_loop: |
        S1 澄清 → spec snippet 生成 → S2 io_contract 补全
        → S3 behavior 补全 → S4 代码生成 → S5 验证
      output: |
        生成的代码文件 + eval 结果（TC-001/002/003 评分）
      output_confirmed: |
        所有 TC 评分 ≥ 0.8，pipeline 输出 done

  goal_spec_visualization:
    summary: |
      p-metaspec 在 Canvas 上展示为四个核心 spec 的卡片网格，
      每个卡片显示 spec 名称、当前 Darwin 分数、状态。
      点击卡片可钻取到对应 YAML 文件。
    center_card:
      role: "锚定 spec 治理体系"
      naming_and_tagline: |
        名称：VibeX Spec-Governance
        标语：用 specs 描述 specs 自身，实现自举闭环
    aspect_cards:
      - id: "generator"
        label: "代码生成机制"
        default_drill_path: "L2-generator-gen-mechanism.yaml"
      - id: "workflow"
        label: "spec-first 流程"
        default_drill_path: "L2-workflow-spec-first-iteration.yaml"
      - id: "constraint"
        label: "技术约束规范"
        default_drill_path: "L4-constraint-goa-style.yaml"
      - id: "iteration"
        label: "迭代与漂移检测"
        default_drill_path: "L4-iteration-trigger-paths.yaml"

# ── 四个核心 spec 索引 ─────────────────────────────
# 这些文件是 L2/L4 层级的 spec，L1 只做索引引用
# 详细内容见各 YAML 文件
core_specs:
  - id: "L2-generator"
    name: "L2-generator-gen-mechanism"
    path: "L2-generator-gen-mechanism.yaml"
    level: "2_skeleton"
    parent: "p-metaspec"
    darwin_score: 89
    status: "done"
    summary: "gen.py 自身机制规范（自举三阶段策略 + 模板变量 + 循环依赖破解）"

  - id: "L2-workflow"
    name: "L2-workflow-spec-first-iteration"
    path: "L2-workflow-spec-first-iteration.yaml"
    level: "2_skeleton"
    parent: "p-metaspec"
    darwin_score: 81
    status: "done"
    summary: "spec-first 6 阶段工作流（W1 Clarify → W6 Iterate）"

  - id: "L4-constraint"
    name: "L4-constraint-goa-style"
    path: "L4-constraint-goa-style.yaml"
    level: "4_feature"
    parent: "p-metaspec"
    darwin_score: 84
    status: "done"
    summary: "goa 风格技术约束声明规范（安全约束 + 跨端点不变量）"

  - id: "L4-iteration"
    name: "L4-iteration-trigger-paths"
    path: "L4-iteration-trigger-paths.yaml"
    level: "4_feature"
    parent: "p-metaspec"
    darwin_score: 84
    status: "done"
    summary: "迭代决策 + 漂移检测 + 消歧规则"

# ── Darwin 评估状态 ───────────────────────────────
darwin_evaluation:
  summary: "四个核心 spec 均通过 Darwin 迭代达标（≥80 分）"
  rounds:
    - round: 3
      baseline: true
      scores:
        L2-generator: 65
        L2-workflow: 61
        L4-constraint: 76
        L4-iteration: 77
    - round: 4
      scores:
        L2-workflow: 72
    - round: 5
      scores:
        L4-constraint: 84
        L4-iteration: 84
    - round: 6
      scores:
        L2-workflow: 81
    - round: 7
      scores:
        L2-generator: 89
        L4-iteration: 84
    - round: 8
      scores:
        L4-iteration: 84
      note: "跨 spec 闭环验证，修复 6 处冲突"
    - round: 9
      note: "最终确认，所有 spec 达标"
      final_scores:
        L2-generator: 89
        L2-workflow: 81
        L4-constraint: 84
        L4-iteration: 84

# ── 跨 Spec 闭环验证（Round 8）────────────────────
cross_spec_validation:
  summary: "Round 8 共发现并修复 6 处跨 spec 冲突"
  conflicts:
    - id: 1
      type: "exit_code 语义分裂（W2）"
      specs: ["L2-workflow 内部"]
      fix: "WF-007: 1→2；WF-008: 2→3"
    - id: 2
      type: "exit_code 语义分裂（W4）"
      specs: ["L2-workflow 内部"]
      fix: "同上"
    - id: 3
      type: "exit_code 语义分裂（全局索引）"
      specs: ["L2-workflow 内部"]
      fix: "同上"
    - id: 4
      type: "exit_code 标签错误（全局依赖表）"
      specs: ["L2-workflow 内部"]
      fix: "补全 exit_code=1/4 语义"
    - id: 5
      type: "CLI 接口不匹配"
      specs: ["L2-workflow", "L2-generator"]
      fix: "更新全局依赖定义与 G1.0 对齐"
    - id: 6
      type: "io_contract 命名不一致"
      specs: ["L4-iteration 内部"]
      fix: "`inputs`→`input`；`outputs`→`output`"
  closure_verification:
    - "L2-workflow W4 → L2-generator G1.0：CLI 参数完全对齐 ✅"
    - "L2-workflow W6 → L4-iteration I6.1：输入/输出字段一一对应 ✅"
    - "L4-iteration decision → L2-workflow W6：全部 6 种决策类型匹配 ✅"
    - "parent 链路：4 个 spec 全部声明 parent: p-metaspec ✅"

# ── 补全前后对比 ─────────────────────────────────
gap_analysis:
  before_after:
    - dimension: "Spec 自身可描述"
      before: "❌"
      after: "✅ L2-generator（89分）"
    - dimension: "Spec 执行流程可追溯"
      before: "❌"
      after: "✅ L2-workflow（81分）"
    - dimension: "技术约束可自动验证"
      before: "❌"
      after: "✅ L4-constraint（84分）"
    - dimension: "迭代路径可预测"
      before: "❌"
      after: "✅ L4-iteration（84分）"
    - dimension: "Spec 可自举生成"
      before: "❌"
      after: "✅ 四者形成闭环"
    - dimension: "跨 Spec 接口一致性"
      before: "❌"
      after: "✅ 6 处冲突全部修复"

# ── 下轮迭代建议 ─────────────────────────────────
next_iteration:
  - priority: "low"
    item: "L2-workflow：补全 backlog 决策类型到 W6.io_contract"
  - priority: "low"
    item: "L4-constraint：实现 constraint_validator.py（目前仅规范接口）"
  - priority: "low"
    item: "L2-generator：Stage1 手工引导的分步验收流程"
  - priority: "low"
    item: "L4-iteration：动态优先级权重（weight 因子）定义"
  - priority: "medium"
    item: "全体：建立 CI gate，自动验证所有 4 个 spec 的跨引用一致性"
  - priority: "high"
    item: "Stage 3 自举验证：运行 spec_bootstrap_pipeline.py，验证 TC-001/002/003 ≥ 0.8"

changelog:
  - version: "2.0"
    date: "2026-04-23"
    author: "hermes"
    changes:
      - "用 L1-goal-template.yaml 格式重写 SPEC.md"
      - "新增 YAML frontmatter（spec.level = 1_project-goal）"
      - "新增 io_contract（L1 职责边界）"
      - "新增 product_value_layers（三层价值分离）"
      - "新增 l1_l2_lineage（L1/L2 分工明确）"
      - "新增 core_specs 索引（四个核心 YAML 文件）"
      - "保留 Darwin 评估状态和 Round 8 跨 spec 闭环验证结论"
      - "新增 next_iteration（Stage 3 自举验证优先）"
  - version: "1.0"
    date: "2026-04-22"
    author: "hermes"
    changes:
      - "初始版本：Markdown 格式，描述四个核心 spec 的关系与 Darwin 评估结果"

---

## 非 YAML 层：元层级架构图

> 以下图表是 Markdown 可读性补充，不参与机器解析。

```
                    ┌─────────────────────────────┐
                    │  L0: 元规格（spec-templates）│
                    │  定义 spec 系统自身          │
                    └──────────────┬──────────────┘
                                   │ 从属
           ┌───────────────────────┼───────────────────────┐
           ↓                       ↓                       ↓
┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│ L2-generator     │  │ L2-workflow       │  │ L4-constraint    │
│ gen.py 机制规范   │  │ spec-first 流程   │  │ goa 风格验证约束 │
│ 自举闭环核心      │  │ 6 阶段 gate      │  │ 声明式约束嵌入   │
│ Darwin: 89 ✅    │  │ Darwin: 81 ✅   │  │ Darwin: 84 ✅   │
└────────┬─────────┘  └────────┬─────────┘  └────────┬─────────┘
         │                     │                     │
         └─────────────────────┼─────────────────────┘
                               ↓
              ┌──────────────────────────────┐
              │ L4-iteration                 │
              │ 迭代触发路径 + 漂移检测       │
              │ Darwin: 84 ✅                │
              └──────────────────────────────┘
```

## spec_bootstrap_pipeline.py 自举流水线

```
用户意图
    ↓
S1 澄清（Clarify）→ ClarificationPanel UI
    ↓ spec snippet（含 io_contract）
S2 补全 io_contract（Spec Snippet → Improved Snippet）
    ↓ spec snippet with io_contract
S3 补全 behavior（Improved Snippet → Detailed Spec）
    ↓ detailed spec with behaviors
S4 生成代码（Detailed Spec → Artifacts）
    ↓ artifacts
S5 验证（Artifacts → Eval Result）
    ↓ eval result
TC-001 ≥ 0.8？否 → 回到 S2（重新补全 io_contract）
TC-001 ≥ 0.8？是 → done
```
