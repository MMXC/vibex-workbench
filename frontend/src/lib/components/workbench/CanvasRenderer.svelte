<!-- ============================================================
CanvasRenderer — @xyflow/svelte 包装层
E5-U1: Canvas 渲染层集成
E5-U2: dagre 自动布局
E5-U3: 节点交互（拖拽、展开）
E5-U4: SSE → canvasStore → 渲染层同步
============================================================ -->

<script lang="ts">
  import {
    SvelteFlow,
    Controls,
    Background,
    type Node,
    type Edge,
  } from '@xyflow/svelte';
  import '@xyflow/svelte/dist/style.css';
  import { canvasStore } from '$lib/stores/canvas-store';
  import {
    aiBlocksStore,
    type AiBlock,
    type CanvasSkillAction,
    type CanvasSkillPayload,
  } from '$lib/stores/ai-blocks-store';
  import { specAgentContextStore } from '$lib/stores/spec-agent-context-store';
  import { layoutNodes } from '$lib/canvas-layout';

  // Reactive state driven by canvasStore
  let storeNodes = $state<Node[]>([]);
  let storeEdges = $state<Edge[]>([]);
  let selectedNodeId = $state<string | null>(null);
  let detailNode = $state<Node | null>(null);
  let aiBlocks = $state<AiBlock[]>([]);
  let skillPayload = $state<CanvasSkillPayload | null>(null);
  let templateId = $state<'spec_bootstrap_flow' | 'legacy_align_flow'>('spec_bootstrap_flow');

  const templatePresets: Record<string, CanvasSkillPayload> = {
    spec_bootstrap_flow: {
      template: 'spec_bootstrap_flow',
      title: '新项目 L1-L5 初始化',
      summary: '澄清需求后初始化占位链，写盘前确认。',
      blocks: [
        { type: 'steps', title: '流程', items: ['workspace_detect_state', 'clarify', 'workspace_specs_bootstrap', 'verify_spec_suite'] },
        { type: 'kv', title: '参数', data: { workspace_root: '<workspace_root>', overwrite: false, confirm: true } },
      ],
      actions: [
        {
          id: 'run-bootstrap',
          label: '请求初始化',
          type: 'request_api',
          variant: 'primary',
          confirm: '将写入 specs/L1~L5 占位文件，是否继续？',
          payload: { endpoint: '/api/workspace/spec-bootstrap', method: 'POST', workspace_root: '<workspace_root>' },
        },
        { id: 'cancel', label: '取消', type: 'cancel', variant: 'secondary' },
      ],
    },
    legacy_align_flow: {
      template: 'legacy_align_flow',
      title: '旧项目 Spec 对齐',
      summary: '先治理检测，再按缺口增量补链，不默认覆盖。',
      blocks: [
        { type: 'steps', title: '流程', items: ['workspace_detect_state', 'governance_status', 'clarify_gap', 'verify_spec_suite'] },
        { type: 'kv', title: '参数', data: { workspace_root: '<workspace_root>', overwrite: false, mode: 'incremental' } },
      ],
      actions: [
        {
          id: 'run-align',
          label: '请求对齐计划',
          type: 'prefill_composer',
          variant: 'primary',
          payload: { command: '/workspace "按旧项目模式执行 detect + governance + 缺口分析，并给出增量补链计划。"' },
        },
        { id: 'cancel', label: '取消', type: 'cancel', variant: 'secondary' },
      ],
    },
  };

  $effect(() => {
    const unsub = canvasStore.subscribe(s => {
      storeNodes = s.nodes as unknown as Node[];
      storeEdges = s.edges as unknown as Edge[];
    });
    return unsub;
  });

  $effect(() => {
    const unsub = aiBlocksStore.subscribe(s => {
      aiBlocks = s.blocks;
      skillPayload = s.skillPayload;
    });
    return unsub;
  });

  // E5-U2: 自动布局触发 — 首次加载时对无坐标节点布局
  $effect(() => {
    if (storeNodes.length === 0) return;
    const needsLayout = storeNodes.filter(n => !n.position || (n.position.x === 0 && n.position.y === 0));
    if (needsLayout.length > 0) {
      // 提取 edges（从 canvasStore 或 SSE events 建立的 edge）
      const edges = storeEdges.map(e => ({ source: e.source, target: e.target }));
      const posMap = layoutNodes(
        storeNodes.map(n => ({ id: n.id!, type: n.type ?? 'default' })),
        edges
      );
      for (const node of storeNodes) {
        const pos = posMap[node.id!];
        if (pos) {
          canvasStore.updateNode(node.id!, { position: { x: pos.x, y: pos.y } } as any);
        }
      }
    }
  });

  function closeDetail() {
    detailNode = null;
    selectedNodeId = null;
  }

  function executeBlock(block: AiBlock) {
    if (!block.command) return;
    specAgentContextStore.prefillCommand(block.command);
  }

  function pinBlockToCanvas(block: AiBlock) {
    const nodeId = `aiblock:${block.id}`;
    const exists = storeNodes.some(n => n.id === nodeId);
    const patch = {
      type: 'default',
      data: {
        kind: 'ai-block',
        label: block.title,
        content: block.content,
        command: block.command ?? '',
      },
      position: {
        x: 120 + (storeNodes.length % 4) * 260,
        y: 120 + Math.floor(storeNodes.length / 4) * 170,
      },
    };
    if (exists) {
      canvasStore.updateNode(nodeId, patch as any);
      return;
    }
    canvasStore.addNode({
      id: nodeId,
      ...(patch as any),
    });
  }

  function applySkillAction(action: CanvasSkillAction) {
    if (action.confirm && !window.confirm(action.confirm)) return;
    if (action.type === 'cancel') return;
    if (action.type === 'prefill_composer') {
      const cmd = String(action.payload?.command ?? '').trim();
      if (cmd) specAgentContextStore.prefillCommand(cmd);
      return;
    }
    if (action.type === 'open_slot') {
      const prompt = String(action.payload?.prompt ?? '/open-slot "请打开当前 spec 槽位澄清"');
      specAgentContextStore.prefillCommand(prompt);
      return;
    }
    if (action.type === 'request_api') {
      const endpoint = String(action.payload?.endpoint ?? '/api/workspace/spec-bootstrap');
      const method = String(action.payload?.method ?? 'POST').toUpperCase();
      const root = String(action.payload?.workspace_root ?? '');
      const body = action.payload?.body && typeof action.payload.body === 'object'
        ? JSON.stringify(action.payload.body)
        : '{}';
      const cmd =
        `/workspace "请执行 API 请求：${method} ${endpoint}` +
        `${root ? `，workspace_root=${root}` : ''}，body=${body}。先校验参数再执行。"`;
      specAgentContextStore.prefillCommand(cmd);
      return;
    }
    if (action.type === 'pin_canvas') {
      const label = String(action.payload?.label ?? action.label);
      const content = String(action.payload?.content ?? '');
      const block: AiBlock = {
        id: `pin:${action.id}`,
        threadId: 'current',
        messageId: 'payload',
        kind: 'item',
        title: label,
        content,
      };
      pinBlockToCanvas(block);
    }
  }

  function insertTemplateToComposer() {
    const tpl = templatePresets[templateId];
    if (!tpl) return;
    const json = JSON.stringify(tpl, null, 2);
    specAgentContextStore.prefillCommand(
      `/analyse "请基于以下 CanvasSkillPayload 模板填充真实参数并返回 JSON（保留字段结构）"\n\`\`\`json\n${json}\n\`\`\``
    );
  }

  // E5-U3: 节点拖拽结束后保存位置（onnodedragstop → { targetNode, nodes, event }）
  function handleNodeDragStop({
    targetNode,
  }: {
    targetNode: Node | null;
    nodes?: Node[];
    event?: MouseEvent | TouchEvent;
  }) {
    if (!targetNode?.id || !targetNode.position) return;
    canvasStore.updateNode(targetNode.id, { position: targetNode.position } as any);
  }
