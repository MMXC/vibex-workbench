<script lang="ts">
	import { specExplorerStore } from '$lib/stores/spec-explorer-store';

	let { content, openPath }: { content: string; openPath?: string | null } = $props();

	// ── Structured payload detection ────────────────────────────────────────

	interface ToolAction {
		label: string;
		action: string;
		payload?: Record<string, unknown>;
		variant?: 'primary' | 'secondary' | 'danger';
	}

	interface SpecPreviewPayload {
		type: 'spec_preview';
		spec?: {
			name?: string;
			title?: string;
			level?: string;
			stages?: string[];
			description?: string;
		};
		summary?: string;
		actions?: ToolAction[];
	}

	interface SpecDiffPayload {
		type: 'spec_diff';
		before?: string;
		after?: string;
		changed_lines?: Array<{ path?: string; line?: number; content?: string }>;
		actions?: ToolAction[];
	}

	interface KanbanPayload {
		type: 'kanban_board';
		columns?: Array<{ label: string; items: string[] }>;
		actions?: ToolAction[];
	}

	interface StatusPayload {
		type: 'status_report';
		status?: string;
		metrics?: Array<{ label: string; value: string; delta?: string }>;
		actions?: ToolAction[];
	}

	interface SpecpilotBootstrapPayload {
		type: 'specpilot_bootstrap';
		dcPort?: number;
		mfPort?: number;
		dcUrl?: string;
		mfUrl?: string;
		mfRemoteUrl?: string;
		message?: string;
	}

	type StructuredPayload =
		| SpecPreviewPayload
		| SpecDiffPayload
		| KanbanPayload
		| StatusPayload
		| SpecpilotBootstrapPayload
		| Record<string, unknown>;

	function parsePayload(raw: string): StructuredPayload | null {
		const trimmed = raw.trim();
		if (!trimmed.startsWith('{') && !trimmed.startsWith('[')) return null;
		try {
			return JSON.parse(trimmed);
		} catch {
			return null;
		}
	}

	const payload = $derived(parsePayload(content));

	function isSpecPreview(p: StructuredPayload | null): p is SpecPreviewPayload {
		return (p as Record<string, unknown>)?.type === 'spec_preview';
	}

	function isSpecDiff(p: StructuredPayload | null): p is SpecDiffPayload {
		return (p as Record<string, unknown>)?.type === 'spec_diff';
	}

	function isKanban(p: StructuredPayload | null): p is KanbanPayload {
		return (p as Record<string, unknown>)?.type === 'kanban_board';
	}

	function isStatusReport(p: StructuredPayload | null): p is StatusPayload {
		return (p as Record<string, unknown>)?.type === 'status_report';
	}

	function isSpecpilotBootstrap(p: StructuredPayload | null): p is SpecpilotBootstrapPayload {
		return (p as Record<string, unknown>)?.type === 'specpilot_bootstrap';
	}

	// ── Action dispatch ─────────────────────────────────────────────────────

	function dispatch(action: ToolAction) {
		const a = action.action;
		if (a === 'open_central' && action.payload?.path) {
			specExplorerStore.selectFile(String(action.payload.path));
		} else if (a === 'open_spec') {
			const p = (payload as Record<string, unknown>)?.spec as Record<string, unknown> | undefined;
			if (p?.name) specExplorerStore.selectFile(String(p.name));
		} else if (a === 'generate') {
			// dispatch to agent command pipeline
			window.dispatchEvent(
				new CustomEvent('spec-slot:action', {
					detail: { action: 'generate', payload: action.payload },
				})
			);
		} else if (a === 'confirm_apply') {
			window.dispatchEvent(
				new CustomEvent('spec-slot:action', {
					detail: { action: 'confirm_apply', payload: action.payload },
				})
			);
		} else {
			// fallback: post message to parent for agent to handle
			window.parent.postMessage({ type: 'genui-action', action }, '*');
		}
	}

	// ── Rendering helpers ───────────────────────────────────────────────────

	const statusVariant = (s: string) => {
		const t = s?.toLowerCase();
		if (t === 'healthy' || t === 'ready' || t === 'done') return 'green';
		if (t === 'warning' || t === 'running') return 'amber';
		return 'red';
	};

	function openCentral() {
		const p = openPath?.trim();
		if (p) specExplorerStore.selectFile(p);
	}
</script>

