# Component Library

本库组件均为**静态 HTML 片段**，通过 `data-bind` 注入数据。

## 1) `components/card-flow.html`

- 用途：展示流程步骤 + 状态标签
- 关键绑定：
  - `data-bind="title"`
  - `data-bind="summary"`
  - `data-bind="steps"`（数组）

## 2) `components/panel-kv.html`

- 用途：展示工作区、路径、参数等键值
- 关键绑定：
  - `data-bind="kv.title"`
  - `data-bind="kv.items"`（对象键值）

## 3) `components/dialog-confirm.html`

- 用途：确认/取消框（写入与请求动作）
- 关键绑定：
  - `data-bind="confirm.title"`
  - `data-bind="confirm.message"`
  - `data-bind="confirm.confirmLabel"`
  - `data-bind="confirm.cancelLabel"`

## 4) `components/toolbar-actions.html`

- 用途：请求按钮区
- 关键绑定：
  - `data-bind="actions"`（数组）
  - 每个 action 支持 `id/label/type/variant/confirm`

---

## Agent 扩展组件规则

新增组件时必须：

1. 文件名符合 `components/<kind>-<name>.html`
2. 仅使用静态 HTML，不内嵌业务 JS 逻辑
3. 必须声明 `data-component="<name>"` 根节点
4. 在本文件补齐用途和 `data-bind` 说明
5. 如新增 payload 字段，同步更新 `payload-schema.md`
