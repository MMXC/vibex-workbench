---
name: workbench-canvas-ui
description: Workbench Canvas UI Skill — use fixed static HTML components + JSON payload to render interactive canvas cards (confirm/cancel/request/actions) for users. Prefer this when agent needs to present structured operations in UI instead of plain chat text.
---

# workbench-canvas-ui

面向 Workbench 的“模板化可操作画布 UI”技能。核心原则：

- **先选模板，再填数据**（而不是临时生成 UI 代码）
- **组件静态 HTML**，通过 `data-bind` 注入 JSON
- **动作统一协议**（确认框、取消、请求按钮等）
- **允许 agent 扩展组件库**（新增组件并登记）

## 何时使用

当用户希望：

- 在中间画布展示 agent 的结构化输出
- 用按钮/确认框来驱动下一步操作
- 把聊天内容变成可交互卡片（流程、校验、补链、执行）

## 标准工作流

1. 选择模板（见 `templates/*.json`）
2. 生成 `CanvasSkillPayload`（见 `references/payload-schema.md`）
3. 使用 `assets/runtime.js` 注入到组件 HTML（`components/*.html`）
4. 触发 actions（`prefill_composer` / `request_api` / `open_slot` / `pin_canvas`）

## 强约束

- 不直接输出“自由格式 UI”；必须走模板/组件库。
- 所有写入类 action 必须有 `confirm` 文案。
- 若 action 涉及路径，必须展示 `workspace_root` 与目标路径。
- 组件只允许静态 HTML + CSS class + `data-bind` 占位，不嵌业务逻辑。

## 允许 agent 增加组件（你要求的约定）

agent 可以新增组件，但必须同时完成：

1. 新建 `components/<component-name>.html`（静态 HTML）
2. 在 `references/component-library.md` 追加条目（用途/字段/示例）
3. 若新增字段，更新 `references/payload-schema.md`
4. 不破坏现有组件与 action 协议（向后兼容）

推荐新增命名：

- `card-*`：信息卡
- `panel-*`：区域面板
- `dialog-*`：确认/取消弹框
- `cta-*`：操作按钮组

## 最小示例

```json
{
  "template": "spec_bootstrap_flow",
  "title": "新项目初始化链",
  "summary": "先澄清，再生成 L1-L5，占位写盘前确认",
  "blocks": [
    { "type": "steps", "title": "流程", "items": ["detect_state", "clarify", "bootstrap", "verify"] },
    { "type": "kv", "title": "工作区", "data": { "workspace_root": "D:/repo/demo" } }
  ],
  "actions": [
    {
      "id": "run-bootstrap",
      "label": "请求初始化",
      "type": "request_api",
      "confirm": "将写入 specs/L1~L5，是否继续？",
      "payload": { "endpoint": "/api/workspace/spec-bootstrap", "method": "POST" }
    },
    {
      "id": "cancel",
      "label": "取消",
      "type": "cancel"
    }
  ]
}
```
