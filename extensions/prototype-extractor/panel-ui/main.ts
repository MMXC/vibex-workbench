// panel-ui/main.ts — 原型操作台：纯原型对话 + PS 工具驱动
// 无 spec 治理，无 relay，一切以扩展侧栏为唯一对话界面

import {
	fetchAgentHealth,
	getAgentBaseUrl,
	getWorkspaceRootForAgent,
	postAgentChat,
	setAgentBaseUrl,
	setWorkspaceRootForAgent,
} from '../lib/agent-client';

// ── Types ──────────────────────────────────────────────────────────

interface ProtoContext {
	workspaceRoot: string;
	prototypeRel: string;
	specRoot: string;
	level: string;
	threadId?: string;
}

interface SlashCmd {
	command: string;
	label_zh: string;
	prompt: string;
}

// ── State ────────────────────────────────────────────────────────

let _ctx: ProtoContext | null = null;
let _sseSource: EventSource | null = null;
let _threadId: string = '';
let _running = false;

// slash 菜单状态
let _slashOpen = false;
let _slashQuery = '';
let _slashIndex = 0;
let _slashPrevQuery = '';
let _draft = '';

// 原型专属 slash 命令
const SLASH_COMMANDS: SlashCmd[] = [
	{
		command: '拆解',
		label_zh: '拆解当前页面布局区块，按层级标注',
		prompt: '拆解当前页面的布局区块，按 L1/L2/L3 层级标注每个区域的职责和关系',
	},
	{
		command: '高亮',
		label_zh: '高亮指定区域并描述其结构',
		prompt: '高亮当前页面的主要布局区域，并描述每个高亮区域的语义和功能',
	},
	{
		command: '标注',
		label_zh: '在原型上叠加标注层，展示信息架构',
		prompt: '在原型上叠加标注层，展示页面结构：标题、导航、内容、状态、交互',
	},
	{
		command: '改造组件',
		label_zh: '将原型区域改造为可交互 web component',
		prompt: '将选定的原型区域改造为可交互的 web component，定义属性、状态和事件接口',
	},
	{
		command: '调整样式',
		label_zh: '调整元素样式和布局，参考业界最佳实践',
		prompt: '调整原型中指定元素的样式和布局，参考业界最佳实践给出 CSS 建议',
	},
	{
		command: '加状态',
		label_zh: '为组件添加 loading/empty/error 等状态',
		prompt: '为原型中的组件添加 states：loading、empty、error、disabled 等状态展示',
	},
	{
		command: '引导',
		label_zh: '生成 onboarding 引导流',
		prompt: '生成 onboarding 引导流：在原型上叠加步骤提示气泡，说明核心操作流程',
	},
	{
		command: '响应式',
		label_zh: '分析并给出移动端适配方案',
		prompt: '分析原型在移动端的适配方案，给出媒体查询和布局调整建议',
	},
];

// ── DOM refs ──────────────────────────────────────────────────────

const threadEl = (): HTMLElement => document.getElementById('agent-thread')!;
const headlineEl = (): HTMLElement => document.getElementById('proto-headline')!;
const statusEl = (): HTMLElement => document.getElementById('proto-status')!;
const inputEl = (): HTMLTextAreaElement => document.getElementById('agent-input') as HTMLTextAreaElement;
const sendBtn = (): HTMLButtonElement => document.getElementById('agent-send') as HTMLButtonElement;

// ── Context 读取 ─────────────────────────────────────────────────

/** 从 content script 写入的 storage 读取原型上下文 */
async function loadProtoContext(): Promise<ProtoContext | null> {
	try {
		const res = await browser.runtime.sendMessage({ type: 'ext:getSessionContext' });
		if (res?.session && typeof res.session === 'object') {
			const s = res.session as Record<string, string>;
			return {
				workspaceRoot: s.workspaceRoot ?? '',
				prototypeRel: s.prototypeRel ?? '',
				specRoot: s.specRoot ?? '',
				level: s.level ?? 'L1',
				threadId: s.threadId,
			};
		}
	} catch { /* ignore */ }
	return null;
}

