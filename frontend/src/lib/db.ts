// ============================================================
// ⚠️  此文件由 spec-to-sveltekit 自动生成
//     来自: specs/feature
//     生成时间: 2026-04-20
//     模式: backend
//
// ⚠️  不要直接编辑此文件
//     修改 specs/ 目录下的 YAML 文件后重新运行 make generate-frontend
// ============================================================

// Dexie.js 数据库（local 模式）
import Dexie from 'dexie';

export class VibexDB extends Dexie {
  canvas!: Dexie.Table<{
    id: string; name: string; description: string;
    viewport_x: number; viewport_y: number; zoom: number;
    created_at: string; updated_at: string; is_deleted: number;
  }>;
  node!: Dexie.Table<{
    id: string; canvas_id: string; type: string; label: string;
    position_x: number; position_y: number; config: string;
    created_at: string; is_deleted: number;
  }>;
  edge!: Dexie.Table<{
    id: string; canvas_id: string; source_node_id: string;
    source_port: string; target_node_id: string; target_port: string;
    edge_type: string; condition_expression: string; is_deleted: number;
  }>;
  snapshot!: Dexie.Table<{
    id: string; canvas_id: string; data: string;
    created_at: string; is_auto: number;
  }>;
  threads!: Dexie.Table<{
    id: string; title: string; goal?: string; status?: string;
    created_at: string; updated_at: string; is_deleted: number;
    deleted_at?: string | null;
  }>;
  artifacts!: Dexie.Table<{
    id: string; name: string; type: string; content: string;
    mime_type?: string; tags: string[]; thread_id?: string; run_id?: string;
    created_at: string; updated_at?: string; is_deleted: number;
  }>;

  constructor() {
    super('VibexDB');
    this.version(1).stores({
      canvas: 'id, name, updated_at, is_deleted',
      node: 'id, canvas_id, type, is_deleted',
      edge: 'id, canvas_id, source_node_id, target_node_id, is_deleted',
      snapshot: 'id, canvas_id, created_at',
      threads: 'id, updated_at, is_deleted',
      artifacts: 'id, type, created_at, is_deleted',
    });
  }
}

export type DBThread = {
  id: string; title: string; goal?: string; status?: string;
  created_at: string; updated_at: string; is_deleted: number;
  deleted_at?: string | null;
};

export type DBArtifact = {
  id: string; name: string; type: string; content: string;
  mime_type?: string; tags: string[]; thread_id?: string; run_id?: string;
  created_at: string; updated_at?: string; is_deleted: number;
};

export const db = new VibexDB();
