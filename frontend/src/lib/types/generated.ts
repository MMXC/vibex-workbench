// ============================================================
// ⚠️  此文件由 spec-to-code 自动生成
//     来自: specs
//     生成时间: 2026-04-19
//     ⚠️  不要直接编辑此文件 — 改 *.svelte
// ============================================================

// ── Generated Types ──────────────────────────────────────────
export interface Thread {
  id: string;  // 
  goal: string;  // 用户最初输入的目标描述
  title?: string;  // 自动生成的标题（取 goal 前 N 字符）或用户自定义
  description?: string;  // 用户补充的描述
  stage?: string;  // 当前所处阶段
  tags?: string[];  // 用户自定义标签
  status?: string;  // 
  createdAt: string;  // 
  updatedAt?: string;  // 
  deletedAt?: string;  // 软删除时间戳
  parentThreadId?: string | null;  // 父 Thread ID（用于 Thread 分支）
  metadata?: Record<string, unknown>;  // 扩展字段预留
}

export interface ContextSlice {
  id: string;  // 
  threadId: string;  // 
  type: string;  // 
  content: string;  // 
  source?: string;  // 
  createdAt: string;  // 
  importance?: number;  // 重要性评分，影响上下文组装时的权重
}

export interface ThreadMeta {
  threadId?: string;  // 
  color?: string;  // sidebar 中显示的颜色
  icon?: string;  // emoji 或 icon name
  isPinned?: boolean;  // 是否固定在 sidebar 顶部
  sortOrder?: number;  // sidebar 排序顺序
}

export interface CreateThreadOptions {
  parentThreadId?: string | null;  // 从现有 Thread 分支创建
  tags?: string[];  // 
  description?: string;  // 
  color?: string;  // 
  title?: string;  // 自定义标题，不自动生成
}

export interface Run {
  id?: string;  // UUID，服务端下发
  thread_id: string;  // 所属 Thread ID
  goal: string;  // 本次 Run 的用户目标文本
  status: string;  // queued | planning | executing | paused | completed | failed | cancelled
  stage?: string;  // 当前阶段描述（如 'Thinking...' / 'Using tool: read_file'）
  visibility_layer?: number;  // 工具可见性层级：1（自然语言）| 2（结构化）| 3（开发者日志）
  created_at?: string;  // Run 创建时间
  started_at?: string;  // 实际开始执行时间
  finished_at?: string;  // Run 结束时间
  result_summary?: string;  // Run 完成后的摘要文本（最终 LLM 输出截断）
  error_message?: string;  // Run 失败时的错误信息
}

export interface ToolInvocation {
  id?: string;  // UUID，服务端 SSE 事件携带
  run_id: string;  // 所属 Run ID
  tool_name: string;  // 工具名称（如 'read_file' / 'terminal' / 'search_files'）
  tool_display_name?: string;  // 人类可读工具名称
  args?: Record<string, unknown>;  // 工具输入参数（JSON）
  result?: Record<string, unknown>;  // 工具输出结果（JSON）
  error?: string;  // 工具调用失败时的错误信息
  status: string;  // pending | running | completed | failed
  order?: number;  // 同 Run 内的调用顺序（由 SSE 事件顺序保证）
  started_at?: string;  // 调用开始时间
  finished_at?: string;  // 调用结束时间
  duration_ms?: number;  // 执行耗时（毫秒）
  canvas_node_id?: string;  // 关联的 Canvas 节点 ID（由 canvas-sync 填充）
}

export interface MessageDelta {
  id?: string;  // UUID
  run_id: string;  // 所属 Run ID
  role: string;  // 消息角色：user | assistant | system | tool
  content?: string;  // 累积的文本内容（增量追加）
  delta?: string;  // 本次 delta 增量文本
  tool_call_id?: string;  // 如果 role=tool，关联的 ToolInvocation ID
  is_final?: boolean;  // 是否为最终消息块（end 事件到达时置 true）
}

export interface SSEStream {
  thread_id?: string;  // 订阅的 Thread ID
  url?: string;  // SSE 端点 URL
  status?: string;  // disconnected | connecting | connected | reconnecting | error
  last_event_id?: string;  // 最后收到的 event ID（用于恢复）
  error_message?: string;  // 最近一次错误信息
  connected_at?: string;  // 连接建立时间
}

export interface VisibilityLayer {
  layer?: number;  // 1=自然语言（LLM 摘要工具调用）| 2=结构化（工具名+状态）| 3=开发者日志（完整 args/result）
  show_raw_args?: boolean;  // 是否展示原始工具参数（layer=3 时强制 true）
  show_raw_result?: boolean;  // 是否展示原始工具结果（layer=3 时强制 true）
  truncate_length?: number;  // layer=1 时的内容截断长度（字符数）
}

