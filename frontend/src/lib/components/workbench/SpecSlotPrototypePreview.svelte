<script lang="ts">
	import { onMount } from 'svelte';
	import type { SpecSlotSession } from '$lib/stores/spec-slot-session-store';
	import { specSlotSessionStore } from '$lib/stores/spec-slot-session-store';
	import type { A2UIPrototypePanel } from '$lib/services/tool-routing-client';
	import { getAgentApiBase } from '$lib/runtime/agent-transport';

	let {
		session,
		panel,
		fileSrc,
		snippetSrcdoc,
		wsRoot,
		hero = false,
	}: {
		session: SpecSlotSession;
		panel: A2UIPrototypePanel;
		fileSrc: string;
		snippetSrcdoc: string;
		wsRoot: string;
		hero?: boolean;
	} = $props();

	let protoRefreshTick = $state(0);
	let mainFrame: HTMLIFrameElement | undefined = $state();
	let draftFrame: HTMLIFrameElement | undefined = $state();

	const bustedFileSrc = $derived.by(() => {
		if (!fileSrc) return '';
		const sep = fileSrc.includes('?') ? '&' : '?';
		return `${fileSrc}${sep}_vibex_refresh=${protoRefreshTick}`;
	});

	const agentUi = $derived.by(() => session.prototypeAgentUi ?? null);

	const overlayViewport = $derived.by(() => agentUi?.viewport);

	function postToIframe(win: Window | null | undefined, msg: Record<string, unknown>) {
		win?.postMessage(Object.assign({ source: 'vibex-workbench', version: 1 }, msg), '*');
	}

	function refreshPreview() {
		protoRefreshTick += 1;
	}

	function openInNewTab() {
		if (!fileSrc) return;
		const sep = fileSrc.includes('?') ? '&' : '?';
		window.open(`${fileSrc}${sep}_vibex_tab=${Date.now()}`, '_blank', 'noopener,noreferrer');
	}

	function copyExtensionContext() {
		const root = wsRoot || '';
		const rel = panel.mode === 'workspace_file' ? (panel.fileRel ?? '').trim() : '';
		const agent = getAgentApiBase();
		const text = [
			'## VibeX 原型 · 扩展 / Agent 上下文',
			`- workspace_root: \`${root}\``,
			`- prototype.file: \`${rel}\``,
			`- agent_base: \`${agent}\``,
			`- 预览 URL: \`${fileSrc || '(无)'}\``,
			'',
			'### Chrome 扩展（vibex-prototype-extractor）',
			'1. 在 `extensions/prototype-extractor` 构建或加载未打包扩展。',
			'2. 点击「新标签打开」在本机浏览器打开原型页；扩展侧栏选「本地 HTML」解析当前标签。',
			'',
			'### 页面内桥（供 Agent 高亮意图）',
			'在原型 HTML 中加入：`<script src=".vibex/assets/vibex-prototype-agent-bridge.js" defer></script>`',
			'然后：`vibexPrototypeBridge.highlight([\\'#id\\'])` 或 `vibexPrototypeBridge.onboard([{title,body,target,ms}])`。',
		].join('\n');
		void navigator.clipboard.writeText(text).catch(() => {});
	}

	function demoHighlight() {
		postToIframe(mainFrame?.contentWindow, { kind: 'highlight', selectors: ['body', 'h1', 'main'] });
		postToIframe(draftFrame?.contentWindow, { kind: 'highlight', selectors: ['body', 'h1'] });
	}

	function demoOnboard() {
		const steps = [
			{
				title: 'Onboard 示例 A',
				body: 'Agent 可在实页中引导用户关注关键区域；本步高亮 body。',
				target: 'body',
				ms: 3800,
			},
			{
				title: 'Onboard 示例 B',
				body: '下一步可换 selector；bridge 会向抽屉同步叠层。',
				target: 'h1',
				ms: 3600,
			},
		];
		postToIframe(mainFrame?.contentWindow, { kind: 'onboard', steps });
		postToIframe(draftFrame?.contentWindow, { kind: 'onboard', steps });
	}

	function clearOverlay() {
		specSlotSessionStore.clearPrototypeAgentUiActive();
		postToIframe(mainFrame?.contentWindow, { kind: 'clear' });
		postToIframe(draftFrame?.contentWindow, { kind: 'clear' });
	}

	function prefillBridgeSnippet() {
		const scriptLine =
			'<script src=".vibex/assets/vibex-prototype-agent-bridge.js" defer>' + '<' + '/script>';
		specSlotSessionStore.prefillActiveChat(
			[
				'请在当前 prototype.file 对应 HTML 的 <head> 末尾加入：',
				scriptLine,
				'保存后点右侧「刷新预览」，并在页面控制台验证：',
				"window.vibexPrototypeBridge?.highlight(['body','h1'])",
			].join('\n')
		);
	}

	onMount(() => {
		function handler(ev: MessageEvent) {
			specSlotSessionStore.applyPrototypeBridgeMessage(String(ev.origin ?? ''), ev.data);
		}
		window.addEventListener('message', handler);
		return () => window.removeEventListener('message', handler);
	});
