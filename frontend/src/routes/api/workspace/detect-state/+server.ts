import { json } from '@sveltejs/kit';
import fs from 'fs';
import path from 'path';

export async function POST(event) {
	const body = await event.request.json().catch(() => ({}));
	const workspaceRoot = path.resolve(body.workspace_root || body.workspaceRoot || '');

	if (!workspaceRoot || !fs.existsSync(workspaceRoot)) {
		return json({ state: 'error', signals: [], suggestions: ['目录不存在或无法访问'] });
	}

	const signals: { path: string; exists: boolean; reason: string }[] = [];
	const suggestions: string[] = [];

	// Check 1: specs/ directory
	const specsDir = path.join(workspaceRoot, 'specs');
	const hasSpecsDir = fs.existsSync(specsDir) && fs.statSync(specsDir).isDirectory();
	signals.push({
		path: 'specs/',
		exists: hasSpecsDir,
		reason: hasSpecsDir ? '目录存在' : '目录不存在',
	});

	// Check 2: Makefile
	const hasMakefile = fs.existsSync(path.join(workspaceRoot, 'Makefile'));
	signals.push({
		path: 'Makefile',
		exists: hasMakefile,
		reason: hasMakefile ? '存在' : '不存在',
	});

	// Check 3: frontend/package.json
	const hasPkgJson = fs.existsSync(path.join(workspaceRoot, 'frontend', 'package.json'));
	signals.push({
		path: 'frontend/package.json',
		exists: hasPkgJson,
		reason: hasPkgJson ? '存在' : '不存在',
	});

	// Check 4: generators/gen.py
	const hasGenPy = fs.existsSync(path.join(workspaceRoot, 'generators', 'gen.py'));
	signals.push({
		path: 'generators/gen.py',
		exists: hasGenPy,
		reason: hasGenPy ? '存在' : '不存在',
	});

	// Check 5: verify_specs binary
	const hasVerify = fs.existsSync(path.join(workspaceRoot, 'verify_specs'));
	signals.push({
		path: 'verify_specs',
		exists: hasVerify,
		reason: hasVerify ? '二进制存在' : '二进制不存在（可运行 make verify-specs 编译）',
	});

	// Determine state
	let state: string;
	if (hasSpecsDir && hasMakefile && hasPkgJson && hasGenPy) {
		state = 'ready';
		suggestions.push('✅ 项目就绪，可以进入 Workbench');
	} else if (hasSpecsDir || hasMakefile || hasPkgJson) {
		state = 'partial';
		if (!hasMakefile) suggestions.push('建议: 添加 Makefile（含 generate / lint-specs 目标）');
		if (!hasSpecsDir) suggestions.push('建议: 运行「初始化脚手架」或手动创建 specs/ 目录');
		if (!hasVerify) suggestions.push('建议: 运行 make verify-specs 编译 verify_specs 二进制');
	} else {
		state = 'empty';
		suggestions.push('📦 点击「初始化脚手架」快速创建最小项目结构');
		suggestions.push('或手动创建 specs/ 目录并添加第一份 L1 spec');
	}

	return json({ state, signals, suggestions });
}