// ── Chat 渲染 ─────────────────────────────────────────────────────

let _msgCount = 0;

function appendMsg(role: 'user' | 'assistant' | 'system' | 'tool', html: string) {
	const thread = threadEl();
	// 移除占位 hint
	const hint = thread.querySelector('.proto-chat-hint');
	if (hint) hint.remove();

	const div = document.createElement('div');
	div.className = `msg ${role}`;
	div.innerHTML = `<span>${role}</span><pre>${html}</pre>`;
	thread.appendChild(div);
	thread.scrollTop = thread.scrollHeight;
}

function appendUserMsg(text: string) {
	appendMsg('user', escapeHtml(text));
}

function appendAgentMsg(text: string) {
	appendMsg('assistant', mdRender(text));
}

function appendToolResult(toolName: string, result: string) {
	appendMsg('tool', `⚙ ${escapeHtml(toolName)}\n${escapeHtml(result)}`);
}

function appendError(text: string) {
	appendMsg('system', `⚠ ${escapeHtml(text)}`);
}

function setRunning(running: boolean) {
	_running = running;
	sendBtn().disabled = running;
	inputEl().disabled = running;
	statusEl().style.display = running ? '' : 'none';
	statusEl().textContent = running ? 'running' : '';
}

function escapeHtml(s: string): string {
	return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

function mdRender(text: string): string {
	return text
		.replace(/```(\w*)\n?([\s\S]*?)```/g, (_, __, code) =>
			`<pre class="proto-code"><code>${escapeHtml(code.trim())}</code></pre>`)
		.replace(/`([^`]+)`/g, '<code class="proto-inline-code">$1</code>')
		.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
		.replace(/\*([^*]+)\*/g, '<em>$1</em>')
		.replace(/^- (.+)$/gm, '<li>$1</li>')
		.replace(/(<li>.*<\/li>)/s, '<ul>$1</ul>')
		.replace(/\n\n/g, '</p><p>')
		.replace(/\n/g, '<br>')
		.replace(/^<\/p><p>/, '')
		.replace(/<\/p><p>$/, '');
}

// ── Slash 菜单 ─────────────────────────────────────────────────────

function filterSlashCommands(q: string): SlashCmd[] {
	const qq = q.toLowerCase();
	return SLASH_COMMANDS.filter(
		c => c.command.toLowerCase().startsWith(qq) || c.label_zh.toLowerCase().includes(qq)
	).slice(0, 12);
}

function updateSlashFromText(text: string, cursor: number) {
	const before = text.slice(0, cursor);
	const m = before.match(/(?:^|\s)\/([\w-]*)$/);
	if (!m) {
		_slashOpen = false;
		_slashPrevQuery = '';
		_draft = text;
		renderSlashMenu();
		return;
	}
	_slashOpen = true;
	_slashQuery = m[1] ?? '';
	const repStart = before.lastIndexOf('/');
	const filtered = filterSlashCommands(_slashQuery);
	if (_slashQuery !== _slashPrevQuery) {
		_slashIndex = 0;
		_slashPrevQuery = _slashQuery;
	}
	if (filtered.length > 0) {
		_slashIndex = Math.min(_slashIndex, filtered.length - 1);
	}
	_draft = text;
	renderSlashMenu();
}

function renderSlashMenu() {
	const container = document.getElementById('slash-menu');
	if (!container) return;
	if (!_slashOpen) {
		container.innerHTML = '';
		return;
	}
	const filtered = filterSlashCommands(_slashQuery);
	container.innerHTML = filtered.map((cmd, i) => `
		<button type="button" class="slash-item${i === _slashIndex ? ' slash-active' : ''}" data-cmd="${escapeHtml(cmd.command)}">
			<code>/${escapeHtml(cmd.command)}</code>
			<span class="slash-desc">${escapeHtml(cmd.label_zh)}</span>
		</button>
	`).join('');

	container.querySelectorAll<HTMLButtonElement>('.slash-item').forEach(btn => {
		btn.addEventListener('mousedown', e => e.preventDefault());
		btn.addEventListener('click', () => pickSlashCommand(btn.dataset.cmd ?? ''));
	});
}

function pickSlashCommand(command: string) {
	const input = inputEl();
	const before = _draft.slice(0, _draft.lastIndexOf('/'));
	const tail = _draft.slice(input.selectionStart ?? _draft.length);
	_draft = `${before}/${command} ${tail}`;
	_slashOpen = false;
	renderSlashMenu();
	input.value = _draft;
	input.focus();
	const pos = (before.length + command.length + 1);
	input.setSelectionRange(pos, pos);
}

// ── SSE 订阅 ──────────────────────────────────────────────────────

async function startSSE(threadId: string) {
	if (_sseSource) {
		_sseSource.close();
		_sseSource = null;
	}
	_threadId = threadId;

	const base = await getAgentBaseUrl();
	const url = `${base}/api/sse/${encodeURIComponent(threadId)}`;

	try {
		_sseSource = new EventSource(url);

		_sseSource.addEventListener('agent:chunk', (ev: MessageEvent) => {
			try {
				const data = JSON.parse(ev.data) as { content?: string };
				if (data.content) {
					// 追加到最后一条 assistant 消息
					const last = threadEl().lastElementChild;
					if (last?.classList.contains('proto-msg--assistant')) {
						const body = last.querySelector('.proto-msg-body');
						if (body) body.innerHTML += mdRender(data.content);
					}
					threadEl().scrollTop = threadEl().scrollHeight;
				}
			} catch { /* ignore parse errors */ }
		});

		_sseSource.addEventListener('agent:message', (ev: MessageEvent) => {
			try {
				const data = JSON.parse(ev.data) as { content?: string; role?: string };
				if (data.role === 'assistant' && data.content) {
					appendAgentMsg(data.content);
				}
			} catch { /* ignore */ }
		});

		_sseSource.addEventListener('agent:tool:call', (ev: MessageEvent) => {
			try {
				const data = JSON.parse(ev.data) as { name?: string; args?: Record<string, unknown>; callId?: string };
				void handlePSToolCall(data.name, data.args, data.callId);
			} catch { /* ignore */ }
		});

		_sseSource.addEventListener('run.completed', () => {
			setRunning(false);
			inputEl().disabled = false;
		});

		_sseSource.addEventListener('run.failed', (ev: MessageEvent) => {
			setRunning(false);
			try {
				const data = JSON.parse(ev.data) as { error?: string };
				appendError(`Agent 出错: ${data.error ?? 'unknown'}`);
			} catch { /* ignore */ }
		});

		_sseSource.onerror = () => {
			_sseSource?.close();
			_sseSource = null;
			setRunning(false);
			appendError('SSE 连接断开，请重新发送消息');
		};
	} catch (err) {
		appendError(`无法连接 SSE: ${err instanceof Error ? err.message : String(err)}`);
		setRunning(false);
	}
}

// ── PS 工具分发 ──────────────────────────────────────────────────

async function handlePSToolCall(toolName: string | undefined, args: Record<string, unknown> | undefined, callId: string | undefined) {
	if (!toolName || !callId) return;

	const ctx = _ctx;
	if (!ctx?.workspaceRoot) {
		appendError('缺少 workspaceRoot，无法执行 PS 工具');
		return;
	}

	let result: string;
	let toolError: string | undefined;

	try {
		switch (toolName) {
			case 'ps_highlight': {
				const selectors = (args?.selectors as string[] | undefined) ?? [];
				const color = (args?.color as string | undefined) ?? '#7170ff';
				await dispatchToContentScript({ type: 'PS_EXT_MSG_elem:highlight', selectors, color });
				result = `高亮 ${selectors.length} 个元素（${color}）`;
				break;
			}
			case 'ps_annotate': {
				const mode = (args?.mode as string | undefined) ?? 'toggle';
				await dispatchToContentScript({ type: 'PS_EXT_MSG_annotation:show', mode });
				result = `标注已${mode === 'show' ? '显示' : '隐藏'}`;
				break;
			}
			case 'ps_parse': {
				await dispatchToContentScript({ type: 'panel:parsePage' });
				result = '页面已解析';
				break;
			}
			case 'ps_onboard': {
				const steps = (args?.steps as Array<{ title?: string; body?: string; target?: string; ms?: number }> | undefined) ?? [];
				await dispatchToContentScript({ type: 'PS_EXT_MSG_onboard:start', steps });
				result = `引导 ${steps.length} 步`;
				break;
			}
			case 'ps_refresh': {
				await dispatchToContentScript({ type: 'page:reload' });
				result = '页面已刷新';
				break;
			}
			default:
				result = `未知工具: ${toolName}`;
		}
	} catch (e) {
		toolError = e instanceof Error ? e.message : String(e);
		result = `执行失败: ${toolError}`;
	}

	appendToolResult(toolName, result);
	await postToolResult({ toolName, callId, result, error: toolError, threadId: _threadId });
}

async function dispatchToContentScript(msg: Record<string, unknown>): Promise<void> {
	const tabs = await browser.tabs.query({ active: true, currentWindow: true });
	const tabId = tabs[0]?.id;
	if (!tabId) throw new Error('无活跃标签页');
	await browser.tabs.sendMessage(tabId, msg);
}

async function postToolResult(payload: Record<string, string | undefined>): Promise<void> {
	const base = await getAgentBaseUrl();
	await fetch(`${base}/api/extension/tool-result`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(payload),
	}).catch(() => { /* ignore */ });
}

// ── 发送消息 ─────────────────────────────────────────────────────

async function sendMessage() {
	if (_running) return;
	const text = _draft.trim();
	if (!text) return;

	const ctx = _ctx;
	const ws = ctx?.workspaceRoot || (await getWorkspaceRootForAgent());
	if (!ws) {
		appendError('请先在设置中填写工作区根路径');
		return;
	}

	_slashOpen = false;
	renderSlashMenu();
	appendUserMsg(text);
	inputEl().value = '';
	_draft = '';
	setRunning(true);

	const res = await postAgentChat(text, { threadId: _threadId || undefined });
	if (!res.ok) {
		appendError(`发送失败: ${res.error ?? 'unknown'}`);
		setRunning(false);
		return;
	}

	if (res.threadId && res.threadId !== _threadId) {
		_threadId = res.threadId;
		await startSSE(_threadId);
	} else if (!_sseSource && _threadId) {
		await startSSE(_threadId);
	}
}

// ── 工具条按钮 ───────────────────────────────────────────────────

async function refreshPage() {
	await dispatchToContentScript({ type: 'page:reload' });
}

async function toggleAnnotations() {
	await dispatchToContentScript({ type: 'PS_EXT_MSG_annotation:show', mode: 'toggle' });
}

async function startHighlight() {
	const input = inputEl();
	input.value = '请高亮当前页面的主要布局区域';
	await sendMessage();
}

async function startOnboard() {
	await dispatchToContentScript({
		type: 'PS_EXT_MSG_onboard:start',
		steps: [
			{ title: '欢迎', body: '这是原型操作台简介', target: 'body', ms: 3000 },
		],
	});
}

// ── 初始化 ───────────────────────────────────────────────────────

async function init() {
	// 连接设置
	await initAgentBridge();

	// 读取原型上下文
	_ctx = await loadProtoContext();
	if (_ctx?.prototypeRel) {
		headlineEl().textContent = _ctx.prototypeRel;
	}

	// 设置面板折叠
	const bridgeEl = document.getElementById('agent-bridge')!;
	bridgeEl.style.display = 'none';
	document.getElementById('btn-open-settings')?.addEventListener('click', () => {
		const hidden = bridgeEl.style.display === 'none';
		bridgeEl.style.display = hidden ? '' : 'none';
	});

	// 快捷按钮 — 直接发送给 Agent
	document.querySelectorAll<HTMLButtonElement>('.quick-btn').forEach(btn => {
		btn.addEventListener('click', () => {
			const prompt = btn.dataset.prompt ?? '';
			if (prompt && !_running) {
				inputEl().value = prompt;
				void sendMessage();
			}
		});
	});

	// 发送
	sendBtn().addEventListener('click', () => void sendMessage());
	inputEl().addEventListener('input', () => {
		updateSlashFromText(inputEl().value, inputEl().selectionStart ?? 0);
	});
	inputEl().addEventListener('click', () => {
		updateSlashFromText(inputEl().value, inputEl().selectionStart ?? 0);
	});
	inputEl().addEventListener('keydown', (e: KeyboardEvent) => {
		if (_slashOpen) {
			const filtered = filterSlashCommands(_slashQuery);
			if (e.key === 'ArrowDown') {
				e.preventDefault();
				_slashIndex = Math.min(_slashIndex + 1, filtered.length - 1);
				renderSlashMenu();
				return;
			}
			if (e.key === 'ArrowUp') {
				e.preventDefault();
				_slashIndex = Math.max(_slashIndex - 1, 0);
				renderSlashMenu();
				return;
			}
			if (e.key === 'Enter' && !e.ctrlKey) {
				if (filtered.length > 0) {
					e.preventDefault();
					pickSlashCommand(filtered[_slashIndex]?.command ?? filtered[0].command);
				}
				return;
			}
			if (e.key === 'Escape') {
				e.preventDefault();
				_slashOpen = false;
				renderSlashMenu();
				return;
			}
		}
		if (e.key === 'Enter' && e.ctrlKey) {
			e.preventDefault();
			void sendMessage();
		}
	});

	// 工具条
	document.getElementById('btn-highlight')?.addEventListener('click', () => void startHighlight());
	document.getElementById('btn-annotate')?.addEventListener('click', () => void toggleAnnotations());
	document.getElementById('btn-onboard')?.addEventListener('click', () => void startOnboard());
	document.getElementById('btn-refresh')?.addEventListener('click', () => void refreshPage());
}

async function initAgentBridge(): Promise<void> {
	const baseEl = document.getElementById('agent-base-url') as HTMLInputElement | null;
	const wsEl = document.getElementById('agent-workspace-root') as HTMLInputElement | null;
	const statusEl = document.getElementById('agent-bridge-status');
	if (!baseEl || !wsEl || !statusEl) return;

	baseEl.value = await getAgentBaseUrl();
	wsEl.value = await getWorkspaceRootForAgent();

	const setBridgeStatus = (cls: string, text: string) => {
		statusEl.className = 'proto-bridge-status ' + cls;
		statusEl.textContent = text;
	};

	document.getElementById('btn-agent-save')?.addEventListener('click', async () => {
		await setAgentBaseUrl(baseEl.value);
		await setWorkspaceRootForAgent(wsEl.value);
		setBridgeStatus('ok', '已保存');
		_ctx = await loadProtoContext();
	});

	document.getElementById('btn-agent-ping')?.addEventListener('click', async () => {
		await setAgentBaseUrl(baseEl.value);
		setBridgeStatus('', '检测中…');
		const h = await fetchAgentHealth();
		if (h.ok && h.json) {
			const wd = String(h.json.workspace_dir ?? '');
			setBridgeStatus('ok', `在线 model=${String(h.json.model ?? '')} workspace=${wd}`);
		} else {
			setBridgeStatus('err', `失败: ${h.error ?? 'unknown'}`);
		}
	});

	// 启动时探测
	const h = await fetchAgentHealth();
	if (h.ok && h.json) {
		setBridgeStatus('ok', `在线 · ${String(h.json.model ?? '')}`);
	} else {
		setBridgeStatus('err', `Agent 未就绪（可改基址后点连接）`);
	}
}

// 启动
void init();
