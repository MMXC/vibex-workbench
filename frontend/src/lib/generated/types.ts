// Auto-generated from vibex-workbench specs

export interface Step {
  id: string;
  name: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
  result?: string;
  error?: string;
}

export interface GenerationJob {
  id: string;
  startedAt: Date;
  completedAt: Date | null;
  status: 'running' | 'done' | 'error';
  steps: Step[];
}

export interface SpecNode {
  id: string;
  specId: string;
  level: number;
  name: string;
  x: number;
  y: number;
  color: string;
}

export interface SpecEdge {
  id: string;
  fromSpecId: string;
  toSpecId: string;
  label: string | null;
}

export interface MermaidGraph {
  code: string;
  generatedAt: Date;
  specIds: string[];
}

export interface Change {
  id: string;
  description: string;
  type: 'bug' | 'feature' | 'refactor';
}

export interface SpecLocation {
  specId: string;
  level: number;
  confidence: number;
  reason: string;
}

export interface SpecFile {
  id: string;
  path: string;
  content: string;
  level: number;
  name: string;
  parent: string | null;
  status: 'active' | 'draft' | 'deprecated';
  updatedAt: Date;
}

export interface Tab {
  id: string;
  specFileId: string;
  label: string;
  isActive: boolean;
}

export interface Violation {
  id: string;
  specId: string;
  type: 'missing-parent' | 'circular-ref' | 'invalid-level';
  message: string;
  line: number | null;
}

export interface LayoutState {
  id: string;
  leftSidebarWidth: number;
  rightSidebarWidth: number;
  leftSidebarOpen: boolean;
  rightSidebarOpen: boolean;
}
