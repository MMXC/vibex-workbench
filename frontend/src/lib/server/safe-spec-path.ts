import path from 'node:path';
import { getWorkspaceRoot } from './workspace-root';

/**
 * 仅允许读取仓库内 `specs/` 下的文件，防止路径穿越。
 */
export function resolveSafeSpecsPath(relPath: string): string | null {
	const root = getWorkspaceRoot();
	const specsRoot = path.resolve(root, 'specs');
	const trimmed = relPath.trim().replace(/\\/g, '/');
	if (trimmed.includes('..')) return null;
	const full = path.resolve(root, trimmed);
	const specsResolved = path.resolve(specsRoot);
	if (!full.startsWith(specsResolved)) return null;
	return full;
}
