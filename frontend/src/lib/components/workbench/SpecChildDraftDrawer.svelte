<!-- 子 spec 推断草稿：落盘前在抽屉中修订 YAML（对齐槽位抽屉布局习惯） -->
<script lang="ts">
	import { specChildDraftStore } from '$lib/stores/spec-child-draft-store';
</script>

{#if $specChildDraftStore.open && $specChildDraftStore.session}
	{@const session = $specChildDraftStore.session}
	{@const busy = $specChildDraftStore.busy}
	{@const err = $specChildDraftStore.error}
	<div class="drawer-backdrop" role="presentation">
		<section class="child-draft-drawer" aria-label="子 spec 草稿抽屉">
			<header class="drawer-head">
				<div class="title">
					<span class="eyebrow">Child Spec Draft</span>
					<h2>新建子 spec · {session.candidate.specName}</h2>
					<p>
						{session.layer === 'L2' ? 'L2→L3 模块' : 'L4→L5 切片'} · {session.relativePath}
					</p>
				</div>
				<div class="tools">
					<button type="button" onclick={() => specChildDraftStore.close()} disabled={busy}>
						关闭
					</button>
					<button
						type="button"
						class="primary"
						onclick={() => void specChildDraftStore.confirmWrite()}
						disabled={busy}
					>
						{busy ? '写入中…' : '写入并挂载到 children'}
					</button>
				</div>
			</header>
			<div class="draft-body">
				<p class="hint">
					以下为模板初稿，可与左侧 Agent 对话完善后再写入；也可直接编辑 YAML。
				</p>
				<textarea
					class="yaml-edit"
					spellcheck="false"
					value={session.draftYaml}
					oninput={(e) =>
						specChildDraftStore.setDraftYaml((e.currentTarget as HTMLTextAreaElement).value)}
					aria-label="子 spec YAML 草稿"
				></textarea>
				{#if err}
					<p class="err">{err}</p>
				{/if}
			</div>
		</section>
	</div>
{/if}

<style>
	.drawer-backdrop {
		position: fixed;
		inset: 42px 18px 24px 74px;
		z-index: 82;
		display: flex;
		pointer-events: none;
	}

	.child-draft-drawer {
		pointer-events: auto;
		display: flex;
		flex-direction: column;
		width: 100%;
		height: 100%;
		min-height: 0;
		border: 1px solid rgba(114, 214, 208, 0.38);
		border-radius: 20px;
		background: #0b0d12;
		box-shadow: 0 26px 90px rgba(0, 0, 0, 0.58);
		overflow: hidden;
	}

	.drawer-head {
		flex-shrink: 0;
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 18px;
		padding: 15px 16px;
		border-bottom: 1px solid #303746;
		background:
			radial-gradient(circle at 12% 0%, rgba(114, 214, 208, 0.12), transparent 36%),
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
		display: inline-flex;
		gap: 8px;
		flex-shrink: 0;
	}

	.tools button {
		padding: 6px 12px;
		border-radius: 8px;
		border: 1px solid #465064;
		background: rgba(18, 22, 30, 0.9);
		color: #dfe6f2;
		font-size: 12px;
		cursor: pointer;
		font-family: inherit;
	}

	.tools button.primary {
		border-color: rgba(114, 214, 208, 0.55);
		background: rgba(114, 214, 208, 0.14);
		color: #dffcf9;
		font-weight: 700;
	}

	.tools button:disabled {
		opacity: 0.55;
		cursor: not-allowed;
	}

	.draft-body {
		flex: 1;
		min-height: 0;
		display: flex;
		flex-direction: column;
		padding: 12px 16px 16px;
		gap: 10px;
	}

	.hint {
		margin: 0;
		font-size: 12px;
		color: #858fa1;
		line-height: 1.45;
	}

	.yaml-edit {
		flex: 1;
		min-height: 220px;
		width: 100%;
		resize: vertical;
		padding: 12px;
		border-radius: 10px;
		border: 1px solid #303746;
		background: #07080c;
		color: #e8ecf5;
		font-family: 'Cascadia Code', ui-monospace, monospace;
		font-size: 12px;
		line-height: 1.45;
	}

	.err {
		margin: 0;
		font-size: 12px;
		color: #f87171;
	}
</style>
