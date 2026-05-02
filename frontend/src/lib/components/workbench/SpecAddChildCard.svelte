<!--
  SpecAddChildCard.svelte — + child spec card with create-child-spec intent
  Behavior B2 from FEAT-spec-graph-expansion: click + card → open agent clarification
  Implements level_rules from FEAT-spec-graph-expansion.yaml

  Level mapping:
    L1 → L2 | target: specs/L2-skeleton/
    L2 → L3 | target: specs/L3-module/
    L3 → L4 | target: specs/L4-feature/
    L4 → L5 | target: specs/L5-slice/
    L5 → none (boundary, no children allowed)
-->
<script lang="ts">
	interface Props {
		currentLevel: string;
		parentName: string;
		parentPath: string;
		existingChildren?: string[];
		onCreateChild?: (intent: CreateChildIntent) => void;
	}

	let {
		currentLevel,
		parentName,
		parentPath,
		existingChildren = [],
		onCreateChild,
	}: Props = $props();

	export type CreateChildIntent = {
		parentName: string;
		parentPath: string;
		targetLevel: string;
		targetDir: string;
		existingChildren: string[];
	};

	// Level → child level + target directory
	const levelRules: Record<string, { childLevel: string; targetDir: string } | null> = {
		L1: { childLevel: 'L2', targetDir: 'specs/L2-skeleton/' },
		L2: { childLevel: 'L3', targetDir: 'specs/L3-module/' },
		L3: { childLevel: 'L4', targetDir: 'specs/L4-feature/' },
		L4: { childLevel: 'L5', targetDir: 'specs/L5-slice/' },
		L5: null, // L5 is the implementation boundary — no further children
	};

	let rule = $derived(levelRules[currentLevel] ?? null);
	let canAddChild = $derived(rule !== null);

	// position of + card: always at child level position
	const lanePercents: Record<string, number> = { L1: 8, L2: 29, L3: 50, L4: 71, L5: 92 };

	let cardLeft = $derived.by(() => {
		if (!rule) return 0;
		return lanePercents[rule.childLevel] ?? 50;
	});

	function handleClick() {
		if (!rule || !canAddChild) return;
		const intent: CreateChildIntent = {
			parentName,
			parentPath,
			targetLevel: rule.childLevel,
			targetDir: rule.targetDir,
			existingChildren,
		};
		onCreateChild?.(intent);
	}
</script>

{#if rule}
	<!-- + child spec card at child level position -->
	<div
		class="add-card"
		style:left="{cardLeft}%"
		style:top="84%"
		role="button"
		tabindex="0"
		title="新增 {rule.childLevel} 子 spec，保存到 {rule.targetDir}"
		onclick={handleClick}
		onkeydown={(e) => {
			if (e.key === 'Enter' || e.key === ' ') handleClick();
		}}
	>
		<span class="k">+ add child</span>
		<strong>+ {rule.childLevel}</strong>
		<small>{rule.targetDir}</small>
	</div>

	<!-- Edge from center to + card -->
	{@const centerLeft = lanePercents[currentLevel] ?? 50}
	<span
		class="edge child-edge"
		style:left="{centerLeft + 9}%"
		style:width="{cardLeft - centerLeft - 13}%"
	></span>
{:else}
	<!-- L5 boundary: no further children allowed -->
	<div class="boundary-card" style:left="{lanePercents['L5']}%" style:top="84%">
		<span class="k">boundary</span>
		<strong>L5</strong>
		<small>文件级实现边界</small>
	</div>
{/if}

<style>
	.add-card {
		position: absolute;
		transform: translate(-50%, -50%);
		width: 142px;
		padding: 0.48rem 0.62rem;
		font-size: 10px;
		line-height: 1.2;
		border-radius: 12px;
		border: 1px dashed #465064;
		background: #10131a;
		color: #d4d4d8;
		text-align: left;
		cursor: pointer;
		opacity: 0.82;
		transition: opacity 160ms ease, border-color 160ms ease, background 160ms ease;
	}

	.add-card:hover {
		opacity: 1;
		border-color: #72d6d0;
		background: rgba(114, 214, 208, 0.08);
	}

	.add-card:focus {
		outline: none;
		border-color: #72d6d0;
		background: rgba(114, 214, 208, 0.12);
	}

	.boundary-card {
		position: absolute;
		transform: translate(-50%, -50%);
		width: 142px;
		padding: 0.48rem 0.62rem;
		font-size: 10px;
		line-height: 1.2;
		border-radius: 12px;
		border: 1px dashed #52525b;
		background: rgba(82, 82, 91, 0.08);
		color: #6f7888;
		text-align: left;
		pointer-events: none;
		opacity: 0.7;
	}

	.k {
		display: block;
		margin-bottom: 4px;
		color: #6f7888;
		font-family: 'Cascadia Code', ui-monospace, monospace;
		font-size: 9px;
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	.add-card .k {
		color: #72d6d0;
	}

	strong {
		display: block;
		color: #eef0f5;
		font-size: 11px;
		font-weight: 800;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		margin-bottom: 2px;
	}

	small {
		display: block;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		color: #858fa1;
		font-size: 9px;
	}

	.add-card small {
		color: #72d6d0;
	}

	.edge {
		position: absolute;
		height: 1.5px;
		background: transparent;
		border-top: 1px dashed #465064;
		pointer-events: none;
		transform: translateY(-50%);
	}

	.child-edge {
		top: 84%;
	}
</style>