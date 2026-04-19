# Spec — E4: Artifact Registry

> PRD: `prd.md` E4 章节
> 规范层级: L4 Feature Spec

## Artifact 持久化规范

### IndexedDB Schema（via Dexie.js）

```typescript
// 表: artifacts
interface ArtifactRecord {
  id: string;
  thread_id?: string;
  run_id?: string;
  type: string;       // 'code' | 'image' | 'text'
  name: string;
  content: string;
  language?: string;
  mime_type: string;
  size_bytes: number;
  tags: string[];
  status: string;
  deleted_at: string | null;
  created_at: string;
  updated_at: string;
  last_accessed_at: string;
}
```

---

## UI 状态规范

### ArtifactPanel 组件四态

#### 1. 加载态
```
┌───────────────────┐
│ Artifacts (--)   │
│ [🔍 搜索...]      │
│ ┌───────────────┐ │
│ │ ████████ ███ │ │
│ │ ████████     │ │
│ └───────────────┘ │
└───────────────────┘
```
- 骨架屏：2 条占位，`height: 36px`，`border-radius: 6px`
- 搜索框骨架：`height: 32px`

#### 2. 空状态
```
┌───────────────────┐
│ Artifacts (0)     │
│ [🔍 搜索...]      │
│                   │
│    [上传图标]       │
│  还没有产物        │
│  AI 生成的内容会    │
│  显示在这里        │
│                   │
│  [上传文件]        │
└───────────────────┘
```
- 引导文案：3 行，具体说明 Artifact 来源
- 上传按钮：`background: #4f46e5`, `color: white`, `border-radius: 8px`
- 禁止只写"无内容"或留白

#### 3. 正常态
```
┌───────────────────┐
│ Artifacts (5)     │
│ [🔍 搜索...]      │
├───────────────────┤
│ [code] main.py    │
│ [image] logo.png  │
│ [text] output.md  │
└───────────────────┘
```
- 类型标签：`[code]` 紫色 `#7c3aed`, `[image]` 绿色 `#16a34a`, `[text]` 蓝色 `#2563eb`
- hover：`background: #1a1a1a`

#### 4. 错误态
```
┌───────────────────┐
│ Artifacts (--)   │
│ [🔍 搜索...]      │
│                   │
│  ⚠️ 加载失败       │
│  [重试]           │
└───────────────────┘
```
- 重试按钮：同 Thread 错误态

---

## Artifact 预览 Modal

### 触发条件
点击 Artifact Item

### Modal 布局
```
┌──────────────────────────────────────────┐
│  main.py                            [✕]  │
├──────────────────────────────────────────┤
│  language: python  │  size: 2.4 KB       │
│  created: 2026-04-20  │  tags: [py]     │
├──────────────────────────────────────────┤
│                                          │
│  # main.py                               │
│  def hello():                            │
│    print("world")                        │
│                                          │
├──────────────────────────────────────────┤
│  [@{artifactId}]  [复制内容]  [删除]      │
└──────────────────────────────────────────┘
```
- 宽度：600px，最大高度 80vh，滚动
- 代码高亮：使用 `highlight.js` 或 `Prism`
- 图片预览：`max-width: 100%`, `max-height: 60vh`
- 操作栏：`[@{artifactId}]` 引用按钮（点击复制）

---

## Artifact 拖拽规范

### 拖拽行为
1. `draggable="true"` on `ArtifactItem`
2. `dragstart` → 设置 `dataTransfer` 为 `@{artifactId}`
3. `Composer.svelte` 的 `<textarea>` 作为 drop zone
4. `drop` → `composerText += @{artifactId}`

### 拖拽态视觉反馈
- 源节点：`opacity: 0.5` + `cursor: grabbing`
- drop zone：边框变虚线 `#4f46e5`，背景 `rgba(79,70,229,0.1)`

---

## 验收标准

```typescript
// F4.1 — 持久化
await uploadFile('test.py');
expect(artifactCount).toBe(1);
await refreshPage();
expect(artifactCount).toBe(1);

// F4.2 — 预览
await click(artifactItem('main.py'));
expect(previewModal).toBeVisible();
expect(codeHighlight).toBeVisible();  // 代码高亮
expect(copyButton).toBeVisible();

// F4.3 — 引用注入
const composerBefore = composerText;
await drag(artifactItem('main.py')).to(composerInput);
expect(composerText).toContain('@{artifactId}');
```

### DoD
- [ ] Dexie.js `artifacts` 表已创建
- [ ] Artifact 上传 → IndexedDB 写入
- [ ] 刷新页面 → Artifact 列表恢复
- [ ] 点击 Artifact → 预览 Modal 弹出
- [ ] 代码类型使用语法高亮，图片类型使用 `<img>` 预览
- [ ] 拖拽 Artifact 到 Composer → 注入 `@{artifactId}`
- [ ] ArtifactPanel 四态定义完整
- [ ] 空状态有引导文案
- [ ] 所有间距使用 8 倍数，颜色使用 CSS Token