export interface Artifact {
  id?: string;  // UUID，服务端 SSE 事件携带
  thread_id: string;  // 所属 Thread ID
  run_id: string;  // 生成此 Artifact 的 Run ID
  type: string;  // artifact 类型：code | markdown | json | image | data | generic | diagram
  name: string;  // 显示名称（可包含路径，如 'src/utils/helper.ts'）
  content: string;  // artifact 正文内容（代码 / markdown / json string 等）
  language?: string;  // 代码语言（当 type=code 时，如 'typescript' / 'python'）
  version: number;  // 当前版本号，每次 updateArtifact +1
  mime_type?: string;  // MIME 类型（如 'text/typescript' / 'image/png'）
  size_bytes?: number;  // content 大小（字节数）
  tags?: string[];  // 用户/系统标签数组
  status: string;  // active | archived | deleted（软删除）
  deleted_at?: string | null;  // 软删除时间戳
  metadata?: Record<string, unknown>;  // 扩展字段：{ lineCount, charCount, sha256?, url?, thumbnailBase64? }
  referenced_by?: string[];  // 引用此 artifact 的 runId[]（由 canvas-sync / run-engine 维护）
  created_at: string;  // 首次创建时间（ISO8601）
  updated_at: string;  // 最近更新时间（ISO8601）
  last_accessed_at?: string;  // 最近访问时间（用于 LRU 淘汰）
}

export interface ArtifactVersion {
  id?: string;  // UUID，格式 {artifactId}_v{version}
  artifact_id: string;  // 所属 Artifact ID
  version: number;  // 版本号（与 artifact.version 对应）
  content: string;  // 该版本的内容快照
  change_summary?: string;  // 变更摘要（diff 描述）
  created_by_run_id?: string;  // 生成此版本的 Run ID
  created_at: string;  // 版本创建时间（ISO8601）
}

export interface ArtifactTag {
  id?: string;  // 
  artifact_id: string;  // 关联的 Artifact ID
  tag: string;  // 标签名（小写化，trimmed）
  created_by?: string;  // user | agent | system
  created_at: string;  // 
}

export interface StorageStats {
  used_bytes?: number;  // 当前使用的字节数
  quota_bytes?: number;  // 浏览器 IndexedDB 配额（估算）
  artifact_count?: number;  // 活跃 artifact 数量
  version_count?: number;  // 历史版本快照总数
  last_calculated_at?: string;  // 最近计算时间
}

export interface WorkbenchNode {
  id: string;  // 节点唯一 ID（格式：'{type}:{sourceId}'，如 'run:abc123'）
  type: string;  // 节点类型
  position: Record<string, unknown>;  // 节点在画布上的坐标（px）
  width?: number;  // 节点宽度（px），由 ReactFlow 自动测量
  height?: number;  // 节点高度（px）
  data: Record<string, unknown>;  // 节点业务数据（类型相关，见下）
  selected: boolean;  // 是否被选中
  dragging: boolean;  // 是否正在拖拽
  expanded: boolean;  // 节点是否展开详情
  zIndex?: number;  // 渲染层级（用于模态浮层优先级）
  parentId?: string;  // 父节点 ID（用于分组，future use）
  createdAt: string;  // 节点创建时间
}

export interface ThreadNodeData {
  threadId: string;  // 关联 Thread ID
  title?: string;  // Thread 标题（来自 threadStore.threadMeta）
  goal: string;  // Thread 目标文本（截断至 100 字符）
  runCount: number;  // 关联 Run 数量
  status: string;  // thread / archived
  createdAt: string;  // 创建时间
  updatedAt?: string;  // 最后更新时间
}

export interface RunNodeData {
  runId: string;  // 关联 Run ID
  threadId: string;  // 所属 Thread ID
  goal: string;  // Run 目标（截断至 80 字符）
  status: string;  // queued | planning | executing | paused | completed | failed | cancelled
  stage?: string;  // 当前阶段描述（如 'Thinking...'）
  toolCount: number;  // 关联 ToolInvocation 数量
  artifactCount: number;  // 关联 Artifact 数量
  progress: number;  // 进度百分比（0–100）
  createdAt: string;  // 
  startedAt?: string;  // 
  finishedAt?: string;  // 
}

export interface ToolNodeData {
  toolInvocationId: string;  // 工具调用 ID
  runId: string;  // 所属 Run ID
  toolName: string;  // 工具名称（如 read_file / terminal / search_files）
  toolDisplayName?: string;  // 人类可读工具名称
  toolIcon?: string;  // 工具图标名称（Lucide icon）
  status: string;  // pending | running | completed | failed
  args?: Record<string, unknown>;  // 工具调用参数（layer=3 时显示）
  result?: Record<string, unknown>;  // 工具返回结果（layer=3 时显示）
  error?: string;  // 失败错误信息
  summary?: string;  // 工具调用摘要（layer=1 时显示的压缩描述）
  order?: number;  // 同 Run 内的调用序号
  startedAt?: string;  // 
  finishedAt?: string;  // 
  durationMs?: number;  // 执行耗时（毫秒）
}

export interface ArtifactNodeData {
  artifactId: string;  // Artifact 唯一 ID
  name: string;  // Artifact 名称
  mimeType: string;  // MIME 类型
  previewUrl?: string;  // 缩略图 URL（来自 artifactStore.previews）
  runId?: string;  // 产出该 Artifact 的 Run ID
  toolInvocationId?: string;  // 产出该 Artifact 的 ToolInvocation ID
  sizeBytes?: number;  // 文件大小（bytes）
  language?: string;  // 代码语言（用于代码类 Artifact）
  createdAt: string;  // 
}

