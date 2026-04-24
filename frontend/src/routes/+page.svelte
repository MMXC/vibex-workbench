<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';

	onMount(() => {
		goto('/workbench');
	});
</script>

<div class="splash">
	<p>VibeX Workbench</p>
	<p class="hint">正在跳转…</p>
</div>

<style>
	.splash {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		height: 100vh;
		color: #888;
		font-family: sans-serif;
	}
	.hint { font-size: 0.95rem; }
</style>
