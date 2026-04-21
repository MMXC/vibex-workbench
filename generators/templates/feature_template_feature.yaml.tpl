# Feature Spec ${FEATURE_ID}
# Parent: ${PARENT_ID}
# Generated: ${TIMESTAMP}

spec:
  id: "${FEATURE_ID}"
  version: "1.0"
  level: "4_feature"
  name: "${SAFE_NAME}"
  parent: "${PARENT_ID}"
  status: draft
  created_at: "${TIMESTAMP}"

content:
  description: "${FEATURE_NAME}"

  # M0：愿景承接 — 生成后按 goal 替换 constraint_ids；若 parent 为非 MOD 的 L4，须改 l3.module_spec_path / 增加 l4_aggregate_parent（见范本 ide-chrome、layout_resize）
  vision_traceability:
    summary: "${FEATURE_NAME} — 愿景承接（生成后补一句可验收摘要）"
    l1:
      goal_path: "specs/project-goal/vibex-workbench-goal.yaml"
      constraint_ids: ["C1", "C7"]
    l2:
      skeleton_path: "specs/architecture/vibex-workbench-skeleton.yaml"
      module_id: "${PARENT_ID}"
      anchors_hint:
        - "content.modules 中 ${PARENT_ID}"
        - "content.upstream_l1"
    l3:
      module_spec_path: "specs/module/${PARENT_ID}_module.yaml"
    acceptance_summary:
      - "（生成后填写：用户/评审可见的两条验收句）"

  sub_specs:
    uiux: "${SAFE_NAME}_uiux.yaml"
