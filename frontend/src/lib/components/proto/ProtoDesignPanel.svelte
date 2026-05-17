<script lang="ts">
	import { specpilotStatusStore } from '$lib/stores/specpilot-status-store';

	type Tab = 'spec' | 'dc' | 'ec' | 'mf' | 'adapter';
	let activeTab = $state<Tab>('spec');

	let spStatus = $state<{ dcPort: number; mfPort: number } | null>(null);
	$effect(() => {
		const unsub = specpilotStatusStore.subscribe(s => { if (s) spStatus = s; });
		return unsub;
	});

	const dcBase = $derived(spStatus ? `http://localhost:${spStatus.dcPort}/api` : null);
	const mfBase = $derived(spStatus ? `http://localhost:${spStatus.mfPort}` : null);

	// DC data
	let dcKeys = $state<string[]>([]);
	let dcValues = $state<Record<string, string>>({});
	let newKey = $state('');
	let newVal = $state('');

	// EC data
	let ecEvents = $state<string[]>([]);
	let ecEventName = $state('');
	let ecEventData = $state('{}');
	let ecLog = $state<{ts: string; name: string; data: string}[]>([]);
	let ecSource: EventSource | null = null;

	// MF data
	let mfComponents = $state<{name: string; mfUrl: string; dcKey?: string}[]>([]);

	const tabs: {id: Tab; label: string}[] = [
		{ id: 'spec', label: 'Spec' },
		{ id: 'dc', label: 'DataCenter' },
		{ id: 'ec', label: 'EventCenter' },
		{ id: 'mf', label: 'MF' },
		{ id: 'adapter', label: 'Adapter' },
	];

	async function loadDC() {
		if (!dcBase) return;
		try {
			const r = await fetch(`${dcBase}/dc/list`);
			if (r.ok) {
				const d = await r.json();
				dcKeys = d.keys || [];
				for (const k of dcKeys) {
					const vr = await fetch(`${dcBase}/dc/${encodeURIComponent(k)}`);
					if (vr.ok) {
						const dv = await vr.json();
						dcValues[k] = typeof dv.value === 'object' ? JSON.stringify(dv.value) : String(dv.value);
					}
				}
			}
		} catch {}
	}

	async function dcSet(key: string, val: string) {
		if (!dcBase) return;
		try {
			await fetch(`${dcBase}/dc/${encodeURIComponent(key)}`, {
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ value: val }),
			});
		} catch {}
	}

	async function dcCreate() {
		if (!newKey.trim() || !dcBase) return;
		await dcSet(newKey.trim(), newVal);
		newKey = ''; newVal = '';
		await loadDC();
	}

	async function loadMF() {
		if (!mfBase) return;
		try {
			const r = await fetch(`${mfBase}/api/components`);
			if (r.ok) {
				const d = await r.json();
				mfComponents = d.components || [];
			}
		} catch {}
	}

	function subscribeEC() {
		if (!dcBase || ecSource) return;
		ecSource = new EventSource(`${dcBase}/ec/subscribe`);
		ecSource.onmessage = (e) => {
			try {
				const d = JSON.parse(e.data);
				ecLog = [...ecLog.slice(-49), { ts: new Date().toLocaleTimeString(), name: d.event, data: JSON.stringify(d.data) }];
			} catch {}
		};
	}

	async function ecPublish() {
		if (!dcBase || !ecEventName.trim()) return;
		try {
			await fetch(`${dcBase}/ec/publish`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ event: ecEventName, data: JSON.parse(ecEventData || '{}') }),
			});
			ecEventName = ''; ecEventData = '{}';
		} catch {}
	}

	$effect(() => {
		if (activeTab === 'dc') loadDC();
		if (activeTab === 'ec') { subscribeEC(); return () => { ecSource?.close(); ecSource = null; }; }
		if (activeTab === 'mf') loadMF();
	});
</script>

