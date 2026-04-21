import path from 'node:path';

/** 仓库根（默认：frontend 的上一级，即 monorepo 根） */
export function getWorkspaceRoot(): string {
	const env = typeof process !== 'undefined' ? process.env.WORKSPACE_ROOT : undefined;
	if (env && env.length > 0) return path.resolve(env);
	return path.resolve(process.cwd(), '..');
}
