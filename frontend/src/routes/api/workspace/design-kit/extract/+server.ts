import { json } from '@sveltejs/kit';
import fs from 'fs';
import path from 'path';
import { evaluatePrototypeGate } from '$lib/workbench/ui-workflow-gate';

const MAX_BYTES = 600_000;
const OUT_DIR = path.join('.vibex', 'prototypes');

function escapeHtml(s: string): string {
	return s
		.replace(/&/g, '&amp;')
		.replace(/</g, '&lt;')
		.replace(/>/g, '&gt;')
		.replace(/"/g, '&quot;');
}

function stripScriptTags(html: string): string {
	return html
		.replace(/<script\b[^>]*>[\s\S]*?<\/script>/gi, '')
		.replace(/\son\w+\s*=\s*(?:"[^"]*"|'[^']*'|[^\s>]+)/gi, '');
}

function buildExtractedDocument(opts: {
	title: string;
	sourcePath: string;
	previewHtml: string;
	sourceExcerpt: string;
}): string {
	return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1"/>
<title>${escapeHtml(opts.title)}</title>
<style>
  body { font-family: system-ui, Segoe UI, sans-serif; margin: 0; background: #0e1016; color: #e2e8f0; line-height: 1.45; }
  header { padding: 12px 16px; border-bottom: 1px solid #303746; background: rgba(28,32,42,.9); }
  h1 { font-size: 14px; margin: 0; color: #7aa2ff; }
  .meta { font-size: 11px; color: #94a3b8; margin-top: 6px; word-break: break-all; }
  main { padding: 16px; }
  .preview { border: 1px solid #303746; border-radius: 12px; padding: 12px; background: rgba(28,32,42,.5); min-height: 80px; }
  pre.src { margin-top: 16px; padding: 12px; border-radius: 10px; background: #0a0c10; border: 1px solid #242b38; overflow: auto; font-size: 11px; color: #a3abb9; white-space: pre-wrap; }
</style>
</head>
<body>
<header>
  <h1>VibeX 剥离原型 · 物料预览</h1>
  <div class="meta">源：${escapeHtml(opts.sourcePath)}</div>
  <div class="meta">自上而下：可渲染预览（已剔 script / 行内事件）与源摘录；请按 .vibex/design/DESIGN.md 对齐真实栈样式。</div>
</header>
<main>
  <div class="preview">${opts.previewHtml}</div>
  <h2 class="meta" style="margin:16px 0 8px">源摘录</h2>
  <pre class="src">${escapeHtml(opts.sourceExcerpt)}</pre>
</main>
</body>
</html>`;
}

function isInsideWorkspace(workspaceRoot: string, candidate: string): boolean {
	const rel = path.relative(workspaceRoot, candidate);
	return rel !== '..' && !rel.startsWith(`..${path.sep}`);
}

export async function POST(event) {
	const body = await event.request.json().catch(() => ({}));
	const workspaceRoot = path.resolve(body.workspace_root || body.workspaceRoot || '');
	const sourceRelRaw = (body.source_path || body.sourcePath || '').trim();
	const outBasename = ((body.out_basename || body.outBasename || '') as string)
		.replace(/[^a-zA-Z0-9\-_]/g, '')
		.replace(/\.html$/i, '');
	const confirm = body.confirm === true;
	const specYaml = String(body.spec_yaml ?? body.specYaml ?? '').trim();

	if (!workspaceRoot || !fs.existsSync(workspaceRoot)) {
		return json({ ok: false, error: 'workspace_root invalid' }, { status: 400 });
	}
	if (!sourceRelRaw) {
		return json({ ok: false, error: 'source_path required (relative to workspace)' }, { status: 400 });
	}
	if (!confirm) {
		return json({ ok: false, error: '需确认：传入 confirm: true 写入 .vibex/prototypes' }, { status: 400 });
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

	const sourceAbs = path.resolve(workspaceRoot, sourceRelRaw);
	const relSrc = path.relative(workspaceRoot, sourceAbs);
	if (relSrc.startsWith('..') || path.isAbsolute(relSrc)) {
		return json({ ok: false, error: 'forbidden: path traversal' }, { status: 403 });
	}
	if (!fs.existsSync(sourceAbs) || !fs.statSync(sourceAbs).isFile()) {
		return json({ ok: false, error: 'source file not found' }, { status: 404 });
	}

	const st = fs.statSync(sourceAbs);
	if (st.size > MAX_BYTES) {
		return json({ ok: false, error: `file too large (max ${MAX_BYTES} bytes)` }, { status: 400 });
	}

	const raw = fs.readFileSync(sourceAbs, 'utf-8');
	const ext = path.extname(sourceAbs).toLowerCase();
	const baseName =
		outBasename ||
		path
			.basename(sourceAbs, ext)
			.replace(/[^\w\-]+/g, '-')
			.replace(/^-+|-+$/g, '')
			.slice(0, 80) ||
		'extracted';

	const outDirAbs = path.resolve(workspaceRoot, OUT_DIR);
	if (!isInsideWorkspace(workspaceRoot, outDirAbs)) {
		return json({ ok: false, error: 'internal path error' }, { status: 500 });
	}
	fs.mkdirSync(outDirAbs, { recursive: true });

	const outName = `${baseName}-extracted.html`;
	const outAbs = path.join(outDirAbs, outName);
	if (!isInsideWorkspace(workspaceRoot, outAbs)) {
		return json({ ok: false, error: 'invalid output path' }, { status: 400 });
	}

	let previewHtml = '';
	if (ext === '.html' || ext === '.htm') {
		const cleaned = stripScriptTags(raw);
		const bodyMatch = cleaned.match(/<body\b[^>]*>([\s\S]*)<\/body>/i);
		previewHtml = (bodyMatch ? bodyMatch[1] : cleaned).trim();
		if (!previewHtml) previewHtml = '<p class="meta">（空主体）</p>';
	} else {
		previewHtml = `<p class="meta">非 HTML 源文件：预览区仅展示占位；请以源摘录为准在物料中重建 markup。</p>`;
	}

	const excerpt = raw.length > 120_000 ? raw.slice(0, 120_000) + '\n… [truncated]' : raw;
	const doc = buildExtractedDocument({
		title: baseName,
		sourcePath: sourceRelRaw.split(path.sep).join('/'),
		previewHtml,
		sourceExcerpt: excerpt,
	});

	fs.writeFileSync(outAbs, doc, 'utf-8');

	const relOut = path.join(OUT_DIR, outName).split(path.sep).join('/');
	const specSnippet = `prototype:
  file: ${relOut}
  status: draft`;

	return json({
		ok: true,
		writtenPath: relOut,
		sourcePath: sourceRelRaw.split(path.sep).join('/'),
		specSnippet,
	});
}