</script>

<div class="canvas-renderer">
  <aside class="skill-template-picker" aria-label="Canvas Skill 模板选择器">
    <label for="skill-template-select">模板</label>
    <select id="skill-template-select" bind:value={templateId}>
      <option value="spec_bootstrap_flow">spec_bootstrap_flow</option>
      <option value="legacy_align_flow">legacy_align_flow</option>
    </select>
    <button type="button" class="primary" onclick={insertTemplateToComposer}>插入模板到 Composer</button>
  </aside>

  {#if skillPayload}
    <aside class="skill-panel" aria-label="Canvas Skill 模板卡片">
      <header class="skill-head">
        <span class="tpl">{skillPayload.template}</span>
        <strong>{skillPayload.title}</strong>
        {#if skillPayload.summary}
          <p>{skillPayload.summary}</p>
        {/if}
      </header>
      {#if skillPayload.blocks && skillPayload.blocks.length > 0}
        <div class="skill-blocks">
          {#each skillPayload.blocks as block, i (`${block.type}-${i}`)}
            <article class="skill-block">
              <div class="line">
                <span class="k">{block.type}</span>
                {#if block.title}<strong>{block.title}</strong>{/if}
              </div>
              {#if block.items && block.items.length > 0}
                <ul>{#each block.items as it}<li>{it}</li>{/each}</ul>
              {:else if block.data}
                <pre>{JSON.stringify(block.data, null, 2)}</pre>
              {:else if block.content}
                <pre>{block.content}</pre>
              {/if}
            </article>
          {/each}
        </div>
      {/if}
      {#if skillPayload.actions && skillPayload.actions.length > 0}
        <div class="skill-actions">
          {#each skillPayload.actions as action (action.id)}
            <button
              type="button"
              class:danger={action.variant === 'danger'}
              class:primary={action.variant === 'primary'}
              onclick={() => applySkillAction(action)}
            >
              {action.label}
            </button>
          {/each}
        </div>
      {/if}
    </aside>
  {/if}

  <aside class="ai-block-hud" aria-label="AI 回复提取块">
    <header>
      <strong>AI 提取块</strong>
      <small>{aiBlocks.length} blocks</small>
    </header>
    {#if aiBlocks.length === 0}
      <p class="hud-empty">等待 assistant 回复中的命令 / 代码块 / JSON / 要点…</p>
    {:else}
      <div class="hud-list">
        {#each aiBlocks as block (block.id)}
          <article class="hud-card">
            <div class="hud-top">
              <span class="k">{block.kind}</span>
              <strong>{block.title}</strong>
            </div>
            <pre>{block.content.slice(0, 220)}</pre>
            <div class="hud-actions">
              {#if block.command}
                <button type="button" onclick={() => executeBlock(block)}>填入 Composer</button>
              {/if}
              <button type="button" onclick={() => pinBlockToCanvas(block)}>加入画布</button>
            </div>
          </article>
        {/each}
      </div>
    {/if}
  </aside>

  <SvelteFlow
    nodes={storeNodes}
    edges={storeEdges}
    fitView
    onnodeclick={({ node }) => {
      if (!node?.id) return;
      selectedNodeId = node.id;
      detailNode = node;
    }}
    onnodedragstop={handleNodeDragStop}
  >
    <Controls />
    <Background />
  </SvelteFlow>

  <!-- E5-U3: 节点详情面板 -->
  {#if detailNode}
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="detail-overlay" onclick={closeDetail}>
      <div class="detail-panel" onclick={(e) => e.stopPropagation()}>
        <div class="detail-header">
          <span class="detail-type"
            >[{detailNode.data?.kind ?? detailNode.type ?? 'node'}]</span
          >
          <span class="detail-label">{detailNode.data?.label ?? detailNode.id}</span>
          <button onclick={closeDetail}>×</button>
        </div>
        <div class="detail-body">
          {#if detailNode.data?.args}
            <div class="detail-section">
              <span class="detail-key">args:</span>
              <pre class="detail-code">{JSON.stringify(detailNode.data.args, null, 2)}</pre>
            </div>
          {/if}
          {#if detailNode.data?.result}
            <div class="detail-section">
              <span class="detail-key">result:</span>
              <pre class="detail-code">{JSON.stringify(detailNode.data.result, null, 2)}</pre>
            </div>
          {/if}
          {#if detailNode.data?.error}
            <div class="detail-section">
              <span class="detail-key error">error:</span>
              <pre class="detail-code error">{detailNode.data.error}</pre>
            </div>
          {/if}
          {#if detailNode.data?.status}
            <div class="detail-section">
              <span class="detail-key">status:</span>
              <span class="detail-status" class:running={detailNode.data.status === 'running'} class:completed={detailNode.data.status === 'completed'} class:failed={detailNode.data.status === 'failed'}>{detailNode.data.status}</span>
            </div>
          {/if}
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
  .canvas-renderer { width: 100%; height: 100%; position: relative; }
  :global(.svelte-flow) { background: var(--wb-bg-base, #0b0c10); }
  :global(.svelte-flow .node) { border-radius: 6px; }

  .skill-template-picker {
    position: absolute;
    left: 12px;
    bottom: 12px;
    z-index: 12;
    display: flex;
    align-items: center;
    gap: 8px;
    border: 1px solid #2f425f;
    border-radius: 10px;
    background: rgba(14, 21, 33, 0.95);
    padding: 8px;
  }
  .skill-template-picker label { color: #a3abb9; font-size: 11px; }
  .skill-template-picker select {
    border: 1px solid #3e557a;
    background: #111b2c;
    color: #e6edff;
    border-radius: 8px;
    padding: 4px 8px;
    font-size: 12px;
  }
  .skill-template-picker button {
    border: 1px solid #7aa2ff;
    border-radius: 8px;
    background: #1a2a44;
    color: #e6edff;
    font-size: 12px;
    padding: 5px 8px;
    cursor: pointer;
  }

  .skill-panel {
    position: absolute;
    left: 12px;
    top: 12px;
    z-index: 12;
    width: min(520px, 54%);
    max-height: calc(100% - 24px);
    overflow: auto;
    border: 1px solid #33507a;
    border-radius: 12px;
    background: rgba(13, 20, 31, 0.95);
    padding: 10px;
  }
  .skill-head .tpl {
    display: inline-block;
    font-size: 10px;
    color: #9ec1ff;
    border: 1px solid #3d5a88;
    border-radius: 999px;
    padding: 2px 6px;
    margin-bottom: 6px;
  }
  .skill-head strong { display: block; color: #eef0f5; font-size: 14px; }
  .skill-head p { margin: 6px 0 0; color: #a3abb9; font-size: 12px; }
  .skill-blocks { display: grid; gap: 8px; margin-top: 10px; }
  .skill-block { border: 1px solid #2f425f; border-radius: 10px; padding: 8px; background: #121b29; }
  .skill-block .line { display: flex; gap: 8px; align-items: center; margin-bottom: 6px; }
  .skill-block .k {
    font-size: 10px;
    color: #9ec1ff;
    background: rgba(122, 162, 255, .15);
    border-radius: 999px;
    padding: 2px 6px;
  }
  .skill-block strong { color: #eaf0ff; font-size: 12px; }
  .skill-block ul { margin: 0; padding-left: 18px; color: #d3ddef; font-size: 12px; }
  .skill-block pre {
    margin: 0;
    font-size: 11px;
    max-height: 160px;
    overflow: auto;
    color: #cfdbef;
    white-space: pre-wrap;
    word-break: break-word;
  }
  .skill-actions { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 10px; }
  .skill-actions button {
    border: 1px solid #395072;
    border-radius: 8px;
    background: #162238;
    color: #e6edff;
    font-size: 12px;
    padding: 6px 10px;
    cursor: pointer;
  }
  .skill-actions button.primary { border-color: #7aa2ff; }
  .skill-actions button.danger { border-color: #e16d75; }

  .ai-block-hud {
    position: absolute;
    top: 12px;
    right: 12px;
    z-index: 12;
    width: min(380px, 42%);
    max-height: calc(100% - 24px);
    overflow: auto;
    border: 1px solid #303746;
    border-radius: 12px;
    background: rgba(16, 19, 26, 0.92);
    backdrop-filter: blur(2px);
    padding: 10px;
  }

  .ai-block-hud header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 8px;
  }

  .ai-block-hud strong { color: #eef0f5; font-size: 12px; }
  .ai-block-hud small { color: #a3abb9; font-size: 11px; }
  .hud-empty { margin: 0; color: #7d8698; font-size: 11px; line-height: 1.5; }
  .hud-list { display: grid; gap: 8px; }
  .hud-card { border: 1px solid #303746; border-radius: 10px; background: #11151d; padding: 8px; }
  .hud-top { display: flex; align-items: center; gap: 8px; margin-bottom: 6px; }
  .hud-top .k {
    font-size: 10px;
    padding: 2px 6px;
    border-radius: 999px;
    background: rgba(122, 162, 255, 0.2);
    color: #9ec1ff;
  }
  .hud-top strong { color: #e8ecff; font-size: 11px; }
  .hud-card pre {
    margin: 0;
    white-space: pre-wrap;
    word-break: break-word;
    max-height: 120px;
    overflow: auto;
    font-size: 11px;
    color: #cbd4e2;
  }
  .hud-actions { display: flex; gap: 6px; margin-top: 6px; }
  .hud-actions button {
    border: 1px solid #3d4a61;
    background: #141b26;
    color: #e6ecfb;
    border-radius: 8px;
    padding: 4px 8px;
    font-size: 11px;
    cursor: pointer;
  }
  .hud-actions button:hover { border-color: #7aa2ff; background: rgba(122, 162, 255, 0.16); }

  /* 详情面板 */
  .detail-overlay {
    position: absolute;
    inset: 0;
    background: rgba(0,0,0,0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 100;
  }
  .detail-panel {
    background: var(--wb-bg-panel, #151820);
    border: 1px solid var(--wb-border, #303746);
    border-radius: 10px;
    width: 480px;
    max-height: 70vh;
    overflow: auto;
  }
  .detail-header {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 12px 14px;
    border-bottom: 1px solid var(--wb-border, #303746);
  }
  .detail-type { color: var(--wb-brand, #7aa2ff); font-size: 12px; }
  .detail-label { color: var(--wb-text, #eef0f5); font-size: 13px; flex: 1; font-weight: 500; }
  .detail-header button { background: none; border: none; color: var(--wb-muted, #6f7888); font-size: 18px; cursor: pointer; }
  .detail-header button:hover { color: var(--wb-text, #eef0f5); }
  .detail-body { padding: 14px; display: flex; flex-direction: column; gap: 10px; }
  .detail-section { display: flex; flex-direction: column; gap: 4px; }
  .detail-key { color: var(--wb-muted, #6f7888); font-size: 11px; text-transform: uppercase; letter-spacing: 0.05em; }
  .detail-key.error { color: var(--accent-red, #e16d75); }
  .detail-code {
    background: #0b0d12;
    border: 1px solid var(--wb-border, #303746);
    border-radius: 6px;
    padding: 8px;
    font-size: 11px;
    color: var(--wb-text-sec, #a3abb9);
    white-space: pre-wrap;
    word-break: break-all;
    max-height: 200px;
    overflow: auto;
    margin: 0;
  }
  .detail-code.error { border-color: var(--accent-red, #e16d75); color: var(--accent-red, #e16d75); }
  .detail-status { font-size: 12px; padding: 2px 8px; border-radius: 10px; }
  .detail-status.running { background: #1e3a5f; color: #60a5fa; }
  .detail-status.completed { background: #1a3a2a; color: #4ade80; }
  .detail-status.failed { background: #3a1a1a; color: #f87171; }
</style>