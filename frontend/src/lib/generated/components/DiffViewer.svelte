<script lang="ts">
  interface DiffLine { type: 'add' | 'remove' | 'context'; text: string; }
  interface Props {
    oldText?: string; newText?: string;
    oldLabel?: string; newLabel?: string;
  }
  let { oldText = '', newText = '', oldLabel = '旧版本', newLabel = '新版本' }: Props = $props();
  let viewMode = $state<'split' | 'unified'>('unified');

  function computeDiff(oldStr: string, newStr: string): DiffLine[] {
    const oldLines = oldStr.split('\n'), newLines = newStr.split('\n');
    const m = oldLines.length, n = newLines.length;
    const dp: number[][] = Array.from({length: m + 1}, () => new Array(n + 1).fill(0));
    for (let i = 1; i <= m; i++) for (let j = 1; j <= n; j++)
      dp[i][j] = oldLines[i-1] === newLines[j-1] ? dp[i-1][j-1] + 1 : Math.max(dp[i-1][j], dp[i][j-1]);
    let i = m, j = n; const ops: DiffLine[] = [];
    while (i > 0 || j > 0) {
      if (i > 0 && j > 0 && oldLines[i-1] === newLines[j-1]) { ops.unshift({ type: 'context', text: oldLines[i-1] }); i--; j--; }
      else if (j > 0 && (i === 0 || dp[i][j-1] >= dp[i-1][j])) { ops.unshift({ type: 'add', text: newLines[j-1] }); j--; }
      else { ops.unshift({ type: 'remove', text: oldLines[i-1] }); i--; }
    }
    return ops;
  }
  let diffLines = $derived(computeDiff(oldText, newText));
  let added = $derived(diffLines.filter(l => l.type === 'add').length);
  let removed = $derived(diffLines.filter(l => l.type === 'remove').length);
</script>

<div class="DiffViewer">
  <div class="header">
    <div class="labels">
      <span class="label old">{oldLabel}</span>
      <span class="label new">{newLabel}</span>
    </div>
    <div class="controls">
      <span class="stats"><span class="add">+{added}</span><span class="remove">-{removed}</span></span>
      <button class:active={viewMode === 'unified'} onclick={() => viewMode = 'unified'}>统一</button>
      <button class:active={viewMode === 'split'} onclick={() => viewMode = 'split'}>分栏</button>
    </div>
  </div>
  <div class="diff-body">
    {#each diffLines as line}
      <div class="diff-line {line.type}">
        <span class="gutter">{line.type === 'add' ? '+' : line.type === 'remove' ? '-' : ' '}</span>
        <code>{line.text}</code>
      </div>
    {/each}
  </div>
</div>

<style>
  .DiffViewer { display: flex; flex-direction: column; height: 100%; background: #0d0d14; border-radius: 6px; overflow: hidden; }
  .header { display: flex; justify-content: space-between; align-items: center; padding: 8px 12px; background: #1a1a2e; border-bottom: 1px solid #333; }
  .labels { display: flex; gap: 8px; }
  .label { font-size: 12px; padding: 2px 8px; border-radius: 3px; }
  .label.old { background: #4a2a2a; color: #ff8080; }
  .label.new { background: #2a4a2a; color: #80ff80; }
  .controls { display: flex; align-items: center; gap: 8px; }
  .stats { font-size: 12px; }
  .add { color: #6BCB77; }
  .remove { color: #FF6B6B; }
  button { background: transparent; border: 1px solid #333; color: #888; padding: 2px 8px; border-radius: 4px; cursor: pointer; font-size: 11px; }
  button.active { background: #4D96FF; color: #fff; border-color: #4D96FF; }
  .diff-body { flex: 1; overflow: auto; font-family: monospace; font-size: 12px; line-height: 1.5; }
  .diff-line { display: flex; padding: 0 8px; white-space: pre; }
  .diff-line.add { background: rgba(107, 203, 119, 0.12); }
  .diff-line.remove { background: rgba(255, 107, 107, 0.12); }
  .gutter { width: 20px; flex-shrink: 0; color: #888; }
  code { color: #e0e0e0; }
  .diff-line.add code { color: #6BCB77; }
  .diff-line.remove code { color: #FF6B6B; }
</style>