{#if payload && isSpecpilotBootstrap(payload)}
	<!-- ── SpecPilot Bootstrap Result ── -->
	<div class="genui-card bootstrap">
		<div class="genui-card__header">
			<span class="genui-card__icon">🚀</span>
			<span class="genui-card__title">SpecPilot 已启动</span>
		</div>
		<div class="genui-card__body">
			<div class="bootstrap-badges">
				<div class="service-badge running">
					<span class="badge-dot"></span>
					<span>DC: {payload.dcPort ?? 7890}</span>
				</div>
				<div class="service-badge running">
					<span class="badge-dot"></span>
					<span>MF: {payload.mfPort ?? 5177}</span>
				</div>
			</div>
			{#if payload.mfRemoteUrl}
				<a href={payload.mfRemoteUrl} target="_blank" rel="noopener" class="bootstrap-link">
					打开 MF 原型 →
				</a>
			{/if}
			{#if payload.message}
				<p class="bootstrap-msg">{payload.message}</p>
			{/if}
		</div>
		{#if payload.actions?.length}
			<div class="genui-card__actions">
				{#each payload.actions as act}
					<button
						type="button"
						class="genui-btn"
						class:primary={act.variant === 'primary'}
						onclick={() => dispatch(act)}
					>
						{act.label}
					</button>
				{/each}
			</div>
		{/if}
	</div>

{:else if payload && isSpecPreview(payload)}
	<!-- ── Spec Preview Card ── -->
	<div class="genui-card spec-preview">
		<div class="genui-card__header">
			<span class="genui-card__icon">📋</span>
			<span class="genui-card__title">
				{payload.spec?.title ?? payload.spec?.name ?? 'Spec 预览'}
			</span>
			{#if payload.spec?.level}
				<span class="genui-card__badge">{payload.spec.level}</span>
			{/if}
		</div>
		<div class="genui-card__body">
			{#if payload.spec?.stages?.length}
				<div class="spec-stages">
					{#each payload.spec.stages as stage}
						<span class="stage-chip">{stage}</span>
					{/each}
				</div>
			{/if}
			{#if payload.summary}
				<p class="spec-summary">{payload.summary}</p>
			{/if}
		</div>
		{#if payload.actions?.length}
			<div class="genui-card__actions">
				{#each payload.actions as act}
					<button
						type="button"
						class="genui-btn"
						class:primary={act.variant === 'primary'}
						onclick={() => dispatch(act)}
					>
						{act.label}
					</button>
				{/each}
			</div>
		{/if}
	</div>

{:else if payload && isSpecDiff(payload)}
	<!-- ── Spec Diff Card ── -->
	<div class="genui-card spec-diff">
		<div class="genui-card__header">
			<span class="genui-card__icon">🔄</span>
			<span class="genui-card__title">Spec 变更确认</span>
		</div>
		<div class="genui-card__body">
			{#if payload.before || payload.after}
				<div class="diff-pair">
					<div class="diff-panel before">
						<div class="diff-label">当前</div>
						<pre>{payload.before ?? '(空)'}</pre>
					</div>
					<div class="diff-arrow">→</div>
					<div class="diff-panel after">
						<div class="diff-label">变更后</div>
						<pre>{payload.after ?? '(空)'}</pre>
					</div>
				</div>
			{/if}
			{#if payload.changed_lines?.length}
				<div class="diff-lines">
					{#each payload.changed_lines as line}
						<div class="diff-line">
							{#if line.path}<span class="diff-path">{line.path}</span>{/if}
							{#if line.line != null}<span class="diff-line-num">:{line.line}</span>{/if}
							{#if line.content}<code>{line.content}</code>{/if}
						</div>
					{/each}
				</div>
			{/if}
		</div>
		{#if payload.actions?.length}
			<div class="genui-card__actions">
				{#each payload.actions as act}
					<button
						type="button"
						class="genui-btn"
						class:primary={act.variant === 'primary'}
						class:danger={act.variant === 'danger'}
						onclick={() => dispatch(act)}
					>
						{act.label}
					</button>
				{/each}
			</div>
		{/if}
	</div>

{:else if payload && isKanban(payload)}
	<!-- ── Kanban Board Card ── -->
	<div class="genui-card kanban">
		<div class="genui-card__header">
			<span class="genui-card__icon">📋</span>
			<span class="genui-card__title">任务看板</span>
		</div>
		<div class="genui-card__body kanban-board">
			{#each payload.columns ?? [] as col}
				<div class="kanban-col">
					<div class="kanban-col-label">{col.label}</div>
					{#each col.items as item}
						<div class="kanban-item">{item}</div>
					{/each}
				</div>
			{/each}
		</div>
		{#if payload.actions?.length}
			<div class="genui-card__actions">
				{#each payload.actions as act}
					<button type="button" class="genui-btn" onclick={() => dispatch(act)}>
						{act.label}
					</button>
				{/each}
			</div>
		{/if}
	</div>

{:else if payload && isStatusReport(payload)}
	<!-- ── Status Report Card ── -->
	<div class="genui-card status-report">
		<div class="genui-card__header">
			<span class="genui-card__icon">📊</span>
			<span class="genui-card__title">
				状态报告
				{#if payload.status}
					<span class="status-dot {statusVariant(payload.status)}"></span>
				{/if}
			</span>
		</div>
		<div class="genui-card__body">
			{#if payload.metrics?.length}
				<div class="metrics-grid">
					{#each payload.metrics as m}
						<div class="metric-card">
							<div class="metric-label">{m.label}</div>
							<div class="metric-value">{m.value}</div>
							{#if m.delta}
								<div class="metric-delta {m.delta.startsWith('+') ? 'up' : 'down'}">
									{m.delta}
								</div>
							{/if}
						</div>
					{/each}
				</div>
			{/if}
		</div>
		{#if payload.actions?.length}
			<div class="genui-card__actions">
				{#each payload.actions as act}
					<button type="button" class="genui-btn" onclick={() => dispatch(act)}>
						{act.label}
					</button>
				{/each}
			</div>
		{/if}
	</div>

{:else}
	<!-- ── Fallback: raw tool output ── -->
	<div class="tool-bubble">
		<pre>{content}</pre>
		{#if openPath?.trim()}
			<button type="button" class="open-central" onclick={openCentral}>
				在中央打开
				<code>{openPath.trim()}</code>
			</button>
		{/if}
	</div>
{/if}

<style>
	/* ── Base card ── */
	.genui-card {
		display: flex;
		flex-direction: column;
		gap: 10px;
		max-width: min(100%, 680px);
		border-radius: 14px;
		border: 1px solid rgba(239, 198, 107, 0.34);
		background: rgba(239, 198, 107, 0.06);
		padding: 14px 16px;
	}

	.genui-card__header {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.genui-card__icon {
		font-size: 16px;
		line-height: 1;
	}

	.genui-card__title {
		color: #d4d8e3;
		font-size: 14px;
		font-weight: 600;
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.genui-card__badge {
		background: rgba(122, 162, 255, 0.18);
		border: 1px solid rgba(122, 162, 255, 0.45);
		color: #7aa2ff;
		font-size: 10px;
		font-weight: 700;
		border-radius: 999px;
		padding: 2px 8px;
		letter-spacing: 0.06em;
	}

	.genui-card__body {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.genui-card__actions {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
		padding-top: 6px;
		border-top: 1px solid rgba(255, 255, 255, 0.06);
	}

	/* ── Buttons ── */
	.genui-btn {
		border: 1px solid rgba(122, 162, 255, 0.45);
		background: rgba(122, 162, 255, 0.1);
		color: #c8d3f5;
		font-size: 12px;
		font-weight: 600;
		padding: 6px 12px;
		border-radius: 10px;
		cursor: pointer;
		transition: border-color 0.15s, background 0.15s;
	}

	.genui-btn:hover {
		border-color: #7aa2ff;
		background: rgba(122, 162, 255, 0.18);
		color: #eef0f5;
	}

	.genui-btn.primary {
		background: rgba(122, 162, 255, 0.2);
		border-color: #7aa2ff;
		color: #eef0f5;
	}

	.genui-btn.danger {
		background: rgba(248, 113, 113, 0.12);
		border-color: rgba(248, 113, 113, 0.5);
		color: #f87171;
	}

	/* ── SpecPilot Bootstrap ── */
	.bootstrap-badges {
		display: flex;
		gap: 10px;
		flex-wrap: wrap;
	}

	.service-badge {
		display: flex;
		align-items: center;
		gap: 7px;
		padding: 6px 12px;
		border-radius: 10px;
		border: 1px solid;
		font-size: 12px;
		font-weight: 600;
	}

	.service-badge.running {
		border-color: rgba(72, 214, 208, 0.45);
		background: rgba(72, 214, 208, 0.08);
		color: #72d6d0;
	}

	.badge-dot {
		width: 7px;
		height: 7px;
		border-radius: 50%;
		background: currentColor;
		flex-shrink: 0;
	}

	.bootstrap-link {
		color: #7aa2ff;
		font-size: 13px;
		text-decoration: none;
		font-weight: 600;
	}

	.bootstrap-link:hover {
		text-decoration: underline;
	}

	.bootstrap-msg {
		color: #6f7888;
		font-size: 12px;
		margin: 0;
	}

	/* ── Spec Preview ── */
	.spec-stages {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
	}

	.stage-chip {
		background: rgba(122, 162, 255, 0.1);
		border: 1px solid rgba(122, 162, 255, 0.3);
		color: #7aa2ff;
		font-size: 11px;
		border-radius: 6px;
		padding: 3px 8px;
	}

	.spec-summary {
		color: #9ba3b8;
		font-size: 13px;
		margin: 0;
		line-height: 1.5;
	}

	/* ── Spec Diff ── */
	.diff-pair {
		display: grid;
		grid-template-columns: 1fr auto 1fr;
		gap: 8px;
		align-items: stretch;
	}

	.diff-panel {
		border-radius: 10px;
		border: 1px solid rgba(255, 255, 255, 0.08);
		overflow: hidden;
	}

	.diff-panel.before {
		border-color: rgba(248, 113, 113, 0.3);
	}

	.diff-panel.after {
		border-color: rgba(72, 214, 208, 0.3);
	}

	.diff-label {
		font-size: 10px;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.08em;
		padding: 4px 8px;
	}

	.diff-panel.before .diff-label {
		background: rgba(248, 113, 113, 0.12);
		color: #f87171;
	}

	.diff-panel.after .diff-label {
		background: rgba(72, 214, 208, 0.1);
		color: #72d6d0;
	}

	.diff-panel pre {
		margin: 0;
		padding: 8px 10px;
		font-size: 12px;
		color: #c8d3f5;
		white-space: pre-wrap;
		background: transparent;
		border: none;
		border-radius: 0;
	}

	.diff-arrow {
		color: #465064;
		font-size: 16px;
		display: flex;
		align-items: center;
	}

	.diff-lines {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.diff-line {
		display: flex;
		align-items: baseline;
		gap: 6px;
		font-size: 12px;
	}

	.diff-path {
		color: #7aa2ff;
		font-family: ui-monospace, monospace;
		font-size: 11px;
	}

	.diff-line-num {
		color: #465064;
		font-family: ui-monospace, monospace;
		font-size: 10px;
	}

	.diff-line code {
		color: #c8d3f5;
		font-family: ui-monospace, monospace;
	}

	/* ── Kanban ── */
	.kanban-board {
		display: flex;
		gap: 10px;
		overflow-x: auto;
	}

	.kanban-col {
		min-width: 140px;
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.kanban-col-label {
		font-size: 11px;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.08em;
		color: #6f7888;
		padding: 4px 0;
		border-bottom: 1px solid rgba(255, 255, 255, 0.06);
	}

	.kanban-item {
		background: rgba(28, 32, 42, 0.78);
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 8px;
		padding: 7px 10px;
		font-size: 12px;
		color: #c8d3f5;
	}

	/* ── Status Report ── */
	.metrics-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
		gap: 8px;
	}

	.metric-card {
		background: rgba(28, 32, 42, 0.78);
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 10px;
		padding: 10px 12px;
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.metric-label {
		font-size: 10px;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.08em;
		color: #6f7888;
	}

	.metric-value {
		font-size: 18px;
		font-weight: 800;
		color: #eef0f5;
		font-family: ui-monospace, monospace;
	}

	.metric-delta {
		font-size: 11px;
		font-weight: 600;
	}

	.metric-delta.up {
		color: #72d6d0;
	}

	.metric-delta.down {
		color: #f87171;
	}

	.status-dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		display: inline-block;
	}

	.status-dot.green {
		background: #72d6d0;
		box-shadow: 0 0 6px rgba(72, 214, 208, 0.6);
	}

	.status-dot.amber {
		background: #efc66b;
		box-shadow: 0 0 6px rgba(239, 198, 107, 0.6);
	}

	.status-dot.red {
		background: #f87171;
		box-shadow: 0 0 6px rgba(248, 113, 113, 0.6);
	}

	/* ── Fallback raw ── */
	.tool-bubble {
		display: flex;
		flex-direction: column;
		gap: 8px;
		max-width: min(100%, 760px);
	}

	pre {
		margin: 0;
		padding: 10px 12px;
		border: 1px solid rgba(239, 198, 107, 0.34);
		border-radius: 12px;
		background: rgba(239, 198, 107, 0.08);
		color: #d4d8e3;
		font-family: ui-sans-serif, system-ui, sans-serif;
		font-size: 13px;
		line-height: 1.55;
		white-space: pre-wrap;
		word-break: break-word;
	}

	.open-central {
		align-self: flex-start;
		display: inline-flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 6px;
		padding: 6px 10px;
		border-radius: 10px;
		border: 1px solid rgba(122, 162, 255, 0.45);
		background: rgba(122, 162, 255, 0.1);
		color: #c8d3f5;
		font-size: 12px;
		font-weight: 600;
		cursor: pointer;
	}

	.open-central:hover {
		border-color: #7aa2ff;
		background: rgba(122, 162, 255, 0.18);
		color: #eef0f5;
	}

	code {
		font-family: ui-monospace, monospace;
		font-size: 11px;
		color: #7aa2ff;
		font-weight: 500;
	}
</style>
