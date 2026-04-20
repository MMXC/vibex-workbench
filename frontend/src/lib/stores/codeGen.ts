// ============================================================
// ⚠️  此文件由 spec-to-sveltekit 自动生成
//     来自: specs/feature
//     生成时间: 2026-04-20
//     模式: backend
//
// ⚠️  不要直接编辑此文件
//     修改 specs/ 目录下的 YAML 文件后重新运行 make generate-frontend
// ============================================================

// codeGenStore — generated from canvas_uiux.yaml

import { writable } from 'svelte/store';
import type { GenerationJob, StepLog } from '../types';

interface codeGenState {
  currentJob: GenerationJob | null;
  stepLogs: StepLog[]  // eslint-disable-line @typescript-eslint/no-explicit-any;
  isRunning: boolean;
}

function createcodeGenStore() {
  const initial: codeGenState = {
  currentJob: null,
  stepLogs: [],
  isRunning: false,
  };
  return writable(initial);
}

export const codeGenStore = createcodeGenStore();
