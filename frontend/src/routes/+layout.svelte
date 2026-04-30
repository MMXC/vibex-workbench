<!-- ============================================================
⚠️  此文件由 spec-to-sveltekit 自动生成
来自: specs/feature
生成时间: 2026-04-20
模式: backend

⚠️  不要直接编辑此文件
修改 specs/ 目录下的 YAML 文件后重新运行 make generate-frontend
============================================================ -->

<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { openDirectoryDialog, eventsOn } from '$lib/wails-runtime';
	import '../app.css';
	let { children } = $props();

	/** 直接绑 Wails Go binding，跳过 openDirectoryDialog() wrapper */
	async function wailsOpenDirectory(): Promise<string> {
		const rt = (window as any).runtime;
		if (rt && typeof rt.OpenDirectoryDialog === 'function') {
			const result: string = await rt.OpenDirectoryDialog();
			if (result && (result.includes('/') || result.includes('\\'))) {
				return result;
			}
			console.warn('[layout] Wails OpenDirectoryDialog returned no valid path, ignoring');
			return '';
		}
		// Fallback: 浏览器开发模式（Vite dev server）
		return openDirectoryDialog();
	}

	/** 处理"打开项目"：弹目录选择 → 保存 → 跳转 workbench */
	async function handleOpenProject() {
		try {
			const dir = await wailsOpenDirectory();
			if (!dir) return;
			localStorage.setItem('vibex-workspace-root', dir);
			goto('/workbench');
		} catch (e) {
			console.error('[layout] handleOpenProject error:', e);
		}
	}

	onMount(() => {
		if ('serviceWorker' in navigator) {
			navigator.serviceWorker.register('/sw.js').catch(() => {});
		}

		// 接收 Go backend 发出的 menu:open-project（通过 Wails event system）
		eventsOn('menu:open-project', handleOpenProject);
	});
</script>

<div style="flex:1;overflow:hidden;display:flex;flex-direction:column;min-height:0;">
  {@render children()}
</div>

