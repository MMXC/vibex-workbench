<script lang="ts">
	import { specExplorerStore } from '$lib/stores/spec-explorer-store';

	let { content, openPath }: { content: string; openPath?: string | null } = $props();

	function openCentral() {
		const p = openPath?.trim();
		if (p) specExplorerStore.selectFile(p);
	}
</script>

<div class="tool-bubble">
	<pre>{content}</pre>
	{#if openPath?.trim()}
		<button type="button" class="open-central" onclick={openCentral}>
			在中央打开
			<code>{openPath.trim()}</code>
		</button>
	{/if}
</div>

<style>
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
