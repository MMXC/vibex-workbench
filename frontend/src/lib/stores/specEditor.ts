// ============================================================
// ⚠️  此文件由 spec-to-sveltekit 自动生成
//     来自: specs/feature
//     生成时间: 2026-04-20
//     模式: backend
//
// ⚠️  不要直接编辑此文件
//     修改 specs/ 目录下的 YAML 文件后重新运行 make generate-frontend
// ============================================================

// specEditorStore — generated from canvas_uiux.yaml

import { writable } from 'svelte/store';
import type { Tab, Violation } from '../types';

interface specEditorState {
  currentFile: string | null;
  content: string;
  isDirty: boolean;
  openTabs: Tab[];
  activeTabId: string | null;
  validationErrors: Violation[];
}

function createspecEditorStore() {
  const initial: specEditorState = {
  currentFile: null,
  content: '',
  isDirty: false,
  openTabs: [],
  activeTabId: null,
  validationErrors: [],
  };
  return writable(initial);
}

export const specEditorStore = createspecEditorStore();
