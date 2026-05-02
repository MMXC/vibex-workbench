<!--
  SpecAgentClarification.svelte — agent thread for creating child spec
  Behavior B2-B3 from FEAT-spec-graph-expansion: + card click → clarification thread
  Shows draft, Q&A rounds, confirm/cancel, and validation feedback

  Receives create-child-spec intent from SpecAddChildCard → routes to agent thread
  On confirmation: POST /api/workspace/specs/write → run make validate
  On validation failure: classify errors (yaml_parse_error / parent_not_found / level_mismatch)
-->
<script lang="ts">
	import { browser } from '$app/environment';

	export type CreateChildIntent = {
		parentName: string;
		parentPath: string;
		targetLevel: string;
		targetDir: string;
		existingChildren: string[];
	};

	export type ValidateResult = {
		status: 'pass' | 'fail' | 'warning' | 'running';
		errors?: ValidateError[];
		warnings?: string[];
		duration?: string;
	};

	export type ValidateError = {
		line?: number;
		message: string;
		kind: 'yaml_parse_error' | 'parent_not_found' | 'level_mismatch' | 'missing_l5_boundary' | 'unknown';
	};

	interface Props {
		intent: CreateChildIntent;
		onConfirmed?: (specName: string, specPath: string) => void;
		onCancelled?: () => void;
	}

	let { intent, onConfirmed, onCancelled }: Props = $props();

	// Thread state
	let phase = $state<'drafting' | 'confirming' | 'validating' | 'done'>('drafting');
	let draftText = $state('');
	let draftName = $state('');
	let error = $state<string | null>(null);
	let validating = $state(false);
	let validationResult = $state<ValidateResult | null>(null);
	let copied = $state(false);

	// Phase label
	const phaseLabel = $derived.by(() => {
		switch (phase) {
			case 'drafting': return '澄清中';
			case 'confirming': return '确认写入';
			case 'validating': return '验证中';
			case 'done': return '已完成';
		}
	});

	const phaseColor = $derived.by(() => {
		switch (phase) {
			case 'drafting': return '#f59e0b';
			case 'confirming': return '#7aa2ff';
			case 'validating': return '#a78bfa';
			case 'done': return '#22c55e';
		}
	});

	// Auto-generate suggested filename
	const suggestedFilename = $derived.by(() => {
		const base = draftName || intent.parentName;
		return `SLICE-${base.toLowerCase().replace(/\s+/g, '-')}.yaml`;
	});

	const targetFilePath = $derived(`${intent.targetDir}${suggestedFilename}`);

	// ── context message shown in thread ───────────────────────────
	const contextMessage = $derived(`[系统] 开始为 ${intent.parentName} 创建 ${intent.targetLevel} 子 spec
目标目录：${intent.targetDir}
已有同层 spec：${intent.existingChildren.length > 0 ? intent.existingChildren.join(', ') || '无' : '无'}
请推荐 spec 名称、描述和关键字段。`);

	// ── actions ────────────────────────────────────────────────────
	function classifyError(msg: string): ValidateError['kind'] {
		const m = msg.toLowerCase();
		if (m.includes('yaml') || m.includes('unmarshal') || m.includes('parse')) return 'yaml_parse_error';
		if (m.includes('parent') && (m.includes('not found') || m.includes('missing'))) return 'parent_not_found';
		if (m.includes('level') || m.includes('l5')) return 'level_mismatch';
		if (m.includes('l5') && (m.includes('boundary') || m.includes('file'))) return 'missing_l5_boundary';
		return 'unknown';
	}

	async function confirmAndWrite() {
		if (!draftText.trim()) return;
		phase = 'validating';
		validating = true;
		error = null;

		try {
			// 1. Write spec file
			const writeRes = await fetch('/api/workspace/specs/write', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					path: targetFilePath,
					content: draftText,
				}),
			});

			if (!writeRes.ok) {
				const text = await writeRes.text();
				throw new Error(`写盘失败: ${text}`);
			}

			// 2. Run make validate
			const validateRes = await fetch('/api/workspace/specs/validate', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ path: targetFilePath }),
			});

			const result = await validateRes.json();
			validationResult = {
				status: result.passed ? 'pass' : 'fail',
				errors: (result.errors ?? []).map((e: { line?: number; message: string }) => ({
					line: e.line,
					message: e.message,
					kind: classifyError(e.message),
				})),
				warnings: result.warnings ?? [],
				duration: result.duration ?? '',
			};

			if (result.passed) {
				phase = 'done';
				onConfirmed?.(draftName || suggestedFilename.replace('.yaml', ''), targetFilePath);
			} else {
				// Stay in confirm state so user can edit draft
				phase = 'confirming';
			}
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
			phase = 'confirming';
		} finally {
			validating = false;
		}
	}

	function cancel() {
		phase = 'drafting';
		draftText = '';
		draftName = '';
		error = null;
		validationResult = null;
		onCancelled?.();
	}

	function copyYaml() {
		if (!draftText) return;
		navigator.clipboard.writeText(draftText);
		copied = true;
		setTimeout(() => (copied = false), 2000);
	}

	const errorKindLabel: Record<string, string> = {
		yaml_parse_error: 'YAML 解析错误',
		parent_not_found: 'Parent 未找到',
		level_mismatch: '层级不匹配',
		missing_l5_boundary: 'L5 边界缺失',
		unknown: '未知错误',
	};