export interface MessageNodeData {
  messageId: string;  // 
  runId: string;  // 
  role: string;  // user | assistant | system | tool
  content: string;  // 累积的消息文本
  isStreaming: boolean;  // 是否正在流式接收
  toolCallId?: string;  // 若 role=tool，关联的 ToolInvocation ID
  isFinal: boolean;  // 消息是否已完整（end 事件到达）
  attachments: Record<string, unknown>;  // 附件 artifactId[]
}

export interface WorkbenchEdge {
  id: string;  // 边唯一 ID（格式：'{sourceId}→{targetId}'）
  source: string;  // 源节点 ID
  target: string;  // 目标节点 ID
  type: string;  // causal | reference | streaming
  label?: string;  // 边标签文本（如工具名）
  animated: boolean;  // 是否动画（用于 streaming 边）
  style?: Record<string, unknown>;  // 额外样式（stroke / strokeWidth / strokeDasharray）
  markerEnd?: string;  // 终点箭头类型
  selected: boolean;  // 
  zIndex?: number;  // 
}

export interface Viewport {
  x: number;  // 视口左上角 X 坐标（画布空间）
  y: number;  // 视口左上角 Y 坐标（画布空间）
  zoom: number;  // 缩放比例
}

export interface Command {
  id: string;  // 命令唯一 ID
  type: string;  // add_node | remove_node | update_node | add_edge | remove_edge | batch
  payload: Record<string, unknown>;  // 命令携带的数据（变化前后）
  timestamp: string;  // 命令执行时间
}

export interface UIStore {
  leftSidebarOpen: boolean;  // 左侧边栏展开/折叠状态
  rightSidebarOpen: boolean;  // 右侧边栏展开/折叠状态
  leftSidebarWidth: number;  // 左侧边栏宽度（px），用户可拖拽调整
  rightSidebarWidth: number;  // 右侧边栏宽度（px），用户可拖拽调整
  activePanel: string;  // 当前激活的右侧面板 Tab
  toolVisibilityLayer: number;  // 工具可见性层级（1=自然语言 / 2=结构化 / 3=开发者）
  composerExpanded: boolean;  // Composer 是否展开（false=收起为单行输入框）
  composerText: string;  // Composer 当前文本输入内容
  composerAttachments: Record<string, unknown>;  // Composer 当前附件列表（ArtifactId[]）
  selectedThreadId?: string;  // 当前选中的 Thread ID（同步 threadStore.currentThreadId）
  selectedArtifactId?: string;  // 当前选中的 Artifact ID（同步 artifactStore.selectedId）
  selectedToolInvocationId?: string;  // 当前选中的工具调用 ID
  modals: Record<string, unknown>;  // 当前打开的 Modal 队列（ModalInstance[]）
  dragState: Record<string, unknown>;  // 当前拖拽状态（DragState | null）
}

export interface ModalInstance {
  id: string;  // Modal 实例唯一 ID（crypto.randomUUID）
  type: string;  // Modal 类型
  props?: Record<string, unknown>;  // Modal 专属 props（类型相关）
  priority: number;  // 优先级，数字越大越靠前
  createdAt: string;  // Modal 创建时间（ISO 8601）
}

export interface DragState {
  type: string;  // 拖拽类型
  sourceId?: string;  // 拖拽源 ID（artifactId / threadId）
  sourceIndex?: number;  // 拖拽源在列表中的索引
  ghost?: Record<string, unknown>;  // 拖拽预览 ghost 元素数据
}

export interface ComposerAttachment {
  id: string;  // 附件唯一 ID（临时 UUID，提交后引用 artifactStore）
  artifactId?: string;  // 已上传的 Artifact ID（上传后填充）
  type: string;  // 附件类型
  previewUrl?: string;  // 本地预览 URL（blob: URL 或 data: URL）
  fileName?: string;  // 文件名（image/pdf）或 URL（url 类型）
  fileSize?: number;  // 文件大小（bytes）
  mimeType?: string;  // MIME type
  uploadStatus: string;  // 上传状态
  uploadProgress: number;  // 上传进度（0-100）
  extractedContent?: string;  // URL/PDF 提取后的文本内容（可选）
}

// ============================================================
// ⚠️  此文件由 spec-to-code 自动生成
//     来自: specs
//     生成时间: 2026-04-19
//     ⚠️  不要直接编辑此文件 — 改 *.svelte
// ============================================================

// ── Base Types (enum/union) ───────────────────────────────
export type RunStatus    = 'pending'|'planning'|'executing'|'completed'|'failed'|'cancelled';
export type TaskStatus   = 'pending'|'running'|'completed'|'failed';
export type ArtifactType = 'code'|'markdown'|'image'|'json'|'diagram'|'text';
export type NodeType     = 'thread'|'run'|'task'|'tool'|'artifact'|'message';
