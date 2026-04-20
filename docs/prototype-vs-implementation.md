# Spec-Prototype-Implementation 对齐方案

> **核心理念**：spec-designer 产出原型（可点击 HTML）→ generator/手写产出实现 → 结构级对比作为门禁。扩展了 validate 从「YAML 树」到「界面形态」的验收维度。

---

## 为什么可行

- **原型页** = 某一时刻对 goal/feature spec 的「可点击、可截图」承诺
- **生成物** = gen.py / 手写前端在同一 spec 约束下的实现
- **对比** = 把「规格里答应长什么样」变成可失败的门禁——与 drift/validate 同哲学

---

## 必须先钉死的 3 件事

### 1. 对比粒度

| 层级 | 成本 | 稳定性 | 适合阶段 |
|------|------|--------|----------|
| 整页像素级 | 高 | 脆（字体、viewport、动画） | 不推荐 |
| **结构/关键区块** | 低 | 稳（导航、三栏、主CTA、列表是否存在） | **先做这个** |
| 视觉快照（固定 viewport） | 中 | 中 | 第二阶段 |

**结论**：先结构 + 少量视觉快照，再谈像素。

### 2. 原型谁维护、版本谁为准

两种路径二选一：

**路径 A：原型为权威**
- 原型更新 → spec / changelog / result[] 同步更新
- 生成物必须对齐原型
- 风险：原型比实现「好看」，像素对齐会逼死实现

**路径 B：Spec 文本为权威（推荐）**
- 原型只是某一版「插图」，以 spec 变更为准更新基线
- 原型路径写入 `result[]` + changelog
- 对比维度：「关键布局与信息架构一致」而非全盘视觉克隆
- 改实现或改原型都要在 changelog 留痕

**结论**：路径 B。原型的定位是「spec 的视觉承诺」，不是独立的权威来源。

### 3. 自动化 vs 人工

| 层级 | 启动成本 | 持续成本 | 适合阶段 |
|------|----------|----------|----------|
| 人工：并排 + 清单勾选 | 零 | 低 | **最小可行** |
| 半自动：Playwright 截图 vs baseline | 中 | 中 | 第二阶段 |
| 全自动：截图 diff（Percy/Chromatic） | 高 | 高 | 稳定期 |

**结论**：从人工清单启动，成熟后逐步自动化。

---

## 与现有仓库的接口

### spec-designer 产出
- `prototypes/<feature-id>.html`（单文件，可点击）
- `prototypes/` 目录作为原型存储根

### spec result[] 对齐
```yaml
# specs/feature/<feature>/service.yaml
content:
  result:
    - id: r1
      description: 原型验收截图
      type: artifact
      path: prototypes/<feature-id>.html  # 原型路径钉死
      verification:
        checklist: prototypes/<feature-id>.checklist.md  # 人工验收清单
        status: pending  # pending / verified / mismatch
```

### generator 产出
- Svelte 组件 → 对比同一 feature 维度下的「原型 HTML」与「实现路由/组件」

### 不一致 → 迭代决策树

```
不一致出现
  ├── 行为/边界 spec 写得模糊 → 改 spec（清晰化边界）
  ├── generator 模板问题 → 改生成器模板
  └── 手写层实现漏项 → 补实现
```

---

## 最小可行一步（本期可落地）

每个 L4 feature 在 `result[]` 里固定：

1. **原型路径**：`prototypes/<feature-id>.html`
2. **验收清单**：`prototypes/<feature-id>.checklist.md`

`checklist.md` 格式：
```markdown
# <feature-id> 验收清单

## 关键结构（必须出现）
- [ ] 侧边栏存在且宽度 280px
- [ ] 主内容区为 `.canvas-area`
- [ ] 右栏宽度 320px

## 关键交互
- [ ] 点击 "+ 新建" 出现新线程
- [ ] 发送消息触发 SSE 连接

## 不依赖项（可跳过）
- [ ] 动画细节
- [ ] 字体渲染
- [ ] 精确像素值
```

心跳只问：**本期是否逐项勾完**，不跑截图 diff。

---

## 风险

1. **原型比实现好看**：模型生成的 HTML 往往比工程约束下的 UI 更炫 → 以「结构 + 信息架构」为对齐维度，而非像素克隆
2. **双轨维护**：原型不更新会误伤 CI → 原型路径写进 spec result[] + changelog，改任何一端都留痕
3. **无休止迭代**：没有基线锁定 → 每期 changelog 记录版本号，基线变更需要显式 commit