</script>

<div class="sac-thread">
	<!-- Header -->
	<div class="sac-head">
		<div class="sac-head-left">
			<span class="sac-phase-tag" style="color: {phaseColor}">{phaseLabel}</span>
			<span class="sac-spec-name">+ {intent.targetLevel}: {intent.parentName}</span>
		</div>
		<div class="sac-head-right">
			<span
				class="sac-level-badge"
				title="目标层级"
				style="background: rgba(122,162,255,.12); color: #9fc0ff; border-color: rgba(122,162,255,.3)"
			>
				→ {intent.targetLevel}
			</span>
			<button type="button" class="sac-close" onclick={cancel}>×</button>
		</div>
	</div>

	<!-- Context message (system prompt shown in thread) -->
	<div class="sac-system-msg">
		<pre>{contextMessage}</pre>
	</div>

	<!-- Body -->
	<div class="sac-body">
		<!-- Draft section -->
		<section class="sac-section">
			<div class="sac-section-head">
				<h4 class="sac-section-title">Spec 草稿</h4>
				<div class="sac-section-actions">
					{#if draftText}
						<button type="button" class="btn-sm" onclick={copyYaml}>
							{copied ? '✓ 已复制' : '复制 YAML'}
						</button>
					{/if}
				</div>
			</div>

			<textarea
				class="sac-textarea"
				bind:value={draftText}
				rows={14}
				placeholder={`# ${intent.targetLevel} spec for ${intent.parentName}\n# 请在下方编辑或粘贴 agent 推荐的 YAML 内容…`}
				spellcheck={false}
				disabled={phase === 'done'}
			></textarea>
		</section>

		<!-- Validation result -->
		{#if validationResult}
			<section class="sac-section sac-validate-result {validationResult.status}">
				<div class="sac-section-head">
					<span class="sac-section-title">
						{#if validationResult.status === 'pass'}✓ 通过{/if}
						{#if validationResult.status === 'fail'}✗ 失败{/if}
						{#if validationResult.status === 'warning'}⚠ 警告{/if}
						{#if validationResult.duration}
							<small class="duration"> · {validationResult.duration}</small>
						{/if}
					</span>
				</div>

				{#if validationResult.errors?.length}
					<div class="error-list">
						{#each validationResult.errors as err}
							<div class="error-item">
								<span class="error-kind">{errorKindLabel[err.kind] ?? err.kind}</span>
								{#if err.line}
									<button
										type="button"
										class="error-line"
										title="跳转到行 {err.line}"
									>
										line {err.line}
									</button>
								{/if}
								<span class="error-msg">{err.message}</span>
							</div>
						{/each}
					</div>
				{/if}

				{#if validationResult.warnings?.length}
					<div class="warning-list">
						{#each validationResult.warnings as w}
							<div class="warning-item">⚠ {w}</div>
						{/each}
					</div>
				{/if}
			</section>
		{/if}

		<!-- Error message -->
		{#if error}
			<div class="sac-error">{error}</div>
		{/if}

		<!-- Actions -->
		<div class="sac-actions">
			{#if phase !== 'done'}
				{#if validationResult?.status === 'fail'}
					<button
						type="button"
						class="btn-confirm"
						onclick={confirmAndWrite}
						disabled={validating || !draftText.trim()}
					>
						重新验证
					</button>
				{:else}
					<button
						type="button"
						class="btn-confirm"
						onclick={confirmAndWrite}
						disabled={validating || !draftText.trim()}
					>
						{#if validating}验证中…{:else}✓ 确认写入{/if}
					</button>
				{/if}
				<button type="button" class="btn-cancel" onclick={cancel}>
					取消
				</button>
			{:else}
				<div class="sac-done-banner">
					✓ 已写入 {targetFilePath}
				</div>
			{/if}
		</div>

		<!-- Target info -->
		<div class="sac-target-info">
			<span>目标路径</span>
			<code>{targetFilePath}</code>
			{#if intent.existingChildren.length > 0}
				<span class="sac-siblings">已有: {intent.existingChildren.join(', ')}</span>
			{/if}
		</div>
	</div>
</div>

<style>
	.sac-thread {
		display: flex;
		flex-direction: column;
		height: 100%;
		background: #141416;
	}

	.sac-head {
		flex-shrink: 0;
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.5rem;
		padding: 0.6rem 0.8rem;
		border-bottom: 1px solid #27272a;
	}

	.sac-head-left {
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
		min-width: 0;
	}

	.sac-phase-tag {
		font-size: 10px;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	.sac-spec-name {
		font-size: 13px;
		font-weight: 600;
		color: #fafafa;
		font-family: ui-monospace, monospace;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.sac-head-right {
		display: flex;
		align-items: center;
		gap: 0.4rem;
		flex-shrink: 0;
	}

	.sac-level-badge {
		font-size: 10px;
		padding: 0.2rem 0.5rem;
		border-radius: 4px;
		border: 1px solid;
		font-weight: 500;
	}

	.sac-close {
		background: none;
		border: none;
		color: #71717a;
		font-size: 18px;
		cursor: pointer;
		padding: 0 0.3rem;
		line-height: 1;
	}
	.sac-close:hover {
		color: #e4e4e7;
	}

	.sac-system-msg {
		flex-shrink: 0;
		padding: 0.5rem 0.8rem;
		background: rgba(122, 162, 255, 0.07);
		border-bottom: 1px solid #27272a;
	}

	.sac-system-msg pre {
		margin: 0;
		font-size: 11px;
		font-family: ui-monospace, monospace;
		color: #9fc0ff;
		line-height: 1.5;
		white-space: pre-wrap;
	}

	.sac-body {
		flex: 1;
		min-height: 0;
		overflow: auto;
		padding: 0.75rem;
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.sac-section {
		border: 1px solid #27272a;
		border-radius: 8px;
		overflow: hidden;
	}

	.sac-section-head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0.45rem 0.6rem;
		background: #1a1a1a;
		border-bottom: 1px solid #27272a;
	}

	.sac-section-title {
		font-size: 11px;
		font-weight: 600;
		color: #a1a1aa;
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}

	.duration {
		font-weight: 400;
		color: #71717a;
		text-transform: none;
		letter-spacing: 0;
	}

	.sac-section-actions {
		display: flex;
		gap: 0.4rem;
	}

	.btn-sm {
		font-size: 10px;
		padding: 0.2rem 0.5rem;
		background: #27272a;
		border: 1px solid #3f3f46;
		color: #e4e4e7;
		border-radius: 4px;
		cursor: pointer;
		transition: background 0.15s;
	}
	.btn-sm:hover {
		background: #3f3f46;
	}

	.sac-textarea {
		width: 100%;
		box-sizing: border-box;
		padding: 0.6rem;
		background: #0d0d0e;
		border: none;
		color: #e4e4e7;
		font-size: 11px;
		font-family: ui-monospace, monospace;
		line-height: 1.45;
		resize: vertical;
		outline: none;
	}
	.sac-textarea:disabled {
		opacity: 0.5;
	}

	/* Validation result */
	.sac-validate-result {
		border-radius: 8px;
		overflow: hidden;
	}

	.sac-validate-result.pass {
		border-color: rgba(34, 197, 94, 0.4);
		background: rgba(34, 197, 94, 0.07);
	}

	.sac-validate-result.fail {
		border-color: rgba(239, 68, 68, 0.4);
		background: rgba(239, 68, 68, 0.07);
	}

	.sac-validate-result.warning {
		border-color: rgba(245, 158, 11, 0.4);
		background: rgba(245, 158, 11, 0.07);
	}

	.error-list {
		padding: 0.5rem 0.6rem;
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
	}

	.error-item {
		display: flex;
		align-items: flex-start;
		gap: 0.4rem;
		font-size: 11px;
		color: #fca5a5;
	}

	.error-kind {
		font-size: 9px;
		padding: 0.1rem 0.4rem;
		border-radius: 3px;
		background: rgba(239, 68, 68, 0.2);
		color: #fca5a5;
		font-weight: 600;
		text-transform: uppercase;
		white-space: nowrap;
		flex-shrink: 0;
	}

	.error-line {
		background: none;
		border: none;
		color: #9fc0ff;
		font-size: 10px;
		cursor: pointer;
		text-decoration: underline;
		font-family: ui-monospace, monospace;
	}

	.error-msg {
		color: #d4d4d8;
	}

	.warning-list {
		padding: 0.5rem 0.6rem;
	}

	.warning-item {
		font-size: 11px;
		color: #fcd34d;
	}

	.sac-error {
		padding: 0.5rem 0.6rem;
		background: rgba(239, 68, 68, 0.1);
		border: 1px solid rgba(239, 68, 68, 0.3);
		border-radius: 6px;
		font-size: 11px;
		color: #fca5a5;
	}

	.sac-actions {
		display: flex;
		gap: 0.5rem;
		align-items: center;
	}

	.btn-confirm {
		flex: 1;
		padding: 0.55rem 1rem;
		background: #166534;
		border: 1px solid #22c55e44;
		color: #86efac;
		border-radius: 6px;
		font-size: 12px;
		font-weight: 600;
		cursor: pointer;
		transition: background 0.15s;
	}
	.btn-confirm:hover:not(:disabled) {
		background: #15803d;
	}
	.btn-confirm:disabled {
		opacity: 0.4;
		cursor: not-allowed;
	}

	.btn-cancel {
		padding: 0.55rem 0.8rem;
		background: #27272a;
		border: 1px solid #3f3f46;
		color: #a1a1aa;
		border-radius: 6px;
		font-size: 12px;
		cursor: pointer;
	}
	.btn-cancel:hover {
		background: #3f3f46;
		color: #e4e4e7;
	}

	.sac-done-banner {
		flex: 1;
		text-align: center;
		font-size: 12px;
		font-weight: 600;
		color: #86efac;
		padding: 0.6rem;
		border-radius: 6px;
		background: rgba(34, 197, 94, 0.1);
		border: 1px solid rgba(34, 197, 94, 0.3);
	}

	.sac-target-info {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		flex-wrap: wrap;
		font-size: 10px;
		color: #71717a;
		padding: 0.4rem 0.6rem;
		background: #0d0d0e;
		border-radius: 6px;
	}

	.sac-target-info span {
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}

	.sac-target-info code {
		color: #9fc0ff;
		font-family: ui-monospace, monospace;
	}

	.sac-siblings {
		color: #858fa1;
		margin-left: 0.5rem;
	}
</style>