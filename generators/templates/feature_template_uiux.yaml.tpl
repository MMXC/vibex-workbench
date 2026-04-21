# Feature UI/UX Spec — ${SAFE_NAME}
# Parent: ${FEATURE_ID}
# Generated: ${TIMESTAMP}

spec:
  version: "1.0"
  level: "5a_uiux"
  name: "${SAFE_NAME}_uiux"
  parent: "${SAFE_NAME}"
  status: draft

content:
  # Phase B 可选：与 parent L4 vision_traceability 对齐的一行追溯（不必展开）
  trace_note: |
    L5a：UI 切片；与 ${SAFE_NAME}（L4）vision_traceability 对齐时可补交互验收点。

  # 默认 flow-canvas 布局（可按需修改 components/regions/state_management）
  canvas_layout:
    type: flow-canvas
    width: "100%"
    height: "100%"
  regions: []
  components: []
  state_management:
    stores: []
