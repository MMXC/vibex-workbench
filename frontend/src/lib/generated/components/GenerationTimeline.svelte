<script lang="ts">
  interface Step { id: number; name: string; status: 'pending' | 'running' | 'done' | 'error'; log: string; }
  interface Props { ongenerate?: () => Promise<void>; }
  let { ongenerate }: Props = $props();

  let status = $state<'idle' | 'running' | 'success' | 'error'>('idle');
  let steps = $state<Step[]>([
    { id: 0, name: '解析 spec 文件', status: 'pending', log: '' },
    { id: 1, name: '构建依赖图', status: 'pending', log: '' },
    { id: 2, name: '生成代码骨架', status: 'pending', log: '' },
    { id: 3, name: '写入文件', status: 'pending', log: '' },
    { id: 4, name: '验证产出', status: 'pending', log: '' },
  ]);
  let currentStep = $state(0);

  async function runGenerate() {
    status = 'running';
    for (let i = 0; i < steps.length; i++) {
      steps[i].status = 'running';
      currentStep = i;
      steps[i].log = '执行中...';
      await new Promise(r => setTimeout(r, 600));
      steps[i].status = 'done';
      steps[i].log = '完成';
    }
    try {
      await ongenerate?.();
      status = 'success';
    } catch {
      steps[steps.length - 1].status = 'error';
      status = 'error';
    }
  }
</script>

<div class="GenerationTimeline">
  <div class="header">
    <button onclick={runGenerate} disabled={status === 'running'} class="run-btn">
      {status === 'running' ? '🔄 生成中...' : status === 'success' ? '✅ 完成' : status === 'error' ? '❌ 失败' : '▶ 开始生成'}
    </button>
    {#if status === 'running'}
      <div class="progress-bar">
        <div class="progress-fill" style="width: {Math.round((currentStep / steps.length) * 100)}%"></div>
      </div>
      <span class="pct">{Math.round((currentStep / steps.length) * 100)}%</span>
    {/if}
  </div>

  <div class="steps">
    {#each steps as step}
      <div class="step" class:pending={step.status === 'pending'} class:running={step.status === 'running'} class:done={step.status === 'done'} class:error={step.status === 'error'}>
        <div class="step-icon">
          {#if step.status === 'done'}✅{:else if step.status === 'running'}⏳{:else if step.status === 'error'}❌{:else}⭕{/if}
        </div>
        <div class="step-body">
          <span class="step-name">{step.name}</span>
          {#if step.log}<span class="step-log">{step.log}</span>{/if}
        </div>
      </div>
    {/each}
  </div>
</div>

<style>
  .GenerationTimeline { display: flex; flex-direction: column; gap: 12px; padding: 16px; }
  .header { display: flex; align-items: center; gap: 12px; }
  .run-btn { background: #4D96FF; border: none; color: white; padding: 8px 20px; border-radius: 6px; cursor: pointer; font-size: 13px; font-weight: 500; }
  .run-btn:disabled { background: #333; color: #888; cursor: not-allowed; }
  .progress-bar { flex: 1; height: 4px; background: #333; border-radius: 2px; overflow: hidden; }
  .progress-fill { height: 100%; background: #4D96FF; transition: width 0.3s; }
  .pct { font-size: 11px; color: #888; min-width: 32px; }
  .steps { display: flex; flex-direction: column; gap: 4px; }
  .step { display: flex; align-items: flex-start; gap: 10px; padding: 8px 10px; border-radius: 6px; transition: background 0.15s; }
  .step.running { background: #1a2a4a; }
  .step.done { }
  .step.error { }
  .step-icon { font-size: 14px; flex-shrink: 0; padding-top: 1px; }
  .step-body { display: flex; flex-direction: column; gap: 2px; }
  .step-name { font-size: 13px; color: #e0e0e0; }
  .step.pending .step-name { color: #666; }
  .step.log { font-size: 11px; color: #888; }
  .step.running .step-log { color: #4D96FF; }
  .step.done .step-log { color: #6BCB77; }
  .step.error .step-log { color: #FF6B6B; }
</style>
