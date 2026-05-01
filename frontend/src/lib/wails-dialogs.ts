import { openDirectoryDialog } from './wails-runtime';

/**
 * Wails 菜单/对话框能力统一封装（native-first）。
 * 约定：桌面端优先走 Go binding，浏览器开发模式才走 wrapper fallback。
 */
type NativeFirstOptions = {
	scope?: string;
	methodName: string;
	fallback?: () => Promise<string>;
	allowBrowserFallback?: boolean;
	validate?: (result: string) => boolean;
};

function defaultPathValidator(result: string): boolean {
	return !!result && (result.includes('/') || result.includes('\\'));
}

const NATIVE_OPEN_DIRECTORY_PATH = '/__vibex/native/open-directory';
let openDirectoryHostInFlight: Promise<string> | null = null;

function isWailsDevServerHost(): boolean {
	return window.location.hostname === 'localhost' && window.location.port === '34115';
}

function nativeHostCandidates(): string[] {
	if (window.location.hostname === 'wails.localhost' || window.location.port === '34115') {
		return [NATIVE_OPEN_DIRECTORY_PATH];
	}
	const candidates = [NATIVE_OPEN_DIRECTORY_PATH];
	const current = window.location.origin;
	for (const origin of ['http://wails.localhost:34115', 'http://localhost:34115']) {
		if (origin !== current) candidates.push(`${origin}${NATIVE_OPEN_DIRECTORY_PATH}`);
	}
	return candidates;
}

async function openDirectoryViaHost(validate: (result: string) => boolean): Promise<string> {
	if (openDirectoryHostInFlight) return openDirectoryHostInFlight;
	openDirectoryHostInFlight = openDirectoryViaHostOnce(validate).finally(() => {
		openDirectoryHostInFlight = null;
	});
	return openDirectoryHostInFlight;
}

async function openDirectoryViaHostOnce(validate: (result: string) => boolean): Promise<string> {
	try {
		for (const url of nativeHostCandidates()) {
			try {
				console.warn('[wails-dialogs] POST', url);
				const res = await fetch(url, { method: 'POST' });
				console.warn('[wails-dialogs] host response:', res.status, url);
				if (!res.ok) continue;
				const data = (await res.json()) as { path?: string };
				const path = data.path ?? '';
				console.warn('[wails-dialogs] host path:', path);
				if (validate(path)) return path;
			} catch (e) {
				console.warn('[wails-dialogs] host candidate failed:', url, e);
			}
		}
		return '';
	} catch (e) {
		console.error('[wails-dialogs] host fallback request failed:', e);
		return '';
	}
}

async function callNativeFirst(options: NativeFirstOptions): Promise<string> {
	const {
		scope = 'ui',
		methodName,
		fallback,
		allowBrowserFallback = false,
		validate = defaultPathValidator,
	} = options;

	const w = window as any;
	const fn = w.go?.main?.App?.[methodName];
	if (typeof fn === 'function') {
		try {
			// Wails Go methods that receive context.Context are generated as JS functions
			// with a leading argument. Passing undefined preserves the native binding path.
			const result: string = await fn(undefined);
			if (!result) return '';
			if (validate(result)) return result;
			console.warn(`[${scope}] Wails ${methodName} returned invalid value, ignoring`);
			return '';
		} catch (e) {
			console.error(`[${scope}] Wails ${methodName} direct binding failed`, e);
			if (methodName === 'OpenDirectoryDialog') {
				const path = await openDirectoryViaHost(validate);
				if (path) return path;
			}
			return '';
		}
	}

	// 在 Wails WebView 内若 binding 缺失，不应降级到浏览器 wrapper，
	// 否则可能走到 webkitdirectory 旧链路（仅文件夹名）并触发后续异常。
	const diagnostics = {
		hasWindowGo: !!w.go,
		hasMainApp: !!w.go?.main?.App,
		hasRuntime: !!w.runtime,
		location: window.location.href,
	};

	if (w.runtime || w.go || isWailsDevServerHost() || !allowBrowserFallback) {
		if (methodName === 'OpenDirectoryDialog') {
			const path = await openDirectoryViaHost(validate);
			if (path) return path;
		}
		console.error(
			`[${scope}] Wails binding App.${methodName} is unavailable and host fallback failed`,
			JSON.stringify(diagnostics)
		);
		return '';
	}

	return fallback ? fallback() : '';
}

/** 目录选择：必须走 Wails Go binding；缺失时返回空并打印诊断。 */
export async function openDirectoryNativeFirst(scope = 'ui'): Promise<string> {
	return callNativeFirst({
		scope,
		methodName: 'OpenDirectoryDialog',
	});
}

/** 仅浏览器开发调试使用：允许 showDirectoryPicker fallback。 */
export async function openDirectoryBrowserFallback(scope = 'browser-dev'): Promise<string> {
	return callNativeFirst({
		scope,
		methodName: 'OpenDirectoryDialog',
		fallback: openDirectoryDialog,
		allowBrowserFallback: true,
	});
}
