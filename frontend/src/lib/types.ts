// ============================================================
// ⚠️  此文件由 spec-to-sveltekit 自动生成
//     来自: specs/feature
//     生成时间: 2026-04-20
//     模式: backend
//
// ⚠️  不要直接编辑此文件
//     修改 specs/ 目录下的 YAML 文件后重新运行 make generate-frontend
// ============================================================

// ── Generated Types ──────────────────────────────────────────
export interface SpecNode {
  id?: string;
  specId?: string;
  level?: number;
  name?: string;
  x?: number;
  y?: number;
  color?: string;
}

export interface SpecEdge {
  id?: string;
  fromSpecId?: string;
  toSpecId?: string;
  label?: string | null;
}

export interface MermaidGraph {
  code?: string;
  generatedAt?: Date;
  specIds?: string[];
}

export interface GenerationJob {
  id?: string;
  startedAt?: Date;
  completedAt?: Date | null;
  status?: 'running' | 'done' | 'error';
  steps?: Step[];
}

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

export interface LayoutState {
  id?: string;
  leftSidebarWidth?: number;
  rightSidebarWidth?: number;
  leftSidebarOpen?: boolean;
  rightSidebarOpen?: boolean;
}

export interface SpecFile {
  id?: string;
  path?: string;
  content?: string;
  level?: number;
  name?: string;
  parent?: string | null;
  status?: 'active' | 'draft' | 'deprecated';
  updatedAt?: Date;
}

export interface Tab {
  id?: string;
  specFileId?: string;
  label?: string;
  isActive?: boolean;
}

export interface Violation {
  id?: string;
  specId?: string;
  type?: 'missing-parent' | 'circular-ref' | 'invalid-level';
  message?: string;
  line?: number | null;
}

// ── Missing Generated Types ──────────────────────────────────
export type FileDiffStatus = 'added' | 'modified' | 'deleted' | 'unchanged';

export interface FileDiff {
  id?: string;
  filePath?: string;
  oldContent?: string | null;
  newContent?: string | null;
  status?: FileDiffStatus;
  hunks?: DiffHunk[];
}

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

