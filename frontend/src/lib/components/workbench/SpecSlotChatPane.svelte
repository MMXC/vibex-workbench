<script lang="ts">
	import { tick } from 'svelte';
	import { get } from 'svelte/store';
	import { specSlotSessionStore, type SpecSlotSession } from '$lib/stores/spec-slot-session-store';
	import { specExplorerStore } from '$lib/stores/spec-explorer-store';
	import { fetchAgentCommands, type AgentCommand } from '$lib/workbench/agent-commands';

	let { session }: { session: SpecSlotSession } = $props();

	let draft = $state('');
	let scrollEl = $state<HTMLDivElement | undefined>(undefined);
	let consumedPrefillId = $state<string | null>(null);
	let taEl = $state<HTMLTextAreaElement | undefined>(undefined);

	let agentCommands = $state<AgentCommand[]>([]);
	let slashOpen = $state(false);
	let slashReplaceStart = $state(0);
	let slashQuery = $state('');
	let slashIndex = $state(0);
	let slashPrevQuery = $state('');

	$effect(() => {
		session.key;
		consumedPrefillId = null;
	});

	$effect(() => {
		const pre = session.chatPrefill;
		if (!pre || pre.id === consumedPrefillId) return;
		consumedPrefillId = pre.id;
		draft = pre.text;
		queueMicrotask(() => specSlotSessionStore.clearChatPrefillActive());
	});

	$effect(() => {
		session.messages.length;
		tick().then(() => {
			scrollEl?.scrollTo({ top: scrollEl.scrollHeight, behavior: 'smooth' });
		});
	});

	$effect(() => {
		const ws = get(specExplorerStore).workspaceRoot;
		if (!ws) {
			agentCommands = [];
			return;
		}
		let cancelled = false;
		void fetchAgentCommands(ws).then(rows => {
			if (!cancelled) agentCommands = rows;
		});
		return () => {
			cancelled = true;
		};
	});

	const slashFiltered = $derived.by(() => filterSlashCommands(slashQuery));

	function filterSlashCommands(q: string): AgentCommand[] {
		const qq = q.toLowerCase();
		return agentCommands
			.filter(
				c =>
					c.command.toLowerCase().startsWith(qq) ||
					(c.label_zh ?? '').toLowerCase().includes(qq)
			)
			.slice(0, 12);
	}

	function updateSlashFromText(text: string, cursor: number) {
		const before = text.slice(0, cursor);
		const m = before.match(/(?:^|\s)\/([\w-]*)$/);
		if (!m) {
			slashOpen = false;
			slashPrevQuery = '';
			return;
		}
		slashOpen = true;
		slashQuery = m[1] ?? '';
		slashReplaceStart = before.lastIndexOf('/');
		if (slashQuery !== slashPrevQuery) {
			slashIndex = 0;
			slashPrevQuery = slashQuery;
		}
		const filtered = filterSlashCommands(slashQuery);
		const max = filtered.length - 1;
		if (max >= 0) slashIndex = Math.min(slashIndex, max);
	}

	function pickSlashCommand(cmd: string) {
		const el = taEl;
		if (!el) return;
		const head = draft.slice(0, slashReplaceStart);
		const tail = draft.slice(el.selectionEnd);
		draft = `${head}/${cmd} ${tail}`;
		slashOpen = false;
		queueMicrotask(() => {
			const pos = head.length + cmd.length + 2;
			el.focus();
			el.setSelectionRange(pos, pos);
		});
	}

	async function submit() {
		const text = draft.trim();
		if (!text || session.status === 'running') return;
		draft = '';
		slashOpen = false;
		await specSlotSessionStore.submitActive(text);
	}

	function quickPrompt(text: string) {
		draft = text;
	}
</script>

