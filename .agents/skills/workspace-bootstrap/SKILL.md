---
name: workspace-bootstrap
description: 通过 agent 澄清方式初始化新工作区的 L1-L5 specs。适用于“新建工作区 / 打卡工作区 / 按模板初始化 spec 链路”的低频复杂任务。
---

# workspace-bootstrap

将“新工作区初始化”统一为 agent skill，而不是让调用方承担脚本路径、模板定位、占位符替换等实现细节。

## 何时使用

- 用户要在新目录按 VibeX 分层模板初始化 spec。
- 需要在执行前澄清：项目短名、owner、是否覆盖已有文件。
- 需要可迭代优化初始化策略，不希望改动固定工具协议。

## 输入（澄清后）

- `workspace_root`（必填）
- `project_slug`（可选，默认目录名）
- `owner`（可选，默认 `user`）
- `overwrite`（可选，默认 `false`）
- `confirm`（必填，写盘必须 `true`）

## 输出

- `written_files[]`
- `skipped_files[]`
- `chain`（L1→L5 spec.name）
- `errors[]`

## 执行入口

- 主入口脚本：`scripts/execute.py`
- 引用模板：`references/template-sources.md`

## 约束

- 默认幂等：已有文件不覆盖（除非 `overwrite=true`）。
- 仅写 `specs/L1-goal..L5-slice` 目标链路，不改业务代码。
- 写盘前必须显式确认（`confirm=true`）。
