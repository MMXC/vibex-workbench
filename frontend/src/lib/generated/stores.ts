// Auto-generated from vibex-workbench specs

import { writable } from 'svelte/store';

export const codeGen_store = writable({
  currentJob: null,
  stepLogs: [],
  isRunning: 0,
});

export const dslCanvas_store = writable({
  mermaidCode: "",
  highlightedSpecId: null,
  zoomLevel: 0,
  panOffset: 0,
  viewMode: 0,
});

export const routing_store = writable({
  changeInput: "",
  recommendations: [],
  isLoading: 0,
});

export const specEditor_store = writable({
  currentFile: null,
  content: "",
  isDirty: 0,
  openTabs: [],
  activeTabId: null,
  validationErrors: [],
});

export const ui_store = writable({
  leftSidebarOpen: 0,
  rightSidebarOpen: 0,
  activePanel: null,
  statusMessage: "",
  generationStatus: 0,
});
