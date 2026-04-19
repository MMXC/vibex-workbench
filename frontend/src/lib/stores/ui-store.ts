// UI Store — 管理布局状态和面板可见性
import { writable } from 'svelte/store';

export interface UIState {
  left_sidebar_open: boolean;
  right_sidebar_open: boolean;
  left_sidebar_width: number;
  right_sidebar_width: number;
  active_panel: string | null;
  tool_visibility_layer: number;
  composer_expanded: boolean;
  composer_text: string;
  composer_attachments: unknown[];
  selected_thread_id: string | null;
  selected_artifact_id: string | null;
  selected_tool_invocation_id: string | null;
  modals: unknown[];
  drag_state: unknown | null;
  theme: 'light' | 'dark' | 'system';
}

function createUIStore() {
  const { subscribe, update } = writable<UIState>({
    left_sidebar_open: true,
    right_sidebar_open: false,
    left_sidebar_width: 280,
    right_sidebar_width: 320,
    active_panel: null,
    tool_visibility_layer: 1,
    composer_expanded: false,
    composer_text: '',
    composer_attachments: [],
    selected_thread_id: null,
    selected_artifact_id: null,
    selected_tool_invocation_id: null,
    modals: [],
    drag_state: null,
    theme: 'dark',
  });

  return {
    subscribe,
    toggleLeftSidebar() {
      update(s => ({ ...s, left_sidebar_open: !s.left_sidebar_open }));
    },
    toggleRightSidebar() {
      update(s => ({ ...s, right_sidebar_open: !s.right_sidebar_open }));
    },
    setTheme(theme: UIState['theme']) {
      update(s => ({ ...s, theme }));
    },
    selectThread(id: string | null) {
      update(s => ({ ...s, selected_thread_id: id }));
    },
    selectArtifact(id: string | null) {
      update(s => ({ ...s, selected_artifact_id: id }));
    },
  };
}

export const uiStore = createUIStore();
