---
spec:
  level: L2
  name: prototype-extractor
  title: VibeX Prototype Extractor Browser Extension
  parent: null
  version: 0.1.0
  status: draft
  owner: vibex-team
  created: 2026-05-13

prototype:
  file: null
  note: "由 grill-me 澄清生成草稿，待补充完整 io_contract 和 behaviors"

io_contract:
  inputs:
    - name: web_page_region
      type: DOM_selection
      description: 用户在浏览器中选中的网页区块（通过点击/框选）
    - name: direction_hint
      type: string
      description: 用户对提取方向的自然语言提示（可选）
    - name: spec_context
      type: YAML_fragment
      description: 当前 spec 的 YAML 片段（用于 AI 理解上下文）
  outputs:
    - name: extracted_html
      type: file
      path: "{workspace_root}/.vibex/specs/prototypes/{spec-name}.html"
      description: 独立自包含的 HTML 原型文件
    - name: spec_yaml_fragment
      type: YAML
      description: 可合并到 spec 原型字段的 YAML 片段
  verification:
    - HTML 文件可在浏览器中独立打开并渲染
    - Markdown 文件包含提取区块的语义描述
    - YAML fragment 可与现有 spec YAML 合并

behaviors:
  - id: B1
    trigger: 用户点击网页元素
    action: agent 分析 DOM 结构 + 语义识别 → 生成提取方案
    feedback: Side Panel 展示 agent 推导的提取方向
    confirmation: 用户确认/修正/细化 → 进入下一轮
    loop: 循环直至用户说"讲清楚了"

  - id: B2
    trigger: 用户确认提取方案
    action: 提取选中区块的 DOM子树 + 计算样式
    constraints:
      - 独立自包含（内联所有 CSS，不依赖外部资源）
      - 聚焦单组件行为，不包含页面其他部分
      - 仅支持静态 HTML 页面（不支持 React/Vue/Shadow DOM 等动态渲染）

  - id: B3
    trigger: 提取完成
    action: |
      1. 生成 .html 文件 → .vibex/specs/prototypes/{name}.html
      2. 生成 YAML 片段（metadata + prototype 字段）→ clipboard 或直接写入
    side_effect: |
      原始 URL、时间戳、提取标签写入 YAML header 元数据

  - id: B4
    trigger: 用户将提取物导入 vibex-workbench
    action: |
      1. HTML 写入 .vibex/specs/prototypes/{spec-name}.html
      2. YAML fragment 合并到 spec 的 prototype 字段
      3. Workbench 预览区展示 HTML
    note: vibex-workbench 端预览能力已有，只需适配导入流程

data:
  source_url:
    type: string
    description: 被提取页面的原始 URL
  extracted_at:
    type: timestamp
    description: 提取时间
  tags:
    type: string[]
    description: 用户确认的提取标签
  dom_snapshot:
    type: string
    description: 原始 DOM 子树序列化（用于 diff 对比）

architecture:
  extension_type: Chrome Extension (Manifest V3)
  storage: IndexedDB (本地)，不涉及云端或第三方服务器
  permissions: activeTab + storage（按需声明，避免 *://*/* 全域权限）
  dynamic_rendering: 仅静态 HTML，不需要额外处理
  export_formats:
    - HTML (独立自包含)
    - YAML fragment (合并到 spec)

constraints:
  - V1 不处理源页面变更后的自动同步，用户手动重新提取
  - 不访问表单敏感字段（PII 脱敏由用户负责）
  - 所有处理在浏览器内完成，无服务端组件

out_of_scope:
  - 批量导出
  - 多设备同步
  - 非 Chrome 浏览器（V1 优先 Chrome Web Store）
