import { json } from '@sveltejs/kit';
import fs from 'fs';
import path from 'path';
import { evaluatePrototypeGate } from '$lib/workbench/ui-workflow-gate';

const DESIGN_MD = `# VibeX 设计物料库（DESIGN）

> **门禁**：Agent 生成原型、页面与样式时必须遵循本节；禁止脱离本仓库真实栈与样式约定「自由发挥」导致 UI 漂移。

## 技术栈

- 前端框架：（填写，如 SvelteKit 5）
- 样式方案：（填写，如 CSS / Tailwind / 设计令牌文件路径）
- 组件库：（如有）

## 设计令牌与主题

- 主色 / 辅色 / 背景：
- 圆角、间距、字号阶梯：
- 暗色模式：（是/否，关键变量）

## 布局与组件约定

- 页面栅格、最大宽度、区域命名：
- 可复用组件目录：（如 \`frontend/src/lib/components/...\`）
- **禁止**：未在此声明的随机色值、与设计稿无关的字体栈

## 原型物料输出

- 可交付 HTML 草模归档目录：\`.vibex/prototypes/\`
- 每个 spec 在 \`prototype.file\` 中引用相对工作区根的物料路径（推荐放在 \`.vibex/prototypes/\`）

## 从现有页面剥离

- 使用 Workbench 原型槽内「从页面提取」生成初版物料；再由人工或 Agent 收紧为与生产代码一致的样式子集。

`;

const README_PROTO = `# 原型物料库

本目录存放 **与真实栈对齐** 的 HTML/静态片段，用于 spec 原型槽校验与评审。

- 由 \`.vibex/design/DESIGN.md\` 约束样式与组件边界。
- 命名建议：\`<feature>-<variant>.html\`，并在 manifest 中登记。

`;

const MANIFEST = `{
  "version": 1,
  "entries": []
}
`;

export async function POST(event) {
	const body = await event.request.json().catch(() => ({}));
	const workspaceRoot = path.resolve(body.workspace_root || body.workspaceRoot || '');
	const confirm = body.confirm === true;
	const specYaml = String(body.spec_yaml ?? body.specYaml ?? '').trim();

	if (!workspaceRoot) {
		return json({ ok: false, error: 'workspace_root required' }, { status: 400 });
	}
	if (!fs.existsSync(workspaceRoot)) {
		return json({ ok: false, error: 'workspace not found' }, { status: 400 });
	}
	if (!confirm) {
		return json(
			{
				ok: false,
				error: '需确认：传入 confirm: true 后才会写入 .vibex/design 与 .vibex/prototypes',
			},
			{ status: 400 }
		);
	}

	if (specYaml) {
		const gate = evaluatePrototypeGate(specYaml);
		if (!gate.canCommitPrototype) {
			return json(
				{
					ok: false,
					error: 'prototype_gate_blocked',
					gate_failure: {
						codes: gate.failedCodes,
						checks: gate.checks,
						next_action: gate.nextAction,
					},
				},
				{ status: 409 }
			);
		}
	}

	const written: string[] = [];
	const skipped: string[] = [];

	const designDir = path.join(workspaceRoot, '.vibex', 'design');
	const protoDir = path.join(workspaceRoot, '.vibex', 'prototypes');
	const designFile = path.join(designDir, 'DESIGN.md');
	const readmeFile = path.join(protoDir, 'README.md');
	const manifestFile = path.join(protoDir, 'manifest.json');

	try {
		fs.mkdirSync(designDir, { recursive: true });
		fs.mkdirSync(protoDir, { recursive: true });

		if (!fs.existsSync(designFile)) {
			fs.writeFileSync(designFile, DESIGN_MD, 'utf-8');
			written.push('.vibex/design/DESIGN.md');
		} else {
			skipped.push('.vibex/design/DESIGN.md');
		}

		if (!fs.existsSync(readmeFile)) {
			fs.writeFileSync(readmeFile, README_PROTO, 'utf-8');
			written.push('.vibex/prototypes/README.md');
		} else {
			skipped.push('.vibex/prototypes/README.md');
		}

		if (!fs.existsSync(manifestFile)) {
			fs.writeFileSync(manifestFile, MANIFEST, 'utf-8');
			written.push('.vibex/prototypes/manifest.json');
		} else {
			skipped.push('.vibex/prototypes/manifest.json');
		}
	} catch (e: any) {
		return json({ ok: false, error: e?.message || String(e) }, { status: 500 });
	}

	return json({ ok: true, written, skipped });
}
