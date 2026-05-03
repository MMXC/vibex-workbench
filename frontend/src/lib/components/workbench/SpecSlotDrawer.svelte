<script lang="ts">
	import {
		activeSpecSlotSession,
		specSlotSessionStore,
		type SpecSlotSession,
	} from '$lib/stores/spec-slot-session-store';
	import SpecSlotChatPane from '$lib/components/workbench/SpecSlotChatPane.svelte';
	import SpecSlotVisualPane from '$lib/components/workbench/SpecSlotVisualPane.svelte';

	let state = $state<{
		activeKey: string | null;
		drawerOpen: boolean;
		sessions: Record<string, SpecSlotSession>;
	}>({
		activeKey: null,
		drawerOpen: false,
		sessions: {},
	});

	$effect(() => {
		const unsub = specSlotSessionStore.subscribe(value => {
			state = value;
		});
		return unsub;
	});

	const session = $derived.by(() => activeSpecSlotSession(state));
</script>

{#if state.drawerOpen && session}
	<div class="drawer-backdrop" role="presentation">
		<section class="slot-drawer" aria-label="Spec 槽位澄清抽屉">
			<header class="drawer-head">
				<div class="title">
					<span class="eyebrow">Spec Slot Workspace</span>
					<h2>{session.spec.display.title}</h2>
					<p>{session.slot.label} · {session.slot.status} · {session.spec.path}</p>
				</div>
				<div class="tools">
					<button type="button" onclick={() => specSlotSessionStore.compactActive()}>Compact</button>
					<button type="button" onclick={() => specSlotSessionStore.resetActive()}>Reset</button>
					<button type="button" class="primary" onclick={() => specSlotSessionStore.close()}>Close</button>
				</div>
			</header>
			<div class="drawer-body">
				<div class="chat-wrap">
					<SpecSlotChatPane {session} />
				</div>
				<div class="visual-wrap">
					<SpecSlotVisualPane {session} />
				</div>
			</div>
		</section>
	</div>
{/if}

<style>
	.drawer-backdrop {
		position: fixed;
		inset: 42px 18px 24px 74px;
		z-index: 80;
		display: flex;
		pointer-events: none;
	}

	.slot-drawer {
		pointer-events: auto;
		display: flex;
		flex-direction: column;
		width: 100%;
		height: 100%;
		min-height: 0;
		border: 1px solid rgba(122, 162, 255, 0.42);
		border-radius: 20px;
		background: #0b0d12;
		box-shadow: 0 26px 90px rgba(0, 0, 0, 0.58);
		overflow: hidden;
	}

	.drawer-head {
		flex-shrink: 0;
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 18px;
		padding: 15px 16px;
		border-bottom: 1px solid #303746;
		background:
			radial-gradient(circle at 12% 0%, rgba(122, 162, 255, 0.14), transparent 36%),
			rgba(28, 32, 42, 0.94);
	}

	.eyebrow {
		display: block;
		margin-bottom: 5px;
		color: #72d6d0;
		font-family: 'Cascadia Code', ui-monospace, monospace;
		font-size: 10px;
		font-weight: 800;
		letter-spacing: 0.13em;
		text-transform: uppercase;
	}

	h2,
	p {
		margin: 0;
	}

	h2 {
		color: #eef0f5;
		font-size: 16px;
		line-height: 1.25;
	}

	p {
		margin-top: 5px;
		color: #a3abb9;
		font-size: 12px;
	}

	.tools {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.tools button {
		border: 1px solid #465064;
		border-radius: 999px;
		background: rgba(12, 14, 19, 0.52);
		color: #d4d8e3;
		padding: 6px 11px;
		font-size: 11px;
		font-weight: 800;
		cursor: pointer;
	}

	.tools button:hover {
		border-color: #7aa2ff;
		background: rgba(122, 162, 255, 0.13);
		color: #eef0f5;
	}

	.tools .primary {
		border-color: rgba(114, 214, 208, 0.55);
		color: #bdf7f3;
	}

	.drawer-body {
		flex: 1;
		min-height: 0;
		display: grid;
		grid-template-columns: minmax(360px, 0.95fr) minmax(440px, 1.05fr);
	}

	.chat-wrap,
	.visual-wrap {
		min-width: 0;
		min-height: 0;
		overflow: hidden;
	}

	@media (max-width: 980px) {
		.drawer-backdrop {
			inset: 42px 8px 18px 8px;
		}

		.drawer-body {
			grid-template-columns: 1fr;
			grid-template-rows: minmax(360px, 1fr) minmax(300px, 0.9fr);
		}
	}
</style>
