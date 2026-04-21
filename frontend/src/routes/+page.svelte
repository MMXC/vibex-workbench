<!-- ============================================================
首访 / 通常由 +page.server.ts 307 → /workbench；此页仅兜底（禁用 SSR / 静态导出等）。
⚠️ 不要用 onMount(goto)：会在水合早期触发导航，易出现 hash / hydration 报错。
============================================================ -->

<div class="splash">
	<p>VibeX Workbench</p>
	<p class="hint">若停留在此页，请 <a href="/workbench">打开工作台</a>。</p>
</div>

<style>
	.splash {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 0.75rem;
		height: 100vh;
		color: #888;
	}
	.hint {
		margin: 0;
		font-size: 0.95rem;
	}
	.hint a {
		color: #5856d6;
	}
</style>