<div class="proto-panel">
	<!-- Tab bar -->
	<div class="tab-bar" role="tablist">
		{#each tabs as tab}
			<button
				type="button"
				role="tab"
				class="tab"
				class:active={activeTab === tab.id}
				aria-selected={activeTab === tab.id}
				onclick={() => (activeTab = tab.id)}
			>
				{tab.label}
			</button>
		{/each}
	</div>

	<!-- Tab content -->
	<div class="tab-content" role="tabpanel">

		{#if activeTab === 'spec'}
			<div class="panel-section">
				<div class="section-title">当前 Spec</div>
				<div class="spec-card">
					<div class="spec-empty">在左侧面板选择一个 spec 节点<br>此处将显示其详情</div>
				</div>
			</div>

		{:else if activeTab === 'dc'}
			<div class="panel-section">
				<div class="section-title">
					DataCenter
					{#if spStatus}<span class="port-tag">:{spStatus.dcPort}</span>{/if}
				</div>
				{#if dcBase}
					<div class="dc-list">
						{#each dcKeys as k}
							<div class="dc-row">
								<span class="dc-key">{k}</span>
								<input
									class="dc-val"
									type="text"
									value={dcValues[k] ?? ''}
									onblur={(e) => dcSet(k, (e.target as HTMLInputElement).value)}
								/>
							</div>
						{/each}
					</div>
					<div class="dc-create">
						<input class="dc-input" placeholder="key" bind:value={newKey} />
						<input class="dc-input" placeholder="value" bind:value={newVal} />
						<button type="button" class="dc-add-btn" onclick={dcCreate}>+</button>
					</div>
				{:else}
					<div class="empty-msg">DC 未连接</div>
				{/if}
			</div>

		{:else if activeTab === 'ec'}
			<div class="panel-section">
				<div class="section-title">
					EventCenter
					{#if spStatus}<span class="port-tag">:{spStatus.dcPort}</span>{/if}
				</div>
				<div class="ec-publish">
					<input class="ec-input" placeholder="event name" bind:value={ecEventName} />
					<input class="ec-input" placeholder='{"key": "val"}' bind:value={ecEventData} />
					<button type="button" class="ec-btn" onclick={ecPublish}>发布</button>
				</div>
				<div class="ec-log">
					{#each ecLog as entry}
						<div class="ec-entry">
							<span class="ec-ts">{entry.ts}</span>
							<span class="ec-ev">{entry.name}</span>
							<code class="ec-data">{entry.data}</code>
						</div>
					{/each}
				</div>
			</div>

		{:else if activeTab === 'mf'}
			<div class="panel-section">
				<div class="section-title">
					MF 组件
					{#if spStatus}<span class="port-tag">:{spStatus.mfPort}</span>{/if}
				</div>
				{#if mfComponents.length > 0}
					{#each mfComponents as c}
						<div class="mf-card">
							<span class="mf-name">{c.name}</span>
							<span class="mf-url">{c.mfUrl}</span>
							{#if c.dcKey}<span class="mf-dc">DC: {c.dcKey}</span>{/if}
						</div>
					{/each}
				{:else}
					<div class="empty-msg">暂无注册的 MF 组件</div>
				{/if}
			</div>

		{:else if activeTab === 'adapter'}
			<div class="panel-section">
				<div class="section-title">适配器</div>
				<div class="spec-empty">适配器连接状态列表<br>（待实现）</div>
			</div>
		{/if}

	</div>
</div>

<style>
	.proto-panel {
		width: 100%;
		height: 100%;
		display: flex;
		flex-direction: column;
		overflow: hidden;
		background: #0b0d12;
	}

	.tab-bar {
		display: flex;
		border-bottom: 1px solid #1e2030;
		padding: 0 8px;
		flex-shrink: 0;
		overflow-x: auto;
	}
	.tab {
		border: none;
		border-bottom: 2px solid transparent;
		background: transparent;
		color: #556;
		font-size: 11px;
		font-family: inherit;
		padding: 8px 10px;
		cursor: pointer;
		white-space: nowrap;
		transition: color 0.15s, border-color 0.15s;
	}
	.tab:hover { color: #99a; }
	.tab.active { color: #a78bfa; border-bottom-color: #7c3aed; }

	.tab-content {
		flex: 1;
		overflow-y: auto;
		min-height: 0;
	}

	.panel-section { padding: 12px; }
	.section-title {
		font-size: 11px;
		font-weight: 700;
		color: #667;
		text-transform: uppercase;
		letter-spacing: 0.08em;
		margin-bottom: 10px;
	}
	.port-tag { font-weight: 400; color: #334; margin-left: 4px; }

	.spec-card, .spec-empty {
		background: #111318;
		border: 1px solid #1e2030;
		border-radius: 8px;
		padding: 16px;
	}
	.spec-empty { color: #445; font-size: 12px; text-align: center; line-height: 1.6; }
	.empty-msg { color: #445; font-size: 12px; text-align: center; padding: 16px; }

	/* DC */
	.dc-list { display: flex; flex-direction: column; gap: 4px; margin-bottom: 8px; }
	.dc-row { display: flex; gap: 4px; align-items: center; }
	.dc-key { font-size: 10px; color: #72d6d0; font-family: monospace; min-width: 80px; overflow: hidden; text-overflow: ellipsis; }
	.dc-val { flex: 1; background: #111318; border: 1px solid #1e2030; border-radius: 4px; color: #99a; font-size: 11px; font-family: monospace; padding: 3px 6px; min-width: 0; }
	.dc-val:focus { border-color: #7aa2ff; outline: none; }
	.dc-create { display: flex; gap: 4px; }
	.dc-input { flex: 1; background: #111318; border: 1px solid #1e2030; border-radius: 4px; color: #99a; font-size: 11px; font-family: monospace; padding: 4px 6px; }
	.dc-input:focus { border-color: #7aa2ff; outline: none; }
	.dc-add-btn { width: 28px; border: 1px solid #303746; border-radius: 4px; background: transparent; color: #556; font-size: 14px; cursor: pointer; }
	.dc-add-btn:hover { background: rgba(122,162,255,0.15); color: #aab; border-color: #7aa2ff; }

	/* EC */
	.ec-publish { display: flex; gap: 4px; margin-bottom: 8px; flex-wrap: wrap; }
	.ec-input { flex: 1; min-width: 60px; background: #111318; border: 1px solid #1e2030; border-radius: 4px; color: #99a; font-size: 11px; font-family: monospace; padding: 4px 6px; }
	.ec-input:focus { border-color: #7aa2ff; outline: none; }
	.ec-btn { border: 1px solid #7c3aed; border-radius: 4px; background: rgba(124,58,237,0.15); color: #a78bfa; font-size: 11px; cursor: pointer; padding: 4px 8px; }
	.ec-btn:hover { background: rgba(124,58,237,0.25); }
	.ec-log { display: flex; flex-direction: column; gap: 2px; max-height: 300px; overflow-y: auto; }
	.ec-entry { display: flex; gap: 6px; align-items: baseline; font-size: 10px; padding: 3px 0; border-bottom: 1px solid #111318; }
	.ec-ts { color: #334; flex-shrink: 0; font-family: monospace; }
	.ec-ev { color: #a78bfa; flex-shrink: 0; font-weight: 600; }
	.ec-data { color: #556; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

	/* MF */
	.mf-card { background: #111318; border: 1px solid #1e2030; border-radius: 6px; padding: 8px 10px; margin-bottom: 6px; }
	.mf-name { display: block; font-size: 12px; color: #a78bfa; font-weight: 600; }
	.mf-url { display: block; font-size: 10px; color: #445; font-family: monospace; margin-top: 2px; }
	.mf-dc { display: inline-block; font-size: 9px; color: #72d6d0; background: rgba(114,214,208,0.1); border-radius: 3px; padding: 1px 4px; margin-top: 4px; }
</style>
