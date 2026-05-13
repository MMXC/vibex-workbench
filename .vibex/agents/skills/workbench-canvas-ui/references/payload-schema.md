# CanvasSkillPayload Schema

`CanvasSkillPayload` 是画布渲染输入，所有模板都基于此协议。

## 顶层字段

- `template: string` 模板名（如 `spec_bootstrap_flow`）
- `title: string` 主标题
- `summary?: string` 摘要
- `theme?: "dark" | "light" | "system"`
- `blocks: Block[]` 展示块数组
- `actions?: Action[]` 可交互动作数组
- `meta?: Record<string, string | number | boolean | null>` 辅助信息

## Block

- `type: "steps" | "kv" | "code" | "json" | "notice" | "table"`
- `title?: string`
- `items?: string[]`（`steps`）
- `data?: Record<string, string | number | boolean | null>`（`kv`/`json`）
- `content?: string`（`code`/`notice`）
- `columns?: string[]`、`rows?: Array<Array<string | number | boolean | null>>`（`table`）
- `collapsible?: boolean`

## Action

- `id: string`
- `label: string`
- `type: "prefill_composer" | "request_api" | "open_slot" | "pin_canvas" | "cancel"`
- `confirm?: string`（写入/执行前建议必填）
- `payload?: Record<string, unknown>`
- `variant?: "primary" | "secondary" | "danger"`

## 安全约定

- `request_api` 涉及写操作时，`confirm` 必填。
- `payload.workspace_root` 存在时必须在 UI 中可见。
- `payload.target_path` 存在时必须在 UI 中可见。
