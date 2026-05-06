/**
 * Workbench → Agent Inspector WebSocket（应用级「CDP」事件流）。
 * UI 推送快照；服务端环形缓冲 + GET /api/workbench/inspector/snapshot 可轮询。
 */
import { get } from 'svelte/store';
import { specExplorerStore } from '$lib/stores/spec-explorer-store';
import { threadStore } from '$lib/stores/thread-store';
import { specSlotSessionStore } from '$lib/stores/spec-slot-session-store';

function httpToWsBase(httpUrl: string): string {
	let u = httpUrl.trim().replace(/\/$/, '');
	if (u.startsWith('https://')) return 'wss://' + u.slice('https://'.length);
	if (u.startsWith('http://')) return 'ws://' + u.slice('http://'.length);
	return u;
}

function debounce(fn: () => void, ms: number): () => void {
	let t: ReturnType<typeof setTimeout> | undefined;
	return () => {
		if (t) clearTimeout(t);
		t = setTimeout(fn, ms);
	};
}

export type WorkbenchInspectorHandle = {
	close: () => void;
};

/** 启动 Inspector WS：订阅 store 变更并 debounce 推送快照；mock 模式下 no-op。 */
export function startWorkbenchInspectorStream(
	baseHttpUrl: string,
	opts?: { disabled?: boolean }
): WorkbenchInspectorHandle | void {
	if (opts?.disabled || typeof window === 'undefined') {
		return { close: () => {} };
	}

	const url = `${httpToWsBase(baseHttpUrl)}/api/workbench/inspector/ws`;
	let ws: WebSocket | null = null;
	let reconnectTimer: ReturnType<typeof setTimeout> | undefined;
	let closedByUser = false;
	let attempt = 0;

	function buildEnvelope(): object {
		const se = get(specExplorerStore);
		const th = get(threadStore);
		const slot = get(specSlotSessionStore);
		return {
			v: 1,
			kind: 'snapshot',
			domain: 'App.Workbench',
			method: 'state',
			params: {
				workspaceRoot: se.workspaceRoot || null,
				selectedSpecPath: se.selectedSpecPath,
				centerView: se.centerView,
				leftActivity: se.leftActivity,
				dashboardLevel: se.dashboardLevel,
				specCount: se.specs?.length ?? 0,
				specsLoading: se.specsLoading,
				specsError: se.specsError,
				currentThreadId: th.currentThreadId,
				specSlotDrawerOpen: slot.drawerOpen,
				specSlotActiveKey: slot.activeKey,
				routePath: window.location.pathname + window.location.search,
			},
			ts: new Date().toISOString(),
			source: 'workbench',
		};
	}

	function sendSnapshot() {
		if (!ws || ws.readyState !== WebSocket.OPEN) return;
		try {
			ws.send(JSON.stringify(buildEnvelope()));
		} catch {
			// ignore
		}
	}

	const debouncedSend = debounce(sendSnapshot, 380);

	function connect() {
		if (closedByUser) return;
		try {
			ws = new WebSocket(url);
		} catch {
			scheduleReconnect();
			return;
		}
		ws.onopen = () => {
			attempt = 0;
			sendSnapshot();
		};
		ws.onmessage = () => {
			/* server may broadcast other subscribers */
		};
		ws.onerror = () => {
			/* onclose will reconnect */
		};
		ws.onclose = () => {
			ws = null;
			if (!closedByUser) scheduleReconnect();
		};
	}

	function scheduleReconnect() {
		if (closedByUser) return;
		attempt += 1;
		const delay = Math.min(8000, 400 + attempt * 600);
		reconnectTimer = setTimeout(() => connect(), delay);
	}

	connect();

	const unsubSe = specExplorerStore.subscribe(debouncedSend);
	const unsubTh = threadStore.subscribe(debouncedSend);
	const unsubSlot = specSlotSessionStore.subscribe(debouncedSend);

	const routePoll = window.setInterval(debouncedSend, 4000);

	function close() {
		closedByUser = true;
		if (reconnectTimer) clearTimeout(reconnectTimer);
		window.clearInterval(routePoll);
		unsubSe();
		unsubTh();
		unsubSlot();
		if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
			ws.close();
		}
		ws = null;
	}

	return { close };
}
