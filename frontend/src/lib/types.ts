// ============================================================
// ⚠️  此文件由 spec-to-code 自动生成
//     来自: specs
//     生成时间: 2026-04-21
// ============================================================

// ── Generated Types ──────────────────────────────────────────
export interface GenerationJob {
  id: string;  // 
  startedAt: Date;  // 
  completedAt?: Date | null;  // 
  status: 'running' | 'done' | 'error';  // 
  steps?: GenerationStep[];  // 
}

export interface GenerationStep {
  index?: number;  // 
  name: string;  // 
  status: 'pending' | 'running' | 'done' | 'error';  // 
  startedAt: Date | null;  // 
  durationMs?: number | null;  // 
  error?: string | null;  // 
}

export interface GenerationEvent {
  eventId?: string;  // 
  type: 'generation.started' | 'generation.step' | 'generation.completed' | 'generation.failed';  // 
  payload?: Record<string, unknown>;  // 
  timestamp?: Date;  // 
}

export interface FileDiff {
  filePath?: string;  // 
  changeType?: 'added' | 'modified' | 'deleted';  // 
  before?: string | null;  // 
  after?: string | null;  // 
  diffHunk?: string | null;  // 
}

export interface SpecNode {
  id: string;  // 
  specId?: string;  // 
  level: number;  // 
  name: string;  // 
  x?: number;  // 
  y?: number;  // 
  color?: string;  // 
}

export interface SpecEdge {
  id: string;  // 
  fromSpecId?: string;  // 
  toSpecId?: string;  // 
  label?: string | null;  // 
}

export interface MermaidGraph {
  code?: string;  // 
  generatedAt?: Date;  // 
  specIds?: string[];  // 
}

export interface CanvasNode {
  id: string;  // 
  eventId?: string;  // 
  threadId?: string;  // 
  nodeType?: input | sequence | iteration | branch | subagent | gate | output;  // 
  label?: string;  // 
  x?: number;  // 
  y?: number;  // 
  status: pending | running | done | error;  // 
  color?: string;  // 
  payload?: Record<string, unknown>;  // 
}

export interface CanvasConnection {
  id: string;  // 
  fromNodeId?: string;  // 
  toNodeId?: string;  // 
  type: sequence | iteration-loop;  // 
}

export interface SSEEventSubscription {
  eventPattern?: string;  // 
  handler?: Function;  // 
  activeThreadId?: string | null;  // 
}

export interface RouteDecision {
  route?: goal | feature | bug;  // 
  confidence?: number (0-1);  // 
  reason?: string;  // 
}

export interface ClarificationSession {
  id: string;  // 
  threadId?: string;  // 
  route?: goal | feature | bug;  // 
  state?: idle | receiving | clarification_loop | terminating;  // 
  subState?: string | null;  // 
  questions?: Clarification[];  // 
  answers?: Clarification[];  // 
  suggestions?: Suggestion[];  // 
  isResolved?: boolean;  // 
  createdAt: Date;  // 
  resolvedAt?: Date | null;  // 
}

export interface Clarification {
  id: string;  // 
  type: ask | answer;  // 
  question?: string;  // 
  answer?: string | null;  // 
  timestamp?: Date;  // 
}

export interface Suggestion {
  id: string;  // 
  content: string;  // 
  type: style | direction | detail | boundary;  // 
  accepted?: boolean | null;  // 
}

export interface ConfirmedIO {
  id: string;  // 
  threadId?: string;  // 
  route?: goal | feature | bug;  // 
  input?: string;  // 
  output?: string;  // 
  boundary?: string;  // 
  confirmedBy?: user_confirmation | screenshot_confirmed;  // 
  confirmedAt?: Date;  // 
  specId?: string | null;  // 
  prototypePath?: string | null;  // 
}

export interface BugReport {
  id: string;  // 
  threadId?: string;  // 
  description?: string;  // 
  expected?: string;  // 
  actual?: string;  // 
  reproduceSteps?: string[];  // 
  specId?: string;  // 
  rootCause?: string | null;  // 
  status: open | fixed | wontfix;  // 
  changelog?: BugChangelogEntry[];  // 
}

export interface BugChangelogEntry {
  version?: string;  // 
  date?: Date;  // 
  author?: string;  // 
  changes?: string[];  // 
}

export interface SpecFile {
  id: string;  // 
  path: string;  // 
  content: string;  // 
  level: number;  // 
  name: string;  // 
  parent?: string | null;  // 
  status: 'active' | 'draft' | 'confirmed' | 'deprecated';  // 
  updatedAt: Date;  // 
}

export interface Tab {
  id: string;  // 
  specFileId?: string;  // 
  label?: string;  // 
  isActive?: boolean;  // 
}

export interface Violation {
  id: string;  // 
  specId?: string;  // 
  type: 'missing-parent' | 'circular-ref' | 'invalid-level' | 'io-contract-missing';  // io-contract-missing = spec.content 缺少 input/output/boundary/behavior/changelog

  message?: string;  // 
  line?: number | null;  // 
}

