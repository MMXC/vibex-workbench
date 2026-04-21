/**
 * 工作台分区尺寸（Cursor 式）：持久化 localStorage。
 * Spec: specs/feature/workbench-shell/workbench-layout_resize_feature.yaml
 */
import { writable } from 'svelte/store';
import { browser } from '$app/environment';

/** 中央主栏 (.wb-main) 高度（ResizeObserver） */
export const workbenchMainAreaHeight = writable(0);

/** 右侧 AI 栏 (.wb-right) 内部高度（用于钳制 AI 输入区拖拽） */
export const workbenchRightPanelHeight = writable(0);

const LS_KEY_V2 = 'vibex.workbench.layout.v2';
const LS_KEY_V1 = 'vibex.workbench.layout.v1';

export interface WorkbenchLayoutDims {
  sidebarLeftPx: number;
  panelRightPx: number;
  /** 右侧栏底部 AI 输入条高度（Composer） */
  aiComposerBarPx: number;
  /** 底部 Problems / Terminal 等面板高度 */
  bottomDockPx: number;
}

export const WORKBENCH_LAYOUT_LIMITS = {
  sidebarLeft: { min: 180, max: 560, default: 260 },
  panelRight: { min: 240, max: 720, default: 380 },
  aiComposer: { min: 96, max: 400, default: 156 },
  bottomDock: { min: 120, max: 520, default: 200 },
} as const;

function defaults(): WorkbenchLayoutDims {
  return {
    sidebarLeftPx: WORKBENCH_LAYOUT_LIMITS.sidebarLeft.default,
    panelRightPx: WORKBENCH_LAYOUT_LIMITS.panelRight.default,
    aiComposerBarPx: WORKBENCH_LAYOUT_LIMITS.aiComposer.default,
    bottomDockPx: WORKBENCH_LAYOUT_LIMITS.bottomDock.default,
  };
}

function clampSidebar(n: number) {
  const { min, max } = WORKBENCH_LAYOUT_LIMITS.sidebarLeft;
  return Math.min(max, Math.max(min, Math.round(n)));
}

function clampPanel(n: number) {
  const { min, max } = WORKBENCH_LAYOUT_LIMITS.panelRight;
  return Math.min(max, Math.max(min, Math.round(n)));
}

function clampAiComposer(n: number) {
  const { min, max } = WORKBENCH_LAYOUT_LIMITS.aiComposer;
  return Math.min(max, Math.max(min, Math.round(n)));
}

function clampBottomDock(n: number) {
  const { min, max } = WORKBENCH_LAYOUT_LIMITS.bottomDock;
  return Math.min(max, Math.max(min, Math.round(n)));
}

/** AI 输入条高度相对右侧栏内高的上限（留出对话区最小高度） */
export function clampAiComposerForPanel(composerPx: number, panelInnerHeightPx: number) {
  const chatMin = 140;
  const maxComposer = Math.max(
    WORKBENCH_LAYOUT_LIMITS.aiComposer.min,
    panelInnerHeightPx - chatMin
  );
  return Math.min(maxComposer, Math.max(WORKBENCH_LAYOUT_LIMITS.aiComposer.min, Math.round(composerPx)));
}

function load(): WorkbenchLayoutDims {
  if (!browser) return defaults();
  try {
    const raw2 = localStorage.getItem(LS_KEY_V2);
    if (raw2) {
      const j = JSON.parse(raw2) as Partial<WorkbenchLayoutDims>;
      const d = defaults();
      return {
        sidebarLeftPx: clampSidebar(j.sidebarLeftPx ?? d.sidebarLeftPx),
        panelRightPx: clampPanel(j.panelRightPx ?? d.panelRightPx),
        aiComposerBarPx: clampAiComposer(j.aiComposerBarPx ?? d.aiComposerBarPx),
        bottomDockPx: clampBottomDock(j.bottomDockPx ?? d.bottomDockPx),
      };
    }
    const raw1 = localStorage.getItem(LS_KEY_V1);
    if (raw1) {
      const j = JSON.parse(raw1) as Record<string, unknown>;
      const d = defaults();
      return {
        sidebarLeftPx: clampSidebar(Number(j.sidebarLeftPx) || d.sidebarLeftPx),
        panelRightPx: clampPanel(Number(j.panelRightPx) || d.panelRightPx),
        aiComposerBarPx: clampAiComposer(Number(j.composerBarPx) || d.aiComposerBarPx),
        bottomDockPx: d.bottomDockPx,
      };
    }
  } catch {
    /* ignore */
  }
  return defaults();
}

function persist(d: WorkbenchLayoutDims) {
  if (!browser) return;
  try {
    localStorage.setItem(LS_KEY_V2, JSON.stringify(d));
  } catch {
    /* quota */
  }
}

function createWorkbenchLayoutStore() {
  const { subscribe, set, update } = writable<WorkbenchLayoutDims>(load());

  return {
    subscribe,

    reset() {
      const d = defaults();
      set(d);
      persist(d);
    },

    previewSidebarLeftPx(px: number) {
      update(s => ({ ...s, sidebarLeftPx: clampSidebar(px) }));
    },

    previewPanelRightPx(px: number) {
      update(s => ({ ...s, panelRightPx: clampPanel(px) }));
    },

    previewAiComposerBarPx(px: number, panelInnerHeightPx: number) {
      update(s => ({
        ...s,
        aiComposerBarPx: clampAiComposerForPanel(px, panelInnerHeightPx),
      }));
    },

    previewBottomDockPx(px: number) {
      update(s => ({ ...s, bottomDockPx: clampBottomDock(px) }));
    },

    commit() {
      update(s => {
        persist(s);
        return s;
      });
    },
  };
}

export const workbenchLayoutStore = createWorkbenchLayoutStore();
