<!-- ClarificationPanel — L2 澄清详情抽屉 -->
<!-- 替代 GoalSpecCanvas 中的简单 drawer，支持 Q&A 轮次展示 + 复制/编辑/确认 -->
<script lang="ts">
	export type Phase = 'tech_stack' | 'mvp_prototype' | 'frontend_split' | 'user_stories';
	export type Status = 'draft' | 'in_progress' | 'confirmed' | 'rejected';

	export type SessionSummary = {
		id: string;
		spec_name: string;
		spec_parent: string;
		phase: Phase;
		status: Status;
		rounds: number;
		current_round: number;
		has_draft: boolean;
		updated_at: string;
		confirmed_at?: string;
	};

	export type Round = {
		round: number;
		question: string;
		answer: string;
		confirmed: boolean;
		at: string;
	};

	export type SessionDetail = SessionSummary & {
		rounds: Round[];
		draft: string;
		yaml_content: string;
	};

	export type Props = {
		session: SessionSummary;
		onClose: () => void;
		onConfirmed?: (specName: string) => void;
		onRejected?: (specName: string) => void;
	};

	let { session, onClose, onConfirmed, onRejected }: Props = $props();

	let detail = $state<SessionDetail | null>(null);
	let loading = $state(false);
	let error = $state<string | null>(null);
	let editingDraft = $state(false);
	let draftText = $state('');
	let confirming = $state(false);
	let copied = $state(false);

	const phaseLabels: Record<Phase, string> = {
		tech_stack: '① 技术选型',
		mvp_prototype: '② MVP 原型',
		frontend_split: '③ 前后端分层',
		user_stories: '④ 功能/用户故事',
	};

	const statusLabels: Record<Status, string> = {
		draft: '草稿',
		in_progress: '澄清中',
		confirmed: '已确认',
		rejected: '已撤回',
	};

	const statusColors: Record<Status, string> = {
		draft: '#71717a',
		in_progress: '#f59e0b',
		confirmed: '#22c55e',
		rejected: '#ef4444',
	};

	async function loadDetail() {
		loading = true;
		error = null;
		try {
			const r = await fetch(`/api/clarifications/${encodeURIComponent(session.spec_name)}`);
			if (!r.ok) throw new Error(await r.text());
			detail = await r.json();
			draftText = detail?.draft ?? '';
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			loading = false;
		}
	}

	async function confirmSession() {
		if (!detail?.draft) return;
		confirming = true;
		try {
			const r = await fetch(`/api/clarifications/${encodeURIComponent(session.spec_name)}`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ action: 'confirm', draft: draftText }),
			});
			if (!r.ok) throw new Error(await r.text());
			onConfirmed?.(session.spec_name);
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			confirming = false;
		}
	}

	async function rejectSession() {
		confirming = true;
		try {
			const r = await fetch(`/api/clarifications/${encodeURIComponent(session.spec_name)}`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ action: 'reject' }),
			});
			if (!r.ok) throw new Error(await r.text());
			onRejected?.(session.spec_name);
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			confirming = false;
		}
	}

	function copyYaml() {
		if (!detail?.yaml_content && !draftText) return;
		navigator.clipboard.writeText(detail?.yaml_content ?? draftText);
		copied = true;
		setTimeout(() => (copied = false), 2000);
	}

	function formatTime(ts: string) {
		return new Date(ts).toLocaleString('zh-CN', {
			month: '2-digit', day: '2-digit',
			hour: '2-digit', minute: '2-digit',
		});
	}

	let askQuestion = $state('');
	let askAnswer = $state('');
	let askSending = $state(false);

	async function askRound() {
		if (!askQuestion.trim() || !askAnswer.trim()) return;
		askSending = true;
		error = null;
		try {
			const r = await fetch(`/api/clarifications/${encodeURIComponent(session.spec_name)}`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ action: 'qa', question: askQuestion, answer: askAnswer }),
			});
			if (!r.ok) throw new Error(await r.text());
			const res = await r.json();
			draftText = res.draft ?? draftText;
			askQuestion = '';
			askAnswer = '';
			await loadDetail(); // refresh rounds
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			askSending = false;
		}
	}

	async function saveDraft() {
		if (!draftText.trim()) return;
		try {
			const r = await fetch(`/api/clarifications/${encodeURIComponent(session.spec_name)}`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ action: 'draft', draft: draftText }),
			});
			if (!r.ok) throw new Error(await r.text());
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		}
	}

	$effect(() => {
		loadDetail();
	});
</script>