<section class="chat-pane" aria-label="槽位澄清对话">
	<header class="chat-head">
		<div>
			<span class="k">Clarification Chat</span>
			<h3>{session.slot.label} · {session.spec.display.title}</h3>
		</div>
		<span class="status {session.status}">{session.status}</span>
	</header>

	<div class="quick-row" aria-label="快捷澄清问题">
		<button type="button" onclick={() => quickPrompt('请先问我最少必要问题，用来补齐这个槽位。')}>最少问题</button>
		<button type="button" onclick={() => quickPrompt('请先判断当前槽位需要哪些工具能力，不要直接写业务代码。')}>工具能力</button>
		<button type="button" onclick={() => quickPrompt('请生成当前槽位的工具路由图、命中原因、调试步骤和确认步骤。')}>路由图</button>
		<button type="button" onclick={() => quickPrompt('如果现有工具不足，请给出 metadata-only 工具草案，必须包含 ai_fill_area。')}>工具草案</button>
	</div>

	<div class="messages" bind:this={scrollEl}>
		{#each session.messages as message (message.id)}
			<div class="msg {message.role}">
				<span>{message.role}</span>
				<pre>{message.content}</pre>
			</div>
		{/each}
		{#if session.error}
			<div class="msg system">
				<span>error</span>
				<pre>{session.error}</pre>
			</div>
		{/if}
	</div>

	<div class="input">
		<div class="textarea-wrap">
			<textarea
				bind:this={taEl}
				bind:value={draft}
				rows={4}
				placeholder="继续围绕当前 spec 槽位澄清… 输入 / 选择专业 agent（/命令 + 描述）"
				oninput={(event) => {
					const el = event.currentTarget;
					updateSlashFromText(el.value, el.selectionStart ?? 0);
				}}
				onclick={(event) =>
					updateSlashFromText(
						event.currentTarget.value,
						event.currentTarget.selectionStart ?? 0
					)}
				onkeyup={(event) =>
					updateSlashFromText(
						event.currentTarget.value,
						event.currentTarget.selectionStart ?? 0
					)}
				onkeydown={(event) => {
					if (slashOpen && slashFiltered.length > 0) {
						if (event.key === 'ArrowDown') {
							event.preventDefault();
							slashIndex = Math.min(slashIndex + 1, slashFiltered.length - 1);
							return;
						}
						if (event.key === 'ArrowUp') {
							event.preventDefault();
							slashIndex = Math.max(slashIndex - 1, 0);
							return;
						}
						if (event.key === 'Enter' && !event.ctrlKey) {
							event.preventDefault();
							pickSlashCommand(slashFiltered[slashIndex]?.command ?? slashFiltered[0].command);
							return;
						}
						if (event.key === 'Escape') {
							event.preventDefault();
							slashOpen = false;
							return;
						}
					}
					if (event.key === 'Enter' && event.ctrlKey) submit();
				}}
			></textarea>
			{#if slashOpen && slashFiltered.length > 0}
				<ul class="slash-menu" role="listbox" aria-label="专业 agent 命令">
					{#each slashFiltered as cmd, i (cmd.command)}
						<li role="option" aria-selected={i === slashIndex}>
							<button
								type="button"
								class:slash-active={i === slashIndex}
								onmousedown={(e) => e.preventDefault()}
								onclick={() => pickSlashCommand(cmd.command)}
							>
								<code>/{cmd.command}</code>
								<span class="slash-desc">{cmd.label_zh}</span>
							</button>
						</li>
					{/each}
				</ul>
			{/if}
		</div>
		<div class="actions">
			<span>Ctrl+Enter 发送 · / 专业 agent · 会话按 spec+slot 延续</span>
			<button type="button" onclick={submit} disabled={!draft.trim() || session.status === 'running'}>
				{session.status === 'running' ? '运行中…' : '发送'}
			</button>
		</div>
	</div>
</section>

<style>
	.chat-pane {
		display: flex;
		flex-direction: column;
		height: 100%;
		min-height: 0;
		background: #11141b;
		border-right: 1px solid #303746;
	}

	.chat-head {
		flex-shrink: 0;
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		padding: 14px;
		border-bottom: 1px solid #303746;
		background: rgba(28, 32, 42, 0.84);
	}

	.k {
		display: block;
		margin-bottom: 4px;
		color: #72d6d0;
		font-family: 'Cascadia Code', ui-monospace, monospace;
		font-size: 10px;
		font-weight: 800;
		letter-spacing: 0.12em;
		text-transform: uppercase;
	}

	h3 {
		margin: 0;
		color: #eef0f5;
		font-size: 14px;
		line-height: 1.3;
	}

	.status {
		border: 1px solid #465064;
		border-radius: 999px;
		padding: 4px 9px;
		color: #a3abb9;
		font-family: ui-monospace, monospace;
		font-size: 10px;
		font-weight: 800;
	}

	.status.running {
		color: #efc66b;
		border-color: rgba(239, 198, 107, 0.5);
	}

	.status.error {
		color: #f87171;
		border-color: rgba(248, 113, 113, 0.5);
	}

	.quick-row {
		flex-shrink: 0;
		display: flex;
		flex-wrap: wrap;
		gap: 7px;
		padding: 10px 12px;
		border-bottom: 1px solid #303746;
		background: rgba(12, 14, 19, 0.32);
	}

	.quick-row button,
	.actions button {
		border: 1px solid #465064;
		border-radius: 999px;
		background: rgba(28, 32, 42, 0.78);
		color: #d4d8e3;
		font-size: 11px;
		font-weight: 700;
		cursor: pointer;
	}

	.quick-row button {
		padding: 5px 9px;
	}

	.quick-row button:hover,
	.actions button:hover:not(:disabled) {
		border-color: #7aa2ff;
		background: rgba(122, 162, 255, 0.13);
		color: #eef0f5;
	}

	.messages {
		flex: 1;
		min-height: 0;
		overflow: auto;
		padding: 14px;
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.msg {
		display: flex;
		flex-direction: column;
		gap: 5px;
		max-width: 100%;
	}

	.msg.user {
		align-items: flex-end;
	}

	.msg span {
		color: #6f7888;
		font-family: ui-monospace, monospace;
		font-size: 10px;
		text-transform: uppercase;
	}

	.msg pre {
		max-width: min(100%, 760px);
		margin: 0;
		padding: 10px 12px;
		border: 1px solid #303746;
		border-radius: 12px;
		background: #171b24;
		color: #d4d8e3;
		font-family: ui-sans-serif, system-ui, sans-serif;
		font-size: 13px;
		line-height: 1.55;
		white-space: pre-wrap;
		word-break: break-word;
	}

	.msg.user pre {
		background: rgba(122, 162, 255, 0.16);
		border-color: rgba(122, 162, 255, 0.42);
		color: #eef0f5;
	}

	.msg.system pre {
		background: rgba(239, 198, 107, 0.08);
		border-color: rgba(239, 198, 107, 0.34);
	}

	.input {
		flex-shrink: 0;
		padding: 12px;
		border-top: 1px solid #303746;
		background: rgba(28, 32, 42, 0.84);
	}

	.textarea-wrap {
		position: relative;
	}

	textarea {
		width: 100%;
		box-sizing: border-box;
		resize: none;
		border: 1px solid #303746;
		border-radius: 13px;
		background: #0b0d12;
		color: #eef0f5;
		padding: 10px;
		outline: none;
		font-family: inherit;
		line-height: 1.5;
	}

	textarea:focus {
		border-color: #7aa2ff;
		box-shadow: 0 0 0 1px rgba(122, 162, 255, 0.18);
	}

	.actions {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		margin-top: 8px;
	}

	.actions span {
		color: #858fa1;
		font-size: 11px;
	}

	.actions button {
		padding: 6px 14px;
	}

	.actions button:disabled {
		opacity: 0.55;
		cursor: not-allowed;
	}

	.slash-menu {
		position: absolute;
		left: 8px;
		right: 8px;
		bottom: 100%;
		margin: 0 0 6px 0;
		padding: 6px;
		max-height: 220px;
		overflow: auto;
		list-style: none;
		border-radius: 12px;
		border: 1px solid rgba(122, 162, 255, 0.38);
		background: rgba(18, 22, 30, 0.98);
		box-shadow: 0 12px 40px rgba(0, 0, 0, 0.45);
		z-index: 5;
	}

	.slash-menu li {
		margin: 0;
		padding: 0;
	}

	.slash-menu button {
		display: flex;
		align-items: flex-start;
		gap: 10px;
		width: 100%;
		text-align: left;
		border: 1px solid transparent;
		border-radius: 9px;
		padding: 8px 10px;
		background: transparent;
		color: #d4d8e3;
		cursor: pointer;
		font-size: 12px;
		line-height: 1.35;
	}

	.slash-menu button:hover,
	.slash-menu button.slash-active {
		border-color: rgba(122, 162, 255, 0.35);
		background: rgba(122, 162, 255, 0.12);
		color: #eef0f5;
	}

	.slash-menu code {
		flex-shrink: 0;
		font-family: ui-monospace, monospace;
		font-size: 11px;
		color: #7aa2ff;
	}

	.slash-desc {
		flex: 1;
		min-width: 0;
		color: #a3abb9;
		word-break: break-word;
	}
</style>
