// ============================================================
// ⚠️  此文件由 spec-to-code 自动生成
//     来自: /root/vibex-workbench/specs
//     生成时间: 2026-04-20
//     ⚠️  不要直接编辑此文件 — 改 *.svelte
// ============================================================

// ── Generated Types ──────────────────────────────────────────
export interface LayoutState {
  id?: string;  // 
  leftSidebarWidth?: number;  // 
  rightSidebarWidth?: number;  // 
  leftSidebarOpen?: boolean;  // 
  rightSidebarOpen?: boolean;  // 
  viewMode?: mermaid_or_canvas;  // dsl-canvas 的视图模式，值: mermaid | canvas
}

export interface ConversationThread {
  id?: string;  // 
  title?: string;  // 
  route?: route_type;  // 
  messages?: ConversationMessage[];  // 
  confirmedIO?: ConfirmedIO_or_null;  // 
  createdAt?: Date;  // 
  isActive?: boolean;  // 
}

export interface ConversationMessage {
  id?: string;  // 
  role?: user_or_agent;  // 
  content?: string;  // 
  timestamp?: Date;  // 
  route?: route_type;  // 
  clarificationType?: clarification_type;  // clarification loop 中的消息类型
}

// ============================================================
// ⚠️  此文件由 spec-to-code 自动生成
//     来自: /root/vibex-workbench/specs
//     生成时间: 2026-04-20
//     ⚠️  不要直接编辑此文件 — 改 *.svelte
// ============================================================

// ── Base Types (enum/union) ───────────────────────────────
export type RunStatus    = 'pending'|'planning'|'executing'|'completed'|'failed'|'cancelled';
export type TaskStatus   = 'pending'|'running'|'completed'|'failed';
export type ArtifactType = 'code'|'markdown'|'image'|'json'|'diagram'|'text';
export type NodeType     = 'thread'|'run'|'task'|'tool'|'artifact'|'message';
