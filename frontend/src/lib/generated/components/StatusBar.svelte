<script lang="ts">
  interface Props { message?: string; connectionStatus?: 'connected' | 'disconnected' | 'connecting'; currentFile?: string; }
  let { message = '就绪', connectionStatus = 'connected', currentFile = '' }: Props = $props();
  const statusColors: Record<string, string> = { connected: '#6BCB77', disconnected: '#FF6B6B', connecting: '#FFD93D' };
  let time = $state(new Date().toLocaleTimeString());
  $effect(() => {
    const interval = setInterval(() => { time = new Date().toLocaleTimeString(); }, 1000);
    return () => clearInterval(interval);
  });
</script>

<div class="StatusBar">
  <div class="left">
    <span class="conn-dot" style="background: {statusColors[connectionStatus]}"></span>
    <span>{connectionStatus === 'connected' ? '已连接' : connectionStatus === 'disconnected' ? '未连接' : '连接中'}</span>
    {#if currentFile}<span class="sep">|</span><span>📂 {currentFile}</span>{/if}
  </div>
  <div class="right">
    <span>{message}</span><span class="sep">|</span><span>🕐 {time}</span>
  </div>
</div>

<style>
  .StatusBar { display: flex; justify-content: space-between; align-items: center; padding: 4px 12px; background: #1a1a2e; border-top: 1px solid #333; font-size: 11px; color: #888; height: 28px; box-sizing: border-box; }
  .left, .right { display: flex; align-items: center; gap: 6px; }
  .conn-dot { width: 6px; height: 6px; border-radius: 50%; }
  .sep { color: #444; }
</style>
