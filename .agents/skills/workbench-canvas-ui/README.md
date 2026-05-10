# workbench-canvas-ui

用于 Workbench 中“静态 HTML 组件 + JSON 注入”的可操作画布技能包。

## 目录

- `SKILL.md`：使用规则与扩展约定
- `components/*.html`：可复用静态组件
- `assets/base.css`：统一样式
- `assets/runtime.js`：`data-bind` 注入运行时
- `templates/*.json`：模板 payload 示例
- `references/*.md`：组件与协议文档

## 快速接入

1. 选择一个模板 JSON
2. 将组件 HTML 插入画布容器
3. 调用 `window.WorkbenchCanvasUI.inject(root, payload)` 注入
4. 监听 action 按钮（按 `data-action` / `data-action-id`）

## 扩展

允许 agent 新增组件到 `components/`，但需同步更新：

- `references/component-library.md`
- `references/payload-schema.md`（若字段变更）
