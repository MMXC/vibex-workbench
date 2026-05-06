import {
	DesignKitExtract as wailsDesignKitExtract,
	DesignKitScaffold as wailsDesignKitScaffold,
	DesignKitStatus as wailsDesignKitStatus,
} from '../../../wailsjs/go/main/App.js';

export type DesignKitStatus = {
	ok: boolean;
	workspaceRoot?: string;
	designMdExists?: boolean;
	prototypesDirExists?: boolean;
	designPath?: string;
	prototypesPath?: string;
	error?: string;
};

/** Wails WebView 内 window.runtime 可能晚于 go.main.App 可用；以绑定为准 */
function wailsDesignKitBindingsReady(): boolean {
	return (
		typeof window !== 'undefined' &&
		typeof (window as unknown as { go?: { main?: { App?: { DesignKitExtract?: unknown } } } }).go?.main?.App
			?.DesignKitExtract === 'function'
	);
}

function isWailsDesktopHost(): boolean {
	if (typeof window === 'undefined') return false;
	return window.location.hostname === 'wails.localhost' || window.location.port === '34115';
}

const wailsBindingError =
	'桌面环境未加载 Design Kit 绑定。请使用包含 DesignKit* 方法的最新二进制重新编译（wails build）。';

export async function fetchDesignKitStatus(workspaceRoot: string): Promise<DesignKitStatus> {
	if (!workspaceRoot?.trim()) {
		return { ok: false, error: 'workspaceRoot 为空' };
	}
	if (wailsDesignKitBindingsReady()) {
		const raw = await wailsDesignKitStatus(workspaceRoot);
		return raw as DesignKitStatus;
	}
	if (isWailsDesktopHost()) {
		return { ok: false, error: wailsBindingError };
	}
	const res = await fetch(
		`/api/workspace/design-kit/status?workspaceRoot=${encodeURIComponent(workspaceRoot)}`
	);
	const data = await res.json().catch(() => ({}));
	if (!res.ok) {
		return { ok: false, error: (data as { error?: string }).error || res.statusText };
	}
	return data as DesignKitStatus;
}

export async function scaffoldDesignKit(
	workspaceRoot: string,
	specYaml?: string
): Promise<{
	ok: boolean;
	written?: string[];
	skipped?: string[];
	error?: string;
	gateFailure?: { codes?: string[]; next_action?: string };
}> {
	if (wailsDesignKitBindingsReady()) {
		const raw = await wailsDesignKitScaffold(workspaceRoot, true, specYaml ?? '');
		return raw as {
			ok: boolean;
			written?: string[];
			skipped?: string[];
			error?: string;
			gateFailure?: { codes?: string[]; next_action?: string };
		};
	}
	if (isWailsDesktopHost()) {
		return { ok: false, error: wailsBindingError };
	}
	const res = await fetch('/api/workspace/design-kit/scaffold', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({
			workspace_root: workspaceRoot,
			confirm: true,
			...(specYaml?.trim() ? { spec_yaml: specYaml } : {}),
		}),
	});
	const data = await res.json().catch(() => ({}));
	if (!res.ok) {
		const gf = (data as { gate_failure?: { codes?: string[]; next_action?: string } }).gate_failure;
		return {
			ok: false,
			error: (data as { error?: string }).error ?? res.statusText,
			...(gf ? { gateFailure: gf } : {}),
		};
	}
	return data as { ok: boolean; written?: string[]; skipped?: string[]; error?: string };
}

export async function extractPrototypeFromSource(input: {
	workspaceRoot: string;
	sourcePath: string;
	outBasename?: string;
	/** 当前 spec YAML，用于 Prototype Gate（可选；prototype 槽应传入） */
	specYaml?: string;
}): Promise<{
	ok: boolean;
	writtenPath?: string;
	specSnippet?: string;
	sourcePath?: string;
	error?: string;
	gateFailure?: { codes?: string[]; next_action?: string };
}> {
	if (wailsDesignKitBindingsReady()) {
		const raw = await wailsDesignKitExtract(
			input.workspaceRoot,
			input.sourcePath.trim(),
			input.outBasename?.trim() ?? '',
			true,
			input.specYaml?.trim() ?? ''
		);
		return raw as {
			ok: boolean;
			writtenPath?: string;
			specSnippet?: string;
			sourcePath?: string;
			error?: string;
			gateFailure?: { codes?: string[]; next_action?: string };
		};
	}
	if (isWailsDesktopHost()) {
		return { ok: false, error: wailsBindingError };
	}
	const res = await fetch('/api/workspace/design-kit/extract', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({
			workspace_root: input.workspaceRoot,
			source_path: input.sourcePath.trim(),
			out_basename: input.outBasename?.trim() || undefined,
			confirm: true,
			...(input.specYaml?.trim() ? { spec_yaml: input.specYaml } : {}),
		}),
	});
	const data = await res.json().catch(() => ({}));
	if (!res.ok) {
		const gf = (data as { gate_failure?: { codes?: string[]; next_action?: string } }).gate_failure;
		return {
			ok: false,
			error: (data as { error?: string }).error ?? res.statusText,
			...(gf ? { gateFailure: gf } : {}),
		};
	}
	return data as {
		ok: boolean;
		writtenPath?: string;
		specSnippet?: string;
		sourcePath?: string;
		error?: string;
	};
}
