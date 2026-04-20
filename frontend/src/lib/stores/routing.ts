// ============================================================
// ⚠️  此文件由 spec-to-sveltekit 自动生成
//     来自: specs/feature
//     生成时间: 2026-04-20
//     模式: backend
//
// ⚠️  不要直接编辑此文件
//     修改 specs/ 目录下的 YAML 文件后重新运行 make generate-frontend
// ============================================================

// routingStore — generated from canvas_uiux.yaml

import { writable } from 'svelte/store';
import type { SpecLocation } from '../types';

interface routingState {
  changeInput: string;
  recommendations: SpecLocation[];
  isLoading: boolean;
}

function createroutingStore() {
  const initial: routingState = {
  changeInput: '',
  recommendations: [],
  isLoading: false,
  };
  return writable(initial);
}

export const routingStore = createroutingStore();
