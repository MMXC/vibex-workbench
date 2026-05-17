/**
 * specpilot-status-store.ts
 * SpecPilot 服务状态共享 store。
 * StatusBar 写入（从 /api/specpilot/status 轮询），
 * SpecSlotPrototypePreview 读取（构建 MF iframe URL）。
 */
import { writable } from 'svelte/store';

export interface SpecpilotStatus {
	installed: boolean;
	dcRunning: boolean;
	mfRunning: boolean;
	dcPort: number;
	mfPort: number;
	panelOpen: boolean; // bottom split panel
}

export const specpilotStatusStore = writable<SpecpilotStatus | null>(null);