<div class="clf-panel">
	<!-- Header -->
	<div class="clf-head">
		<div class="clf-head-left">
			<span class="clf-phase-tag" style="color: {statusColors[session.status]}"
				>{phaseLabels[session.phase] ?? session.phase}</span>
			<span class="clf-spec-name">{session.spec_name}</span>
		</div>
		<div class="clf-head-right">
			<span
				class="clf-status-badge"
				style="background: {statusColors[session.status]}22; color: {statusColors[session.status]}; border-color: {statusColors[session.status]}55"
			>
				{statusLabels[session.status]}
			</span>
			<button type="button" class="clf-close" onclick={onClose}>×</button>
		</div>
	</div>

	<!-- Body -->
	<div class="clf-body">
		{#if loading}
			<p class="clf-muted">加载中…</p>
		{:else if error}
			<p class="clf-error">{error}</p>
		{:else if detail}
			<!-- Rounds -->
			<section class="clf-section">
				<h4 class="clf-section-title">澄清轮次（{detail.rounds.length} 轮）</h4>
				{#if detail.rounds.length === 0}
					<p class="clf-muted">尚未开始澄清</p>
				{:else}
					<div class="clf-rounds">
						{#each detail.rounds as round}
							<div class="clf-round" class:confirmed={round.confirmed}>
								<div class="round-num">R{round.round}</div>
								<div class="round-content">
									<p class="round-q">
										<span class="round-label">Q</span>{round.question}
									</p>
									<p class="round-a">
										<span class="round-label">A</span>{round.answer}
									</p>
									<span class="round-time">{formatTime(round.at)}</span>
								</div>
							</div>
						{/each}
					</div>
				{/if}
			</section>

			<!-- Ask: Add new Q&A round -->
			{#if session.status !== 'confirmed' && session.status !== 'rejected'}
				<section class="clf-section clf-ask">
					<h4 class="clf-section-title">添加澄清轮次</h4>
					<textarea
						class="clf-textarea clf-textarea-sm"
						bind:value={askQuestion}
						rows={2}
						placeholder="Q：用户意图 / 关键问题…"
						spellcheck={false}
					></textarea>
					<textarea
						class="clf-textarea clf-textarea-sm"
						bind:value={askAnswer}
						rows={3}
						placeholder="A：澄清结论 / 派生 spec 内容…"
						spellcheck={false}
					></textarea>
					<div class="clf-ask-actions">
						<button
							type="button"
							class="btn-sm"
							disabled={askSending || !askQuestion.trim() || !askAnswer.trim()}
							onclick={askRound}
						>
							{askSending ? '添加中…' : '+ 添加轮次'}
						</button>
					</div>
				</section>
			{/if}

			<!-- L2 Spec 草稿 -->
			<section class="clf-section">
				<div class="clf-section-head">
					<h4 class="clf-section-title">L2 Spec 草稿</h4>
					<div class="clf-section-actions">
						{#if detail.yaml_content || draftText}
							<button type="button" class="btn-sm" onclick={copyYaml}>
								{copied ? '✓ 已复制' : '复制 YAML'}
							</button>
						{/if}
						{#if session.status !== 'confirmed' && session.status !== 'rejected'}
							<button type="button" class="btn-sm" onclick={() => (editingDraft = !editingDraft)}>
								{editingDraft ? '收起' : '编辑'}
							</button>
						{/if}
					</div>
				</div>

				{#if editingDraft}
					<textarea
						class="clf-textarea"
						bind:value={draftText}
						rows={12}
						placeholder="L2 spec YAML 内容…"
						spellcheck={false}
					></textarea>
				{:else if detail.yaml_content || detail.draft}
					<pre class="clf-pre">{(detail.yaml_content || detail.draft)}</pre>
				{:else}
					<p class="clf-muted">尚无草稿内容</p>
				{/if}
			</section>

			<!-- Actions -->
			{#if session.status !== 'confirmed' && session.status !== 'rejected'}
				<div class="clf-actions">
					{#if error}
						<p class="clf-error-sm">{error}</p>
					{/if}
					<button
						type="button"
						class="btn-confirm"
						disabled={confirming || (!detail.draft && !draftText)}
						onclick={confirmSession}
					>
						{confirming ? '确认中…' : '✓ 确认 L2 Spec'}
					</button>
					<button
						type="button"
						class="btn-reject"
						disabled={confirming}
						onclick={rejectSession}
					>
						撤回
					</button>
				</div>
			{:else}
				<div class="clf-confirmed-banner" style="color: {statusColors[session.status]}">
					{#if session.status === 'confirmed' && session.confirmed_at}
						✓ 已确认 · {formatTime(session.confirmed_at)}
					{:else}
						已撤回
					{/if}
				</div>
			{/if}
		{/if}
	</div>
</div>

<style>
	.clf-panel {
		display: flex;
		flex-direction: column;
		height: 100%;
		background: #141416;
	}

	.clf-head {
		flex-shrink: 0;
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.5rem;
		padding: 0.6rem 0.8rem;
		border-bottom: 1px solid #27272a;
	}

	.clf-head-left {
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
		min-width: 0;
	}

	.clf-phase-tag {
		font-size: 10px;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	.clf-spec-name {
		font-size: 13px;
		font-weight: 600;
		color: #fafafa;
		font-family: ui-monospace, monospace;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.clf-head-right {
		display: flex;
		align-items: center;
		gap: 0.4rem;
		flex-shrink: 0;
	}

	.clf-status-badge {
		font-size: 10px;
		padding: 0.2rem 0.5rem;
		border-radius: 4px;
		border: 1px solid;
		font-weight: 500;
	}

	.clf-close {
		background: none;
		border: none;
		color: #71717a;
		font-size: 18px;
		cursor: pointer;
		padding: 0 0.3rem;
		line-height: 1;
	}
	.clf-close:hover {
		color: #e4e4e7;
	}

	.clf-body {
		flex: 1;
		min-height: 0;
		overflow: auto;
		padding: 0.75rem;
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.clf-muted {
		margin: 0;
		color: #71717a;
		font-size: 12px;
	}

	.clf-error {
		margin: 0;
		color: #fca5a5;
		font-size: 12px;
		padding: 0.5rem;
		background: #7f1d1d22;
		border-radius: 4px;
	}

	.clf-section {
		border: 1px solid #27272a;
		border-radius: 8px;
		overflow: hidden;
	}

	.clf-section-head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0.45rem 0.6rem;
		background: #1a1a1a;
		border-bottom: 1px solid #27272a;
	}

	.clf-section-title {
		margin: 0;
		font-size: 11px;
		font-weight: 600;
		color: #a1a1aa;
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}

	.clf-section-actions {
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

	.clf-rounds {
		display: flex;
		flex-direction: column;
		gap: 0;
	}

	.clf-round {
		display: flex;
		gap: 0.6rem;
		padding: 0.6rem;
		border-bottom: 1px solid #1f1f23;
	}
	.clf-round:last-child {
		border-bottom: none;
	}
	.clf-round.confirmed .round-num {
		color: #22c55e;
	}

	.round-num {
		font-size: 10px;
		font-weight: 700;
		color: #71717a;
		flex-shrink: 0;
		width: 20px;
		padding-top: 0.1rem;
	}

	.round-content {
		flex: 1;
		min-width: 0;
	}

	.round-q, .round-a {
		margin: 0 0 0.2rem;
		font-size: 12px;
		line-height: 1.4;
		color: #e4e4e7;
	}

	.round-label {
		font-weight: 700;
		color: #a78bfa;
		margin-right: 0.3rem;
	}

	.round-a .round-label {
		color: #34d399;
	}

	.round-time {
		font-size: 10px;
		color: #52525b;
	}

	.clf-textarea {
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

	.clf-textarea-sm {
		font-size: 11px;
		margin-bottom: 0.4rem;
	}

	.clf-ask {
		background: #0f1014;
		border: 1px solid #27272a;
		border-radius: 6px;
		padding: 0.75rem;
		margin-bottom: 0.75rem;
	}

	.clf-ask-actions {
		display: flex;
		justify-content: flex-end;
		margin-top: 0.3rem;
	}

	.clf-ask .btn-sm {
		background: #3b82f6;
		color: #fff;
		border: none;
		padding: 0.35rem 0.8rem;
		border-radius: 4px;
		font-size: 11px;
		cursor: pointer;
	}

	.clf-ask .btn-sm:disabled {
		background: #27272a;
		color: #52525b;
		cursor: not-allowed;
	}

	.clf-pre {
		margin: 0;
		padding: 0.6rem;
		font-size: 11px;
		font-family: ui-monospace, monospace;
		line-height: 1.45;
		color: #d4d4d8;
		white-space: pre-wrap;
		word-break: break-word;
		background: #0d0d0e;
		max-height: 300px;
		overflow: auto;
	}

	.clf-actions {
		display: flex;
		gap: 0.5rem;
		align-items: center;
		flex-wrap: wrap;
	}

	.clf-error-sm {
		width: 100%;
		margin: 0 0 0.3rem;
		font-size: 11px;
		color: #fca5a5;
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

	.btn-reject {
		padding: 0.55rem 0.8rem;
		background: #27272a;
		border: 1px solid #3f3f46;
		color: #a1a1aa;
		border-radius: 6px;
		font-size: 12px;
		cursor: pointer;
		transition: background 0.15s;
	}
	.btn-reject:hover:not(:disabled) {
		background: #3f3f46;
		color: #e4e4e7;
	}
	.btn-reject:disabled {
		opacity: 0.4;
		cursor: not-allowed;
	}

	.clf-confirmed-banner {
		text-align: center;
		font-size: 12px;
		font-weight: 600;
		padding: 0.6rem;
		border-radius: 6px;
		background: #0d0d0e;
		border: 1px solid #27272a;
	}
</style>