</script>

<div class="proto-kit" class:proto-kit-hero={hero}>
	<div class="proto-toolbar" aria-label="原型预览与扩展">
		<button type="button" class="tb" onclick={refreshPreview} disabled={!fileSrc} title="重新加载 iframe（绕过缓存）">
			刷新预览
		</button>
		<button type="button" class="tb" onclick={openInNewTab} disabled={!fileSrc} title="在系统浏览器打开，便于 Chrome 扩展解析">
			新标签打开
		</button>
		<button type="button" class="tb" onclick={copyExtensionContext} title="复制 Markdown 说明到剪贴板">复制扩展上下文</button>
		<button type="button" class="tb tb-demo" onclick={demoHighlight} title="向 iframe 发 postMessage（需已注入 bridge）">演示高亮</button>
		<button type="button" class="tb tb-demo" onclick={demoOnboard} title="演示分步 onboard">演示 onboard</button>
		<button type="button" class="tb" onclick={prefillBridgeSnippet} title="在左侧对话预填脚本说明">预填 bridge</button>
		<button type="button" class="tb tb-clear" onclick={clearOverlay}>清除叠层</button>
	</div>

	<div class="proto-wrap" class:proto-hero={hero}>
		{#if panel.mode === 'workspace_file'}
			<span class="k">{panel.caption ?? 'Prototype'}</span>
			{#if !wsRoot && panel.fileRel?.trim()}
				<p class="muted proto-fallback">请在工作台选择工作区根目录后再预览 prototype.file。</p>
			{:else if bustedFileSrc}
				<div class="proto-frame-host">
					<iframe
						bind:this={mainFrame}
						class="proto-frame"
						title="prototype file preview"
						sandbox="allow-scripts allow-same-origin"
						referrerpolicy="no-referrer"
						src={bustedFileSrc}
					></iframe>
					{#if agentUi?.rects?.length && overlayViewport}
						<div class="proto-overlay" aria-hidden="true">
							{#each agentUi.rects as r, i (`${r.sel}-${i}`)}
								<div
									class="proto-highlight"
									style:left="{(r.left / overlayViewport.width) * 100}%"
									style:top="{(r.top / overlayViewport.height) * 100}%"
									style:width="{(r.width / overlayViewport.width) * 100}%"
									style:height="{(r.height / overlayViewport.height) * 100}%"
								></div>
							{/each}
						</div>
					{/if}
				</div>
			{:else if panel.fileRel?.trim()}
				<p class="muted proto-fallback">无法生成 prototype.file 预览地址。</p>
			{/if}
			{#if snippetSrcdoc}
				<span class="k proto-draft-k">助手 fenced HTML 草稿</span>
				<div class="proto-frame-host proto-frame-host-draft">
					<iframe
						bind:this={draftFrame}
						class="proto-frame proto-frame-draft"
						title="prototype assistant draft preview"
						sandbox="allow-scripts allow-same-origin"
						referrerpolicy="no-referrer"
						srcdoc={snippetSrcdoc}
					></iframe>
				</div>
			{/if}
		{:else}
			<span class="k">{panel.caption ?? 'Prototype'}</span>
			{#if snippetSrcdoc}
				<div class="proto-frame-host">
					<iframe
						bind:this={mainFrame}
						class="proto-frame"
						title="prototype preview"
						sandbox="allow-scripts allow-same-origin"
						referrerpolicy="no-referrer"
						srcdoc={snippetSrcdoc}
					></iframe>
					{#if agentUi?.rects?.length && overlayViewport}
						<div class="proto-overlay" aria-hidden="true">
							{#each agentUi.rects as r, i (`${r.sel}-${i}`)}
								<div
									class="proto-highlight"
									style:left="{(r.left / overlayViewport.width) * 100}%"
									style:top="{(r.top / overlayViewport.height) * 100}%"
									style:width="{(r.width / overlayViewport.width) * 100}%"
									style:height="{(r.height / overlayViewport.height) * 100}%"
								></div>
							{/each}
						</div>
					{/if}
				</div>
			{:else}
				<p class="muted proto-fallback">暂无原型 HTML，请在对话中生成或配置 prototype.file。</p>
			{/if}
		{/if}
	</div>

	{#if agentUi?.onboard?.title}
		<div class="proto-onboard" role="status">
			{#if agentUi.onboard.step != null && agentUi.onboard.total != null}
				<span class="onb-step">{agentUi.onboard.step}/{agentUi.onboard.total}</span>
			{/if}
			<strong>{agentUi.onboard.title}</strong>
			{#if agentUi.onboard.body}
				<p>{agentUi.onboard.body}</p>
			{/if}
		</div>
	{/if}
</div>

<style>
	.proto-kit {
		display: flex;
		flex-direction: column;
		gap: 8px;
		min-width: 0;
	}

	.proto-kit-hero {
		flex: 1;
		min-height: 0;
	}

	.proto-toolbar {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
		align-items: center;
		padding: 6px 0 2px;
		border-bottom: 1px solid rgba(48, 55, 70, 0.65);
	}

	.tb {
		border: 1px solid #3d4656;
		border-radius: 8px;
		padding: 4px 9px;
		font-size: 10px;
		font-weight: 700;
		cursor: pointer;
		background: #1a1f2a;
		color: #c5cdd8;
	}

	.tb:disabled {
		opacity: 0.38;
		cursor: not-allowed;
	}

	.tb:hover:not(:disabled) {
		background: #232936;
	}

	.tb-demo {
		border-color: rgba(122, 162, 255, 0.45);
		color: #9fc0ff;
	}

	.tb-clear {
		border-color: rgba(248, 113, 113, 0.4);
		color: #fca5a5;
		margin-left: auto;
	}

	.proto-wrap {
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 8px;
		flex-shrink: 0;
	}

	.proto-hero {
		flex: 1;
		min-height: 220px;
	}

	.k {
		display: block;
		color: #72d6d0;
		font-family: 'Cascadia Code', ui-monospace, monospace;
		font-size: 10px;
		font-weight: 800;
		letter-spacing: 0.1em;
		text-transform: uppercase;
	}

	.proto-draft-k {
		margin-top: 6px;
	}

	.proto-frame-host {
		position: relative;
		width: 100%;
		min-height: 200px;
	}

	.proto-frame-host-draft {
		min-height: 160px;
	}

	.proto-frame {
		width: 100%;
		min-height: 200px;
		height: 280px;
		border: 1px solid #242b38;
		border-radius: 12px;
		background: #0a0c10;
	}

	.proto-hero .proto-frame {
		flex: 1;
		min-height: min(52vh, 480px);
		height: auto;
	}

	.proto-frame-draft {
		min-height: 140px;
		height: 220px;
	}

	.proto-overlay {
		position: absolute;
		inset: 0;
		pointer-events: none;
		z-index: 3;
		border-radius: 12px;
		overflow: hidden;
	}

	.proto-highlight {
		position: absolute;
		box-sizing: border-box;
		border: 2px solid rgba(250, 204, 21, 0.95);
		background: rgba(250, 204, 21, 0.12);
		box-shadow: 0 0 0 9999px rgba(0, 0, 0, 0.35);
		border-radius: 6px;
	}

	.proto-onboard {
		margin-top: 4px;
		padding: 10px 12px;
		border-radius: 10px;
		border: 1px solid rgba(114, 214, 208, 0.45);
		background: rgba(114, 214, 208, 0.08);
		color: #e8ecf1;
		font-size: 12px;
		line-height: 1.45;
	}

	.proto-onboard strong {
		display: block;
		margin-bottom: 4px;
		color: #7ee0da;
	}

	.proto-onboard p {
		margin: 0;
		color: #b8c0cc;
	}

	.onb-step {
		float: right;
		font-size: 10px;
		color: #94a3b8;
		font-weight: 800;
	}

	.muted {
		color: #7b8496;
		font-size: 12px;
	}

	.proto-fallback {
		margin: 0;
	}
</style>
