// ============================================================
// VibeX Workbench — IndexedDB Database (Dexie)
// E2 Thread + E4 Artifact 持久化层
// ============================================================

import Dexie, { type Table } from 'dexie';

// ── Type definitions ──────────────────────────────────────────

export interface DBThread {
  id: string;
  title: string;
  goal: string;
  status: string;
  createdAt: string;
  updatedAt: string;
  deletedAt?: string; // 软删除标记
}

export interface DBArtifact {
  id: string;
  name: string;
  type: string;
  content: string;
  mime_type: string;
  tags: string[];
  thread_id?: string;
  run_id?: string;
  created_at: string;
  updated_at?: string;
}

// ── Database class ────────────────────────────────────────────

class WorkbenchDB extends Dexie {
  threads!: Table<DBThread, string>;
  artifacts!: Table<DBArtifact, string>;

  constructor() {
    super('vibex-workbench');
    this.version(1).stores({
      threads: 'id, createdAt, updatedAt, deletedAt',
      artifacts: 'id, type, name, created_at, thread_id, run_id',
    });
  }
}

export const db = new WorkbenchDB();
