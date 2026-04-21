<!-- Git / 搜索 / 扩展占位 -->
<script lang="ts">
	let { title, hint }: { title: string; hint: string } = $props();
</script>

<div class="placeholder">
	<p class="t">{title}</p>
	<p class="h">{hint}</p>
</div>

<style>
	.placeholder {
		padding: 1rem;
		color: var(--wb-text-sec, #8a8a8e);
		font-size: 12px;
		line-height: 1.5;
	}
	.t {
		margin: 0 0 0.35rem;
		color: var(--wb-text, #e8e8ed);
		font-weight: 600;
		font-size: 13px;
	}
	.h {
		margin: 0;
	}
</style>
