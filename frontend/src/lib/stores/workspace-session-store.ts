// workspace-session-store.ts — Manages workspace session state and persistence.
// SLICE-workspace-session-store
import { writable, get } from 'svelte/store';
import { browser } from '$app/environment';

export interface WorkspaceSession {
  workspaceRoot: string;
  projectName: string;
  lastSpecPath: string | null;
  lastSpecName: string | null;
  openPanels: string[];
  activeThreadId: string | null;
  createdAt: string;
  updatedAt: string;
}

function createWorkspaceSessionStore() {
  const defaultSession: WorkspaceSession = {
    workspaceRoot: '',
    projectName: '',
    lastSpecPath: null,
    lastSpecName: null,
    openPanels: [],
    activeThreadId: null,
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
  };

  const { subscribe, set, update } = writable<WorkspaceSession>(defaultSession);

  return {
    subscribe,

    setSession(session: WorkspaceSession) {
      set({ ...session, updatedAt: new Date().toISOString() });
      if (browser) saveToLocalStorage(session);
    },

    setWorkspaceRoot(path: string) {
      update(s => ({ ...s, workspaceRoot: path, updatedAt: new Date().toISOString() }));
      if (browser) saveToLocalStorage(get({ subscribe }));
    },

    setLastSpec(path: string, name: string) {
      update(s => ({ ...s, lastSpecPath: path, lastSpecName: name, updatedAt: new Date().toISOString() }));
      if (browser) saveToLocalStorage(get({ subscribe }));
    },

    setActiveThread(threadId: string | null) {
      update(s => ({ ...s, activeThreadId: threadId, updatedAt: new Date().toISOString() }));
      if (browser) saveToLocalStorage(get({ subscribe }));
    },

    openPanel(panelId: string) {
      update(s => {
        if (s.openPanels.includes(panelId)) return s;
        return { ...s, openPanels: [...s.openPanels, panelId], updatedAt: new Date().toISOString() };
      });
      if (browser) saveToLocalStorage(get({ subscribe }));
    },

    closePanel(panelId: string) {
      update(s => ({
        ...s,
        openPanels: s.openPanels.filter(p => p !== panelId),
        updatedAt: new Date().toISOString(),
      }));
      if (browser) saveToLocalStorage(get({ subscribe }));
    },

    loadFromStorage() {
      if (!browser) return;
      try {
        const stored = localStorage.getItem('vibex-workspace-session');
        if (stored) {
          const parsed = JSON.parse(stored) as WorkspaceSession;
          set({ ...parsed, updatedAt: new Date().toISOString() });
        }
      } catch {
        // ignore corrupt storage
      }
    },

    clear() {
      set(defaultSession);
      if (browser) localStorage.removeItem('vibex-workspace-session');
    },
  };
}

function saveToLocalStorage(session: WorkspaceSession) {
  try {
    localStorage.setItem('vibex-workspace-session', JSON.stringify(session));
  } catch {
    // quota exceeded or private mode
  }
}

export const workspaceSessionStore = createWorkspaceSessionStore();