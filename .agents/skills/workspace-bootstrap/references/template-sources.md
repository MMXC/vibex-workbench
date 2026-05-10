# Template Sources

`workspace-bootstrap` skill 不复制模板内容，而是引用仓库内权威模板：

- `spec-templates/L1-goal/L1-goal-template.yaml`
- `spec-templates/L2-skeleton/L2-skeleton-template.yaml`
- `spec-templates/L3-module/L3-module-template.yaml`
- `spec-templates/L4-feature/L4-feature-template.yaml`
- `spec-templates/L5-slice/L5-slice-template.yaml`

分层治理与初始化契约来源：

- `spec-templates/L0-meta-convention/L0-layer-contract-template.yaml`
  - `content.product_layering_governance`
  - `content.workspace_bootstrap_contract`

说明：

- 模板更新后，skill 执行将自动使用最新模板。
- 若需“冻结版本”，可在 skill 下新增快照模板目录并在 execute 参数中显式切换。
