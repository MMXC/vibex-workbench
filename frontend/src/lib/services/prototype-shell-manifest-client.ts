import type { ShellManifestRoute } from '$lib/workbench/prototype-shell-manifest';
import {
	PrototypeManifestGet as wailsPrototypeManifestGet,
	PrototypeManifestRegister as wailsPrototypeManifestRegister,
} from '../../../wailsjs/go/main/App.js';

export type PrototypeManifestPayload = {
	ok: boolean;
	exists?: boolean;
	manifestPath?: string;
	data?: { version: number; lastGenerated?: string; routes: ShellManifestRoute[] };
	error?: string;
};

function wailsPrototypeBindingsReady(): boolean {
	return (
		typeof window !== 'undefined' &&
		typeof (window as unknown as { go?: { main?: { App?: { PrototypeManifestGet?: unknown } } } }).go?.main?.App
			?.PrototypeManifestGet === 'function'
	);
}

function isWailsDesktopHost(): boolean {
	if (typeof window === 'undefined') return false;
	return window.location.hostname === 'wails.localhost' || window.location.port === '34115';
}

const wailsBindingError =
	'桌面环境未加载 PrototypeManifest 绑定。请使用包含 PrototypeManifestGet/Register 的最新二进制重新编译（wails build）。';

export async function fetchPrototypeManifest(workspaceRoot: string): Promise<PrototypeManifestPayload> {
	if (!workspaceRoot?.trim()) {
		return { ok: false, error: 'workspaceRoot 为空' };
	}
	const root = workspaceRoot.trim();
	if (wailsPrototypeBindingsReady()) {
		const raw = (await wailsPrototypeManifestGet(root)) as PrototypeManifestPayload;
		return raw;
	}
	if (isWailsDesktopHost()) {
		return { ok: false, error: wailsBindingError };
	}
	const res = await fetch(`/api/workspace/prototype-manifest?workspaceRoot=${encodeURIComponent(root)}`);
	const data = (await res.json().catch(() => ({}))) as PrototypeManifestPayload;
	if (!res.ok) {
		return { ok: false, error: data.error || res.statusText };
	}
	return data;
}

export async function registerPrototypeManifestRoute(payload: {
	workspaceRoot: string;
	specName: string;
	specPath: string;
	displayTitle?: string;
	yamlContent: string;
	entryHtml?: string | null;
}): Promise<{
	ok: boolean;
	manifestPath?: string;
	route?: ShellManifestRoute;
	data?: { version: number; lastGenerated?: string; routes: ShellManifestRoute[] };
	error?: string;
}> {
	const root = payload.workspaceRoot.trim();
	const body = JSON.stringify({
		specName: payload.specName,
		specPath: payload.specPath,
		displayTitle: payload.displayTitle,
		yamlContent: payload.yamlContent ?? '',
		entryHtml: payload.entryHtml ?? null,
	});

	if (wailsPrototypeBindingsReady()) {
		const raw = (await wailsPrototypeManifestRegister(root, body)) as {
			ok: boolean;
			manifestPath?: string;
			route?: ShellManifestRoute;
			data?: { version: number; lastGenerated?: string; routes: ShellManifestRoute[] };
			error?: string;
		};
		return raw;
	}
	if (isWailsDesktopHost()) {
		return { ok: false, error: wailsBindingError };
	}
	const res = await fetch('/api/workspace/prototype-manifest', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({
			action: 'register',
			workspaceRoot: root,
			specName: payload.specName,
			specPath: payload.specPath,
			displayTitle: payload.displayTitle,
			yamlContent: payload.yamlContent ?? '',
			entryHtml: payload.entryHtml ?? null,
		}),
	});
	const data = await res.json().catch(() => ({}));
	if (!res.ok) {
		return { ok: false, error: (data as { error?: string }).error || res.statusText };
	}
	return data as {
		ok: boolean;
		manifestPath?: string;
		route?: ShellManifestRoute;
		data?: { version: number; lastGenerated?: string; routes: ShellManifestRoute[] };
	};
}
