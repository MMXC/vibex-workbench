<script lang="ts">
  import { routing_store } from "../stores";

  let input = "";
  let suggestions: Array<{spec: string; layer: string; reason: string}> = [];
  let analyzing = false;

  // Sync with global routing store
  $: routing_store.update(s => ({
    ...s,
    changeInput: input,
    recommendations: suggestions,
    isLoading: analyzing ? 1 : 0,
  }));

  const layer_keywords: Record<string, string[]> = {
    "L1-目标": ["目标", "愿景", "价值", "用户", "为什么"],
    "L2-骨架": ["架构", "技术选型", "框架", "模块划分", "设计"],
    "L3-组织": ["模块", "service", "组件", "包结构", "目录"],
    "L4-功能": ["功能", "feature", "按钮", "交互", "界面", "用户体验"],
    "L5-细节": ["实现", "代码", "bug", "修复", "样式", "css", "样式"],
  };

  const layerColors: Record<string, string> = {
    "L1-目标": "#FF6B6B",
    "L2-骨架": "#FFA94D",
    "L3-组织": "#FFD93D",
    "L4-功能": "#6BCB77",
    "L5-细节": "#4D96FF",
  };

  function analyze() {
    analyzing = true;
    suggestions = [];
    const lower = input.toLowerCase().trim();

    if (!lower) {
      analyzing = false;
      return;
    }

    for (const [layer, keywords] of Object.entries(layer_keywords)) {
      const matched = keywords.filter(k => lower.includes(k.toLowerCase()));
      if (matched.length > 0) {
        suggestions.push({
          spec: `${layer} 层 spec`,
          layer,
          reason: `关键词匹配: ${matched.join(", ")}`,
        });
      }
    }

    // spec-designer 界面建议
    if (lower.includes("界面") || lower.includes("ui") || lower.includes("原型") || lower.includes("按钮") || lower.includes("布局")) {
      suggestions.push({
        spec: "spec-designer 技能 → 绘制界面原型",
        layer: "L5-UIUX",
        reason: "涉及界面设计与原型",
      });
    }

    // 代码生成建议
    if (lower.includes("生成") || lower.includes("实现") || lower.includes("代码")) {
      suggestions.push({
        spec: "spec-to-sveltekit / spec-to-kratos generator",
        layer: "L3-生成",
        reason: "需要代码生成",
      });
    }

    // 默认兜底
    if (suggestions.length === 0) {
      suggestions.push({
        spec: "需要人工判断目标层级",
        layer: "?",
        reason: "无关键词匹配",
      });
    }

    analyzing = false;
  }
</script>

<div class="change-router">
  <div class="header">
    <span class="title">🔀 变更路由</span>
    <span class="hint">输入 bug 描述或 feature 需求 → 分析应修改哪个层级</span>
  </div>

  <textarea
    bind:value={input}
    placeholder="例如：修复登录按钮的样式问题&#10;例如：添加一个用户头像上传功能&#10;例如：需要支持 markdown 预览"
    rows={5}
    on:input={() => { if (input.length > 10) analyze(); }}
  />

  <button class="analyze-btn" on:click={analyze} disabled={analyzing || !input.trim()}>
    {analyzing ? "🔄 分析中..." : "🔍 分析从属"}
  </button>

  {#if suggestions.length > 0}
    <div class="results">
      {#each suggestions as s}
        <div class="suggestion" style="border-left-color: {layerColors[s.layer] || '#888'}">
          <span class="layer-badge" style="background: {layerColors[s.layer] || '#888'}">
            {s.layer}
          </span>
          <div class="suggestion-body">
            <strong>{s.spec}</strong>
            <p>{s.reason}</p>
          </div>
        </div>
      {/each}
    </div>
  {:else if input && input.length <= 10}
    <p class="hint-mini">继续输入，10字以上自动分析</p>
  {/if}
</div>

<style>
  .change-router {
    padding: 16px;
    display: flex;
    flex-direction: column;
    gap: 12px;
    height: 100%;
    box-sizing: border-box;
  }

  .header {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .title {
    font-weight: 600;
    font-size: 15px;
    color: #e0e0e0;
  }

  .hint {
    font-size: 12px;
    color: #666;
  }

  textarea {
    width: 100%;
    background: #1e1e2e;
    color: #e0e0e0;
    border: 1px solid #333;
    border-radius: 6px;
    padding: 10px;
    font-family: inherit;
    font-size: 14px;
    resize: vertical;
    box-sizing: border-box;
    transition: border-color 0.2s;
  }

  textarea:focus {
    outline: none;
    border-color: #4D96FF;
  }

  .analyze-btn {
    background: #4D96FF;
    color: white;
    border: none;
    border-radius: 6px;
    padding: 8px 16px;
    cursor: pointer;
    font-size: 14px;
    font-weight: 500;
    transition: background 0.2s;
    align-self: flex-start;
  }

  .analyze-btn:hover:not(:disabled) {
    background: #3a7bd5;
  }

  .analyze-btn:disabled {
    background: #333;
    color: #666;
    cursor: not-allowed;
  }

  .results {
    display: flex;
    flex-direction: column;
    gap: 8px;
    overflow-y: auto;
    flex: 1;
  }

  .suggestion {
    display: flex;
    gap: 10px;
    padding: 10px 12px;
    border-radius: 6px;
    background: #252540;
    border-left: 3px solid;
    align-items: flex-start;
  }

  .layer-badge {
    color: white;
    font-size: 11px;
    font-weight: 600;
    padding: 2px 6px;
    border-radius: 4px;
    white-space: nowrap;
    flex-shrink: 0;
    margin-top: 1px;
  }

  .suggestion-body {
    flex: 1;
  }

  .suggestion-body strong {
    color: #e0e0e0;
    font-size: 13px;
    display: block;
    margin-bottom: 4px;
  }

  .suggestion-body p {
    color: #888;
    font-size: 12px;
    margin: 0;
  }

  .hint-mini {
    color: #555;
    font-size: 12px;
    margin: 0;
  }
</style>
