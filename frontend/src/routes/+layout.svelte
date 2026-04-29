<!-- ============================================================
⚠️  此文件由 spec-to-sveltekit 自动生成
来自: specs/feature
生成时间: 2026-04-20
模式: backend

⚠️  不要直接编辑此文件
修改 specs/ 目录下的 YAML 文件后重新运行 make generate-frontend
============================================================ -->

<script>
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { openDirectoryDialog, eventsOn } from '$lib/wails-runtime';
  import '../app.css';
  let { children } = $props();

  onMount(() => {
    if ('serviceWorker' in navigator) {
      navigator.serviceWorker.register('/sw.js').catch(() => {});
    }

    // Global: menu:open-project → select dir → save → emit event → go to workbench
    eventsOn('menu:open-project', async () => {
      try {
        const dir = await openDirectoryDialog();
        if (!dir) return;
        localStorage.setItem('vibex-workspace-root', dir);
        eventsEmit('workspace:selected', dir);
        goto('/workbench');
      } catch (e) {
        console.error('[layout] menu:open-project error:', e);
      }
    });
  });
</script>

<div style="flex:1;overflow:hidden;display:flex;flex-direction:column;min-height:0;">
  {@render children()}
</div>

