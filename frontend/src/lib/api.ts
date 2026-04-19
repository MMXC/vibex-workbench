// ============================================================
// ⚠️  此文件由 spec-to-sveltekit 自动生成
//     来自: /root/vibex-workbench/specs
//     生成时间: 2026-04-19
//     模式: backend
//
// ⚠️  不要直接编辑此文件
//     修改 specs/ 目录下的 YAML 文件后重新运行 make generate-frontend
// ============================================================

import type {
} from './types';

const API_BASE = '/api';

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const res = await fetch(API_BASE + path, {
    headers: { 'Content-Type': 'application/json', ...options.headers },
    ...options,
  });
  if (!res.ok) throw new Error(`API ${path} failed: ${res.status}`);
  return res.json();
}

// ── Canvas API ──────────────────────────────────────────────