export interface LayoutState {
  id: string;  // 
  leftSidebarWidth?: number;  // 
  rightSidebarWidth?: number;  // 
  leftSidebarOpen?: boolean;  // 
  rightSidebarOpen?: boolean;  // 
  viewMode?: mermaid_or_canvas;  // dsl-canvas 的视图模式，值: mermaid | canvas
}

export interface ConversationThread {
  id: string;  // 
  title?: string;  // 
  route?: route_type;  // 
  messages?: ConversationMessage[];  // 
  confirmedIO?: ConfirmedIO_or_null;  // 
  createdAt: Date;  // 
  isActive?: boolean;  // 
}

export interface ConversationMessage {
  id: string;  // 
  role?: user_or_agent;  // 
  content: string;  // 
  timestamp?: Date;  // 
  route?: route_type;  // 
  clarificationType?: clarification_type;  // clarification loop 中的消息类型
}

// ============================================================
// ⚠️  此文件由 spec-to-code 自动生成
//     来自: specs
//     生成时间: 2026-04-21
// ============================================================

// ── Base Types (enum/union) ───────────────────────────────
export type RunStatus    = 'pending'|'planning'|'executing'|'completed'|'failed'|'cancelled';
export type TaskStatus   = 'pending'|'running'|'completed'|'failed';
export type ArtifactType = 'code'|'markdown'|'image'|'json'|'diagram'|'text';
export type NodeType     = 'thread'|'run'|'task'|'tool'|'artifact'|'message';


export interface Step {
  id?: string;
  name?: string;
  status?: 'pending' | 'running' | 'done' | 'error';
  log?: string;
  startedAt?: Date;
  completedAt?: Date | null;
}

export interface Change {
  id?: string;
  description?: string;
  type?: 'bug' | 'feature' | 'refactor';
}

export interface SpecLocation {
  specId?: string;
  level?: number;
  confidence?: number;
  reason?: string;
}

export type FileDiffStatus = 'added' | 'modified' | 'deleted' | 'unchanged';

export interface FileDiffCreate {
  filePath: string;
  oldContent?: string;
  newContent?: string;
  status?: FileDiffStatus;
}

export interface FileDiffUpdate {
  id: string;
  oldContent?: string;
  newContent?: string;
  status?: FileDiffStatus;
}

export interface DiffHunk {
  oldStart: number;
  oldLines: number;
  newStart: number;
  newLines: number;
  lines: string[];
}

export interface ImpactReport {
  id?: string;
  specId?: string;
  affectedFiles?: string[];
  breakingChanges?: BreakingChange[];
  createdAt?: Date;
}

export interface BreakingChange {
  file: string;
  type: 'api' | 'ui' | 'data' | 'behavior';
  description: string;
  severity?: 'low' | 'medium' | 'high';
}

export interface ImpactReportCreate {
  specId: string;
  affectedFiles?: string[];
}

export interface ImpactReportUpdate {
  id: string;
  affectedFiles?: string[];
  breakingChanges?: BreakingChange[];
}

export interface SpecAST {
  id?: string;
  specId?: string;
  content?: string;
  parsed?: unknown;
  level?: number;
  parentChain?: string[];
}

export interface SpecASTCreate {
  specId: string;
  content: string;
  level?: number;
}

export interface SpecASTUpdate {
  id: string;
  content?: string;
}

export interface SpecFileCreate {
  path: string;
  content?: string;
  level?: number;
  name?: string;
  parent?: string;
}

export interface SpecFileUpdate {
  id: string;
  content?: string;
  name?: string;
  status?: 'active' | 'draft' | 'deprecated';
}

export interface SpecLocationCreate {
  specId: string;
  level?: number;
  confidence?: number;
  reason?: string;
}

export interface SpecLocationUpdate {
  id: string;
  confidence?: number;
}

export interface SpecNodeCreate {
  specId: string;
  level?: number;
  name?: string;
  x?: number;
  y?: number;
  color?: string;
}

export interface SpecNodeUpdate {
  id: string;
  x?: number;
  y?: number;
  name?: string;
  color?: string;
}

export interface Viewport {
  x?: number;
  y?: number;
  zoom?: number;
  width?: number;
  height?: number;
}

export interface ViewportCreate {
  x?: number;
  y?: number;
  zoom?: number;
  width?: number;
  height?: number;
}

export interface ViewportUpdate {
  id?: string;
  x?: number;
  y?: number;
  zoom?: number;
}

export interface ViolationCreate {
  specId: string;
  type: 'missing-parent' | 'circular-ref' | 'invalid-level';
  message: string;
  line?: number;
}

export interface ViolationUpdate {
  id: string;
  resolved?: boolean;
}

export interface DependencyGraph {
  nodes: SpecNode[];
  edges: SpecEdge[];
}

export interface GenerationJobCreate {
  goal?: string;
  threadId?: string;
}

export interface GenerationJobUpdate {
  id: string;
  status?: 'running' | 'done' | 'error';
  steps?: Step[];
}

export interface StepLog {
  step: string;
  status: 'pending' | 'running' | 'done' | 'error';
  log: string;
  timestamp?: Date;
}

export type Thread = ConversationThread;
