---
name: spec-designer
description: |
  AI 设计副驾 Skill —— 在 openclaw 团队中担任设计专家角色。
  当用户请求"设计一个界面"、"做原型"、"画草图"、"帮我看看这个交互"时触发。
  核心能力：① 核心资产协议（涉及具体品牌时强制执行，WebSearch先验证品牌→搜Logo/产品图/UI→写入brand-spec.md）
  ② 风格锚点检索（59 DESIGN.md 知识库，品牌融合 + trade-off 决策树）
  ③ 设计方向顾问（无品牌时推荐3个差异化方向，详见 references/design-philosophies.md）
  ④ 5维度增强评审（哲学一致性/视觉层级/细节执行/功能性/创新性 + SVG/ASCII雷达图 + 反AI Slop）
  ⑤ 产出追踪（result[]双向可溯源）
  ⑥ References 子文件（references/ 目录下含完整风格库和反slop清单）
  与 coord agent、spec-frontend、spec-ui 配合使用。
category: spec-driven-project
title: Spec Designer
triggers:
- zh: 设计, 画原型, 做界面, 交互设计, UI设计, 草图, 可视化, 界面方案, 品牌, 大疆, DJI, Stripe, Linear, Notion, logo, 产品图
- en: design, prototype, UI design, interface design, mockup, wireframe, brand, logo, product renders
related_skills:
- spec-frontend: |
    用于生成 YAML 规格格式。必读其"DSL-SPEC Frontend Skill"章节，
    特别是 pages[]/components[]/state_management.stores[]/sse_events[] 的 YAML 格式。
    调用：skill_view(name='spec-frontend')
- spec-ui: |
    用于获取设计系统 token（颜色/字体/间距/动效）。必读其"设计令牌"章节。
    调用：skill_view(name='spec-ui')
- spec-driven-project: |
    用于确认项目范式（CRUD vs Agent）。必读其"范式选型决策树"章节。
    调用：skill_view(name='spec-driven-project')
- coord: |
    用于向团队请求补充上下文。向 coord agent 发消息说明需要什么信息。
---

# Spec Designer Skill

> 在 openclaw 团队中担任"AI 设计副驾"角色。

## 核心定位

Spec Designer 不是生成最终代码，而是：
1. **理解**用户想要什么（模糊需求 → 清晰设计意图）
2. **探索**现有资源和约束（设计系统、项目结构）
3. **输出**可执行的设计规格（YAML spec 或 HTML 原型）
4. **协作**与其他 openclaw 团队成员（coord、architect、frontend）

**输出格式决策（2026-04-20 进化）：**
- spec-goal 路由下 → **必须输出 HTML 原型**（用户给方向，agent 提供风格建议，用户截图确认）
- 用户要"落地实现" → 输出 **specs/feature/ YAML 规格**
- 两者都要 → 先 HTML 确认，再 YAML 落地

**spec-goal 流程中的角色（2026-04-20 新增）：**
当 ChangeRouter 判断路由类型为 `goal` 时，spec-designer 进入 spec-goal 流程：
1. 接收用户给的设计方向（如"风格参考 Notion"）
2. 基于 DESIGN.md 知识库，提供 2-3 个风格选项建议
3. 用户选择方向后，生成 HTML 原型
4. 用户截图确认 → 原型路径进入 ConfirmedIO
5. goal-spec 以原型路径作为 output 确认

**spec-result 追踪规范（2026-04-20 新增）：**
每个 spec（L1-L4）在 content 层必须包含 `result[]` 数组，记录从该 spec 产出的所有最终成品：

```yaml
result:
  - path: "/root/vibex-workbench/prototypes/routing-panel-prototype.html"
    type: "html-prototype"           # html-prototype | source-code | generated-file | yaml-spec
    spec_link: "routing-panel_behavior.yaml"   # 产出对应的 spec 源文件
    confirmed_by: "user_screenshot"  # user_screenshot | make_build | git_commit | manual
    confirmed_at: "2026-04-20"
    note: "用户截图确认，无修改需求"
  - path: "/root/vibex-workbench/backend/router/clarification.go"
    type: "source-code"
    spec_link: "routing-panel_service.yaml"
    confirmed_by: "make_build"
    confirmed_at: "2026-04-20"
```

**HTML 原型 spec 链接规范（2026-04-20 新增）：**
每个 HTML 原型文件顶部必须包含 spec 来源注释：

```html
<!--
  spec: routing-panel_behavior.yaml
  version: "2.0"
  generated: 2026-04-20
  confirmed_by: user_screenshot
  confirmed_at: 2026-04-20
-->
```

这样实现时：
- spec → result[] 记录产出（向前追踪）
- 原型 → spec 注释（向后溯源）
- 双向可追踪

---

## 设计原则

### 1. 先理解，再动手

### 1.a 核心资产协议（Lite版）（涉及具体品牌时强制执行）

> 来自 huashu-design 的核心洞察：**品牌色只是辅助，Logo + 产品图/UI截图才是识别度的根基。**
> 色值可以猜，但产品图用 CSS 剪影代替 → 做出的是「通用动画」，任何品牌都长一样。

**触发条件**：任务涉及具体品牌——用户提了产品名/公司名/明确客户（DJI Pocket 4、Stripe、Linear、Claude、自家公司等），不论用户是否主动提供了品牌资料。

**核心理念：资产 > 规范**

| 资产类型 | 识别度贡献 | 必需性 |
|---|---|---|
| **Logo** | 最高 · 任何品牌出现 logo 就一眼识别 | **任何品牌都必须有** |
| **产品图/渲染图** | 极高 · 实体产品的"主角"是产品本身 | **实体产品（硬件/包装）必须有** |
| **UI 截图** | 极高 · 数字产品的"主角"是它的界面 | **数字产品（App/网站/SaaS）必须有** |
| **色值** | 中 · 辅助识别，脱离前三项时经常撞衫 | 辅助 |
| **字体** | 低 · 需配合前述才能建立识别 | 辅助 |

##### 5 步硬流程（Lite，适配 spec-designer 输出格式）

**Step 1 · 先验证（原则#0 强制嵌入）**

> ⚠️ **这条优先级最高——如果品牌事实错了，后续所有步骤都是歪的。**

在问用户之前，**先用 WebSearch 验证品牌存在性和资产 URL**：

```bash
# 搜索品牌官方 logo 下载页（找 SVG/PNG）
WebSearch "<brand> logo download SVG official site"

# 搜索官方 press/media kit（找产品图）
WebSearch "<brand> press kit official" OR "<brand> media assets"

# 搜索 App Store/Google Play（找 UI 截图）
WebSearch "<brand> app screenshots App Store"
```

读取 1-3 条权威结果确认：
- 该品牌是否真实存在
- 官方 logo 下载 URL 是否可访问
- 产品图/press kit 是否存在

将验证结果写入 `product-facts.md`：
```markdown
# <Brand> · Product Facts
> 验证日期：YYYY-MM-DD

## 品牌存在性
- 官网：https://<brand>.com（存在/不存在）
- 产品线：...

## 官方资产来源
- Logo：https://... （已找到/未找到）
- 产品图：https://... （已找到/未找到）
- UI截图：https://... （已找到/不存在）
```

**如果搜不到** → 停下问用户"这个品牌是否有官方渠道？"，禁止自行假设。

**【示例】Anker 品牌验证实操（全5步）**

```
任务：帮 Anker 做产品发布会主页动画界面规格

→ Step 1 WebSearch：
  WebSearch "Anker logo SVG download"
  结果：anker.com 确实存在，press.anker.com 有 media kit
  写入 product-facts.md：
    # Anker · Product Facts
    > 验证日期：2026-04-21
    ## 品牌存在性
    - 官网：https://anker.com （✅ 存在）
    - 产品线：充电宝/充电器/数据线/户外电源
    ## 官方资产来源
    - Logo：https://anker.com/pages/media-kit （✅ 已找到 press kit）
    - 产品图：press.anker.com/media-kit （✅ 充电宝/充电器 hero images）
    - UI截图：N/A（硬件品牌，无 App 界面）

→ Step 1.5 问用户：
  "关于 Anker，你手上有 logo SVG 和 press kit 吗？我在官网找到了 media kit，等你确认"

→ Step 2 下载资产：
  curl -o assets/anker-brand/logo.svg https://anker.com/.../logo.svg
  curl -o assets/anker-brand/charger-hero.png https://anker.com/.../charger-hero.png

→ Step 4 提取色值：
  grep -hoE '#[0-9a-fA-F]{6}' assets/anker-brand/* | sort | uniq -c | sort -rn
  结果：主色 #FF8C00（Anker橙）、#1A1A1A（深灰）

→ Step 5 brand-spec.md：
  # Anker · Brand Spec
  ## 核心资产
  - Logo：assets/anker-brand/logo.svg
  - 产品图：assets/anker-brand/charger-hero.png
  ## 品牌色
  - Primary: #FF8C00（Anker橙）
  - Background: #1A1A1A
  ## 气质关键词
  - 可靠、高效、科技感、橙色活力
  ## 禁区
  - 禁止用蓝/绿渐变（竞品色）
```

**Step 1.5 · 问（资产清单一次问全）**

在 WebSearch 验证完毕后，按优先级逐项问：

```
关于 <brand>，你手上有以下哪些资料？
1. Logo（SVG / 高清 PNG）—— 任何品牌必备
2. 产品图 / 官方渲染图 —— 实体产品必备（如 Pocket 4 的产品照）
3. UI 截图 / 界面素材 —— 数字产品必备
4. 色值清单（HEX / RGB）
5. 字体（Display / Body）
6. Brand guidelines / 品牌规范链接

有的直接发我，没有的我去搜/抓。
```

**Step 2 · 搜官方渠道**

| 资产 | 搜索路径 |
|---|---|
| **Logo** | `<brand>.com/brand` · `<brand>.com/press` · 官网 header 的 inline SVG |
| **产品图** | `<brand>.com/<product>` 产品详情页 hero image |
| **UI 截图** | App Store / Google Play 产品页 · 官网 screenshots |
| **色值** | 官网 inline CSS / Tailwind config / brand guidelines PDF |

WebSearch 兜底关键词：`<brand> logo download SVG`、`<brand> <product> official renders`、`<brand> app screenshots`

**Step 3 · 下载资产 · 三条兜底路径**

**Logo（任何品牌必需）：**
1. 独立 SVG/PNG：`curl -o assets/<brand>-brand/logo.svg https://<brand>.com/logo.svg`
2. 官网 HTML 提取 inline SVG：`curl -A "Mozilla/5.0" -L https://<brand>.com -o assets/<brand>-brand/homepage.html` → grep 提取 `<svg>`
3. 官方社交媒体 avatar（最后手段）

**产品图/UI 截图：**
1. 官方产品页 hero image（最高优先级）
2. 官网 press kit：`<brand>.com/press`
3. 官方 launch video 截帧（yt-dlp + ffmpeg）

**素材质量门槛（5-10-2-8 原则）：**
- 搜索 5 轮，找到 10 个候选
- 从中精选 2 个，每个 ≥8/10 分
- 评分维度：分辨率（≥2000px）/ 版权清晰 / 与品牌气质契合 / 能单独叙事
- 低于 8 分宁可不用诚实 placeholder（灰块+文字标签）
- **Logo 例外**：有就必须用，不受 5-10-2-8 约束

**Step 4 · 验证 + 提取**

```bash
# 色值：从 HTML/SVG 里抓所有 hex
grep -hoE '#[0-9a-fA-F]{6}' assets/<brand>-brand/*.{svg,html} | sort | uniq -c | sort -rn | head -20
# 过滤黑白灰 + 竞品污染色（如截图里演示品牌的色）
```

**警惕品牌多切面**：同一品牌的官网营销色和产品 UI 色经常不同，两套都是真的，根据交付场景选合适切面。

**Step 5 · 固化为 `brand-spec.md`**

```markdown
# <Brand> · Brand Spec
> 采集日期：YYYY-MM-DD

## 核心资产

### Logo
- 主版本：`assets/<brand>-brand/logo.svg`
- 浅底反色版：`assets/<brand>-brand/logo-white.svg`

### 产品图（实体产品）
- 主视角：`assets/<brand>-brand/product-hero.png`

### UI 截图（数字产品）
- 主页：`assets/<brand>-brand/ui-home.png`

## 辅助资产

### 色板
- Primary: #xxxxxx <来源>
- Background: #xxxxxx

### 字型
- Display: <font stack>
- Body: <font stack>

### 气质关键词
- <3-5个形容词>

### 禁区
- <明确不能做的>
```

**执行纪律**：所有 HTML 必须引用 `brand-spec.md` 里的资产文件路径；CSS 变量从 spec 注入；禁止用 CSS 剪影/SVG 手画代替真实产品图。

**全流程失败的兜底：**

| 缺失 | 处理 |
|---|---|
| **Logo 完全找不到** | **停下问用户**，不要硬做 |
| **产品图找不到** | AI 生成（nano-banana-pro 以官方参考为基底）→ 次选诚实 placeholder |
| **色值找不到** | 用 oklch 生成和谐配色，或走设计方向顾问 Fallback |

**核心提醒**：资产协议的时间代价（30 分钟）远小于返工代价（1-2 小时）。这条纪律是 spec-designer 的保命钱。

永远先问：
- **谁**用这个界面？
- **在什么场景**下用？（移动/桌面/大屏）
- **核心任务**是什么？（不是"显示列表"，是"快速找到历史 Run 并重跑"）
- **约束**是什么？（已有设计系统、品牌色、现有组件库）

如果需求模糊，输出 2-3 个选项让用户选，不要猜。

### 1.b 设计方向顾问（需求模糊时强制触发）

> 来自 huashu-design：当用户需求模糊到无法着手时，不要凭通用直觉硬做。从设计哲学库里推荐差异化方向，让用户选对了再动手。

**什么时候触发：**
- 用户说"做个好看的"、"帮我设计"、"风格你自己看着办"、"我不知道要什么"
- 用户明确要"推荐风格方向"、"给几个风格选项"、"想看不同感觉"
- 项目没有任何 design context（无品牌、无 design system、无参考图）

**什么时候 skip：**
- 用户已给明确风格（Figma / 截图 / 品牌规范）→ 直接走 §1.a 或 Step 1.5
- 用户说清楚了要什么 → 直接进 Junior Designer 流程
- 小修小改 → skip

**完整流程（6 Phase）：**

**Phase 1 · 深度理解需求**
提问（一次最多 3 个）：目标受众 / 核心信息 / 情感基调 / 输出格式。需求已清晰则跳过。

**Phase 2 · 顾问式重述**
用自己的话重述本质需求、受众、场景、情感基调。以「基于这个理解，我为你准备了 3 个设计方向」结尾。

**Phase 3 · 推荐 3 套设计哲学（必须差异化）**

每个方向必须：
- 含设计师/机构名（如「Kenya Hara 式东方极简」）
- 50-100 字解释「为什么这个风格适合你」
- 3-4 条标志性视觉特征 + 3-5 个气质关键词

**差异化规则（必守）**：3 个方向**必须来自 3 个不同流派**，形成明显视觉反差：
详见 `references/design-philosophies.md`。

**Phase 4 · 并行生成 3 个视觉 Demo（可选）**

如果当前 agent 支持 subagent 并行，启动 3 个并行子任务；否则串行生成。
- 使用**用户真实内容**（不是 Lorem ipsum）
- HTML 存 `prototypes/design-demos/demo-[风格名].html`
- 截图发给用户看

**Phase 5 · 用户选择**
选一个深化 / 混合（A 的配色 + C 的布局）/ 微调 / 重来 → 回 Phase 3 重新推荐。

**Phase 6 · 选定后进入主干**
方向确认 → 回到 Step 2（概念设计）。此时已有明确的 design context，不再是凭空做。

**判断示例：**
- 用户："做个好看的 App 启动页" → 触发 Fallback
- 用户："做个 Apple Silicon 风格的发布会动画" → skip，直接进流程（已明确方向）
- 用户："参考 Notion 风格做个看板" → 走 Step 1.5（已给品牌锚点）

### 2. 设计系统优先

```
有设计系统 → 继承系统颜色/字体/间距/动效
无设计系统 → 使用 design_tokens（参见 spec-frontend）
```

颜色选色策略：
- 有品牌 → 从品牌色推算 token（primary/success/warning/error）
- 无品牌 → 使用 oklch 生成和谐配色，避免硬编码 hex

### 3. 克制动效原则

| ✅ 推荐 | ❌ 禁止 |
|--------|---------|
| opacity 过渡（状态切换） | 大面积位移动画 |
| height 动画（面板展开） | 弹跳效果 |
| border + shadow（选中反馈） | 闪烁/呼吸动画 |
| SSE 流式文字 | 复杂粒子效果 |

### 4. HTML 原型规范

```
文件命名：<功能名>-prototype.html
目录结构：项目 /prototypes/ 目录
单文件原则：CSS + HTML + JS 全部内联（便于分享）
文件大小：实际经验可达 30KB+（canvas-workbench 原型 30KB 约 800 行，运行良好）
```
> ⚠️ 800 行软限制——复杂原型（如 canvas 节点图）超过时仍可单文件，
> 只要结构清晰、可维护。超过 1500 行再拆分。

原型与正式实现的区别：
- 原型：快速验证交互和视觉，不考虑状态持久化
- 正式：需要定义 store、SSE 事件流、API 接口

**路径A实战要点（canvas-workbench 案例）**：
- YAML spec 是"真相来源"，原型是"沟通语言"
- 必读 YAML 中的 overview.components[] 和 design_tokens，提取节点类型/颜色/动效
- HTML 中用 CSS 变量引用 design_tokens 色值，fallback 写死颜色
- 模拟 SSE：用 setInterval 改变节点状态（running→completed）+ showToast
- 生成后用 browser_navigate 打开截图发给用户确认

### 5. 团队协作原则

```
用户 → coord agent → spec-designer（理解需求）
spec-designer → spec-frontend（输出 YAML 规格）
spec-designer → user（输出 HTML 原型确认）
frontend developer → spec-designer（review 反馈）
```

永远不要替 frontend developer 做实现决策（用什么框架、怎么组织代码）。

---

## 工作流程

> 每个 Step 的格式：`[输入] → 执行 → [输出]`，确保知道从哪里来、到哪里去。

### Step 1: 需求理解
**[输入]** 用户原始需求（文字描述或 coord agent 转述）

**[执行]**
1. 扫描项目上下文文件（见下方列表）
2. **涉及具体品牌时**：在问澄清问题之前，优先走 `§1.a 核心资产协议`——按类型搜 Logo/产品图/UI 截图，写入 `brand-spec.md`。**这条在问澄清问题之前执行**，因为事实错了问什么都是歪的。
3. 如果上下文不清晰，向 coord agent 请求补充
4. **需求模糊或无品牌时**：走 `§1.b 设计方向顾问 Fallback 模式`，推荐 3 个差异化方向，等用户选后再动手

**[输出]** 需求澄清文档（如果需要），否则标注"需求已明确 → 进入 Step 2"

**必读文件清单（按优先级）：**

| 优先级 | 文件路径 | 读什么 |
|--------|---------|--------|
| **P0** | `git pull origin master`（项目根目录） | 先拉最新代码，避免基于旧布局开发 |
| **P1** | `specs/feature/*/workbench-ide-chrome_feature.yaml` | **IDE 信息架构**：5区命名、region_map、默认尺寸 |
| **P1** | `specs/feature/*/workbench-layout_resize_feature.yaml` | **像素契约**：4条分隔条各自的 min/max/default 值 |
| **P1** | `frontend/src/lib/stores/workbench-layout-store.ts` | **已有 store**：layout store 的字段名/方法名，避免重复造轮子 |
| **P2** | `frontend/src/routes/+page.svelte`（主入口） | 路由结构和全局布局框架 |
| **P2** | `frontend/src/lib/components/workbench/*.svelte` | 已有组件列表，避免命名冲突 |
| **P3** | `specs/feature/*/workbench-conversation_feature.yaml` | AI 对话契约（消息结构/SSE 事件名） |
| **P3** | `frontend/src/lib/workbench/workbench-message-sse-bridge.ts` | SSE bridge 实现细节 |

> ⚠️ **git pull 前先 `git stash`**，避免 .env 等本地文件被覆盖导致冲突。



### Step 1.5: 风格知识库检索
**[触发条件]** 用户 prompt 中出现以下任一关键词：
- 品牌名（Linear / Notion / Claude / Vercel / Figma / Stripe / Superhuman / ...）
- 参考词（"参考"、"风格像"、"参考了"、"对标"、"借鉴"）
- 明确的设计偏好词（"暗黑系"、"暖色调"、"极简"、"厚重感"）

**[执行]**
1. **检测品牌关键词** → 映射到 DESIGN.md 目录名：
   ```
   Linear → linear.app    Notion → notion
   Claude → claude        Vercel → vercel
   Figma → figma          Stripe → stripe
   Superhuman → superhuman  Lovable → lovable
   Cal.com → cal         Expo → expo
   elevenlabs → elevenlabs  mintlify → mintlify
   （其余品牌：用户说 X → 直接查找 /project/awesome-design-md-cn/design-md/X/ 或 fuzzy match）
   ```
2. **读取对应 DESIGN.md**：提取以下 4 类 token：
   - **颜色**：brand accent、background、text、border 关键色值
   - **字体**：font-family、weight range、letter-spacing 规律
   - **布局**：border-radius 基准、spacing scale、shadow 模式
   - **动效**：duration、easing、动画类型克制程度
3. **风格锚点摘要**：对每个匹配的设计系统，输出：
   ```markdown
   ## [品牌名] 设计锚点
   - 主题：暗色优先 / 亮色优先 / 双模式
   - 品牌色：#xxxxxx（用于 primary/accent）
   - 字体：FontFamily + 特征 weight/letter-spacing
   - 边框：ultra-thin whisper border / solid border
   - 间距基准：8px / 4px / 16px
   - 圆角：6px(pill) / 8px(cards) / 9999px(buttons)
   - 动效：克制 / 适中 / 丰富
   ```
4. **多品牌融合**（当用户提到多个品牌时）：
   - 列出每个品牌的核心 token
   - 指出差异点（如 Linear 暗黑 vs Notion 暖白）
   - 给出融合建议（如"底色用 Linear #08090a，文字色用 Notion #37352f，卡片用 Linear 层级阴影"）
   - **trade-off 权衡**：当两个品牌的设计意图冲突时，给出决策树：
     ```
     温暖+专业 → Linear（暗色专业感+indigo品牌暖调）而非纯冷
     现代感+亲和 → Notion（暖白+大圆角）而非纯硬核
     沉浸感+信息密度 → Linear（暗底+密集卡片）而非扁平
     AI代码助手（温暖+专业）→ Claude+Linear混合：
       Claude提供温暖感（parchment #f5f4ed 底 + terracotta #c96442 暖品牌）
       Linear提供专业骨架（#08090a 暗底 + indigo #7170ff 专业感 + 密集卡片）
       融合方案：Linear底色 + Claude暖品牌色点缀 + Linear卡片层级 + Linear字重510
     ```

> **"专业感"精确定义** = 暗底背景(#08090a) + 信息密度(紧凑间距) + 字重510(Medium)+ 精确字间距(负letter-spacing)

   - **场景专用 token 补充**：根据用户场景，在基础 token 上追加：
     ```
     场景=看板/卡片列表 → 补充 card_padding/card_gap/avatar_size/badge_radius
     场景=表单/设置页 → 补充 input_height/input_radius/label_weight/error_spacing
     场景=仪表盘/数据 → 补充 chart_colors/table_row_height/grid_gap
     场景=聊天/消息 → 补充 bubble_radius/message_max_width/avatar_gap
     ```

**[输出]**
- 风格锚点摘要（每个品牌一行）
- 如有融合需求：给出 token 组合方案
- 传递给 Step 2 用于"继承哪个设计系统"的决策

**[fallback]** 如果品牌目录不存在：
- 用关键词（"暗黑"/"暖色"/"极简"）作为 query，模糊检索所有 DESIGN.md 的第1段 Visual Theme，找到最接近的 2-3 个
- 告知用户"未找到精确匹配，已检索到最接近的：[xxx]，是否采用？"

### 偏好词 → 品牌自动映射表（辅助 Step 1.5 触发）

当用户描述设计偏好（而非直接说品牌名）时，自动映射到对应 DESIGN.md：

| 用户偏好词 | 推荐锚点品牌 | 原因 |
|-----------|------------|------|
| "温暖友好"/"温暖"/"亲切" | `claude` | 全程暖色系（羊皮纸底+赤陶品牌），无冷色调 |
| "专业冰冷"/"极客"/"硬核" | `vercel` | 极简黑白，品牌色极克制 |
| "密集数据"/"表格控"/"信息多" | `stripe` / `linear.app` | 数据表格+多状态设计 |
| "暖白轻量"/"编辑器感"/"Notion风" | `notion` | 纯白底+12px圆角卡片+自由布局 |
| "暗色沉浸"/"沉浸感"/"现代感" | `linear.app` | #08090a 暗底+indigo品牌色 |
| "AI产品"/"助手感" | `claude` / `elevenlabs` | 温暖色+有机图形 |
| "卡片密集"/"看板"/"列表" | `linear.app` / `notion` | 两者都有优秀的列表/卡片组件 |
| "企业级"/"安全"/"规范" | `stripe` | 表单+状态+错误处理最规范 |

> 如果偏好词模糊匹配多个品牌，列出 Top2 让用户选。

### 风格复用场景（已有锚点 → 新页面）

当用户说"首页用了 X 风格，现在要做 Y 页面，怎么保证一致"，触发此场景：

**[执行]**
1. **确认锚点来源**（按优先级搜索）：
   ```
   搜索路径（按顺序）：
   1. specs/feature/<已有页面>/*_uiux.yaml         ← 最优先，精确匹配
   2. specs/feature/<已有页面>/*.yaml              ← 次优先，任意 yaml
   3. frontend/src/lib/stores/*.ts（提取 design_tokens 注释） ← 兜底
   4. 如果以上均不存在 → 询问用户锚点文件路径，或重新检索 DESIGN.md
   ```
   读取内容：design_tokens{}、design_references[]、components[] 中引用的 CSS 变量名

2. **锁定字段清单**（不允许改动）：
   ```
   design_tokens.colors.*              — 颜色必须完全复用
   design_tokens.typography.font_family — 字体 family 不变
   design_tokens.typography.weight_range — weight 范围不变
   design_tokens.spacing.base           — 间距基准不变
   design_tokens.border_radius.*        — 圆角基准不变
   design_tokens.animation.*            — 动效参数不变
   ```
3. **允许调整字段**（局部变体）：
   ```
   layout.grid_template        — 不同页面可调整布局比例
   components[].size          — 组件尺寸可微调
   regions[].components[]     — 区域组件组合可变化
   pages[].title/path         — 页面标题和路径
   ```
4. **操作步骤**：
   ```
   1. 读取已生成的锚点 YAML：specs/feature/<已有页面>/*_uiux.yaml
   2. 复制 design_tokens{} 和 design_references{} 到新 YAML
   3. 新页面验收时：对比新旧页面的 token 使用表
      → 在 frontend/src/ 下运行：
        grep -rn "#[0-9a-fA-F]\{3,8\}" --include="*.svelte" --include="*.css" | grep -v "var(--"
      → 如果有硬编码色（非 var() 包裹），报告 P1 问题要求修复
      → 补充检测 named colors：
        grep -rnE "(^|\s)(red|blue|green|yellow|orange|purple|white|black)\b" --include="*.svelte" --include="*.css"
   ```
5. **输出**：告知用户"已锁定以下字段：新页面将继承 X 品牌锚点，仅 Y Z 可调整"

**[输出]**
- 锁定字段清单（不可改）
- 允许调整字段（局部变体）
- 操作步骤
- 传递给 Step 2（概念设计）继续

### Step 2: 概念设计
**[输入]** Step 1 输出 + 项目上下文

**[执行]**
确定：
- **交互范式**：CRUD（列表/表单）还是 Agent 工作台（Thread/Run/Artifact）？
- **页面结构**：单页/多页/三栏工作台？
- **核心交互流程**：用户的第一步是什么？最常用的操作是什么？

如果不确定，输出 2-3 个方案简述，让用户选。

**[输出]** 概念设计决策摘要（范式/结构/核心流程），供检查点 A 确认

**检查点策略（实战经验）**：
- 上下文充分时：可跳过 A/B 直接到 C（如已读 YAML+stores+现有 spec）
- 上下文不足时：必须完整走 A→B→C 三步
- 永远在 C（YAML关键片段或HTML截图）后停，等用户 `ok` 再提交

> ⚠️ **[检查点 A]** 展示概念设计决策摘要（范式/结构/核心流程），等用户确认 `ok` 再进入 Step 3。

### Step 3: 视觉设计
**[输入]** Step 2 确认的概念决策

**[执行]**
确定：
- **设计系统**：继承还是自定义 token？
- **布局**：三栏/两栏/单栏/网格？
- **组件状态**：default/selected/error/loading 至少四种
- **动效**：克制，用 opacity 和 transform

**[输出]** 关键组件视觉草图描述（颜色/布局/状态）

> ⚠️ **[检查点 B]** 展示关键组件的视觉草图描述（颜色/布局/状态），等用户确认 `ok` 再进入 Step 4。

### Step 4: 输出交付物
**[输入]** Step 3 确认的视觉设计

**[执行]**
根据输出格式决策，走路径 A 或路径 B：

**路径 A（快速确认）**：HTML 原型
```
/prototypes/<feature-name>-prototype.html
```
- 单文件：CSS + HTML + JS 全部内联
- 文件 < 800 行，超过则拆分
- 输出后等待用户检查，有错修复，干净后确认

**路径 B（正式落地）**：YAML 规格文件
```
specs/feature/<feature>/<feature>_uiux.yaml
```
- 必须包含以下所有区块（见下方模板）
- YAML 写完后展示关键片段供检查点 C 确认

**[输出]** 完整的 YAML 规格文件或可运行的 HTML 原型

> ⚠️ **[检查点 C]** YAML 写完后、自审之前，暂停展示 YAML 关键片段，等用户确认 `ok` 再自审提交。

```yaml
# ============================================================
# YAML 规格模板（可直接复制修改）
# ============================================================
pages:
  - id: "<page-id>"           # 例: "run-engine-panel"
    path: "/run-engine"       # 路由路径
    title: "Run 执行状态面板"
    layout:
      type: "workspace"        # workspace|dashboard|blank|form-page|split
      grid_template: "240px [left] 1fr [main] 300px [right]"
    regions:
      - id: "main-area"
        position: "center"
        components: ["RunStatusCard", "ToolInvocationList"]

components:
  - id: "RunStatusCard"       # 组件 ID（snake_case）
    type: "composite"          # composite=复合组件，atomic=基础组件
    states:                    # 必须定义至少 4 种状态
      - state: "default"
        style: "border: 1px solid var(--color-border)"
      - state: "selected"
        style: "border: 2px solid var(--color-primary); box-shadow: 0 0 0 3px var(--color-primary-alpha)"
      - state: "executing"
        style: "border-color: var(--color-warning); animation: pulse 1s infinite"
      - state: "error"
        style: "border-color: var(--color-error); background: var(--color-error-bg)"

state_management:
  stores:
    - store: "runStore"        # Store 名称（PascalCase）
      description: "Run 执行状态管理"
      state:                  # 每个字段都要声明类型
        - name: "currentRun"
          type: "Run | null"
        - name: "runs"
          type: "Run[]"
        - name: "toolInvocations"
          type: "ToolInvocation[]"
        - name: "messageStream"
          type: "string"       # SSE 流式文字内容
      mutations:
        - name: "setCurrentRun"
          params: ["run: Run | null"]
        - name: "addToolInvocation"
          params: ["ti: ToolInvocation"]
        - name: "updateToolInvocation"
          params: ["id: string", "changes: Partial<ToolInvocation>"]
        - name: "appendMessage"
          params: ["delta: string"]

sse_events:
  - event: "run.started"
    handler: "runStore.setCurrentRun"
  - event: "tool.called"
    handler: "runStore.addToolInvocation"
  - event: "tool.completed"
    handler: "runStore.updateToolInvocation"
  - event: "message.delta"
    handler: "runStore.appendMessage"
  - event: "run.completed"
    handler: "runStore.setCurrentRun"   # 传 null 表示结束
  - event: "run.failed"
    handler: "runStore.updateToolInvocation"  # 更新最后一个的状态为 failed

interactions:
  - id: "retry-run"
    trigger: "点击重试按钮"
    steps:
      - action: "emit"
        event: "run.started"
        payload: "{ runId: currentRun.id }"

design_references:
  # Step 1.5 检索到的设计锚点来源
  - brand: "linear.app"
    role: "背景色/层级阴影/品牌色"
    tokens:
      bg_base: "#08090a"
      brand_primary: "#7170ff"
      border: "rgba(255,255,255,0.08)"
      radius_card: "8px"
      spacing_base: "8px"
  - brand: "notion"
    role: "卡片/暖色文字"
    tokens:
      bg_card: "#ffffff"
      text_primary: "#37352f"
      border: "rgba(0,0,0,0.1)"
      radius_card: "4px"

design_tokens:
  colors:
    primary: "#7170ff"
    success: "#22c55e"
    warning: "#f59e0b"
    error: "#ef4444"
    bg_base: "#08090a"
    text_primary: "#f7f8f8"
  animation:
    duration_fast: "120ms"
    duration_normal: "200ms"
    easing: "cubic-bezier(0.4, 0, 0.2, 1)"
```

> ⚠️ **[检查点 C]** YAML 写完后、自审之前，暂停展示 YAML 关键片段，等用户确认 `ok` 再自审提交。

### Step 5: 总结与下一步

**[输入]** Step 4 输出的交付物

**[执行]**
极其简短地总结：
- **设计决策**：做了什么关键选择，为什么
- **注意事项**：哪些地方需要 frontend developer 特别注意
- **下一步**：原型确认后做什么（生成 YAML / 开始实现）

**[输出]** 总结报告 + 下一步行动项

---

## Preview 模式（preview mode）

> 当 spec 包含 `preview:` 字段时，在人类确认 spec 之前，必须先生成预览产物。
> 这是 spec-designer 的一个子能力，不是独立 skill。

### 触发条件

满足以下任一条件时进入 preview mode：
- spec YAML 中存在 `preview:` 顶层字段
- 用户说"预览一下这个 spec"、"确认前给我看看效果"、"这个功能长什么样"

### 完整流程

**Step P1 · 读取 preview 配置**

读取 spec 文件中的 `preview:` 字段，提取：

```yaml
preview:
  level: L4          # L1/L2/L3/L4/L5
  artifact_type: behavior_preview  # 见下方类型映射
  trigger: confirm_button  # draft_change | confirm_button | spec_select
  confirmation_point: "这个功能行为是你想要的吗？"
  content: |
    用户操作 → 系统响应 → 界面变化的文字描述
    建议配合 AI 生成界面截图
```

**Step P2 · 根据 artifact_type 生成预览**

| `artifact_type` | 预览产物 | 生成方式 |
|---|---|---|
| `goal_diagram` | Mermaid flowchart | 读取 L1 content，生成 `mission → target_users → constraints` 关系图 |
| `architecture_diagram` | Mermaid flowchart | 读取 L2 modules_matrix，生成模块依赖图 |
| `module_structure` | 结构描述 + API 列表 | 读取 L3 public_api + state_definitions，整理成树状文本 |
| `behavior_preview` | 行为描述 + 界面截图（可选） | 读取 L4 behaviors，生成文字描述流程；调用 AI 生成界面截图 |
| `skeleton_preview` | 代码骨架文本预览 | 读取 L5 file_path + code_template，只读展示不写盘 |

**Step P3 · 展示预览给人类**

将预览产物（文字/Mermaid/截图）展示给用户，附带 `confirmation_point` 中的问题。

**Step P4 · 等待人类确认**

- 用户满意 → 继续后续流程
- 用户不满意 → 询问具体修改方向，回到 Step P1 重新生成

### L4 behavior_preview 特别说明

behavior_preview 是最常用的预览类型：
1. 读取 `content.behaviors[]`，将每个 behavior 翻译为"用户视角的操作流程"
2. 用 Mermaid sequence diagram 或文字步骤列表呈现
3. 如果 spec 中有 `design_tokens` 或 UI 组件描述，用 image_generate 生成界面示意截图
4. 截图作为辅助参考（不是必须的）

### 与 spec-first-workflow 的协作

当 Go agent 在 spec-first 流程中遇到带 `preview:` 的 spec 时：
1. Go agent 调用 `skill_load spec-designer` 激活本 skill
2. Go agent 执行 preview mode 流程
3. 预览展示给人类确认后，Go agent 继续 spec-first 流程

### 示例输入/输出

**输入**：spec 中有 `preview: { level: L4, artifact_type: behavior_preview, confirmation_point: "这个流程对吗？" }`

**输出**：
```
[Preview · L4 Behavior]

用户操作流程：
1. 用户点击「新建 Spec」按钮
2. 弹出 L1 类型选择向导
3. 用户填写 name + description
4. 点击「确认」→ spec 写入 specs/L1-goal/
5. 界面刷新，新 spec 出现在列表顶部

confirmation_point: 这个流程对吗？
```

---

## 特殊模式

### Review 模式（review existing specs）

当用户说"帮我看看这个界面"、"review 这个 spec"、"哪里体验不好"、"给我做个评审"时：

**Step R1: 定位要 review 的 spec**
- 读取 `specs/feature/<name>/*.yaml`
- 读取 `frontend/src/` 对应实现
- 如果有 HTML 原型，读取源码检查

**Step R2: 5 维度评估 + 雷达图输出**

强制要求：**每次评审必须同时输出文字评分 + SVG 雷达图**，两者缺一不可。

**5 维度评分标准：**

|| 维度 | 9-10 分标准 | 6-8 分标准 | 3-5 分标准 |
|------|------------|-----------|-----------|
| **哲学一致性** | 完美体现选定哲学，每个细节有依据 | 整体方向正确，个别细节偏离 | 仅表面模仿，未理解哲学内核 |
| **视觉层级** | 视线自然沿设计者意图流动 | 主次清晰，偶有1-2处模糊 | 信息平铺，入口不明确 |
| **细节执行** | 像素级精确，对齐/间距/颜色无瑕疵 | 整体精致，有微小问题 | 明显对齐错误、间距混乱 |
| **功能性** | 每个元素服务目标，零冗余 | 功能导向明确，少量装饰 | 装饰大于内容，用户需努力找信息 |
| **创新性** | 令人耳目一新，有独特表达 | 有自己的想法，不是简单模板 | 大量 cliché，模板套用 |

**强制输出：SVG 雷达图（5轴）**

在评审输出中，必须包含以下 SVG 雷达图（直接内联在 markdown 中）：

```svg
<!-- 五维度雷达图模板：把 (X,Y) 替换为实际分数坐标 -->
<!-- 分数范围 0-10，映射到半径 0-120 -->
<!-- 中心 (150,150)，最大半径 120 -->
<!-- 轴顺序：0°=哲学一致性, 72°=视觉层级, 144°=细节执行, 216°=功能性, 288°=创新性 -->
<svg viewBox="0 0 300 300" xmlns="http://www.w3.org/2000/svg" width="300" height="300">
  <defs>
    <style>
      .grid { stroke: rgba(255,255,255,0.1); fill: none; }
      .axis { stroke: rgba(255,255,255,0.15); stroke-width: 1; }
      .data { fill: rgba(113,112,255,0.3); stroke: #7170ff; stroke-width: 2; }
      .label { fill: rgba(255,255,255,0.7); font-family: system-ui; font-size: 11px; text-anchor: middle; }
      .score { fill: #7170ff; font-family: system-ui; font-size: 12px; font-weight: bold; text-anchor: middle; }
    </style>
  </defs>
  <!-- 背景网格 0.2/0.4/0.6/0.8/1.0 -->
  <polygon class="grid" points="150,126 165,150 150,174 135,150"/>
  <polygon class="grid" points="150,102 180,150 150,198 120,150"/>
  <polygon class="grid" points="150,78 195,150 150,222 105,150"/>
  <polygon class="grid" points="150,54 210,150 150,246 90,150"/>
  <polygon class="grid" points="150,30 225,150 150,270 75,150"/>
  <!-- 轴线 -->
  <line class="axis" x1="150" y1="150" x2="150" y2="30"/>
  <line class="axis" x1="150" y1="150" x2="264" y2="114"/>
  <line class="axis" x1="150" y1="150" x2="214" y2="236"/>
  <line class="axis" x1="150" y1="150" x2="86" y2="236"/>
  <line class="axis" x1="150" y1="150" x2="36" y2="114"/>
  <!-- 数据五边形（把下面5个坐标替换为实际分数×12后的坐标） -->
  <!-- 哲学X=7→(150,66), 视觉Y=8→(233,114), 细节Z=6→(193,218), 功能W=7→(107,218), 创新U=9→(67,114) -->
  <polygon class="data" points="150,66 233,114 193,218 107,218 67,114"/>
  <!-- 轴标签 -->
  <text class="label" x="150" y="18">哲学一致性</text>
  <text class="score" x="150" y="42">7</text>
  <text class="label" x="272" y="114">视觉层级</text>
  <text class="score" x="248" y="130">8</text>
  <text class="label" x="222" y="254">细节执行</text>
  <text class="score" x="207" y="238">6</text>
  <text class="label" x="78" y="254">功能性</text>
  <text class="score" x="93" y="238">7</text>
  <text class="label" x="28" y="114">创新性</text>
  <text class="score" x="52" y="130">9</text>
</svg>
```

> **使用方式**：将上方的 `score` 值替换为实际评审分数（共5处），`data` polygon 的 points 也同步更新（分数×12 转换为像素坐标）。

快速版本（如果无法生成 SVG）：输出 ASCII 雷达图：

```
                    哲学一致性 (X)
                           △
                          /|\
                         / | \
                        /  |  \
  视觉层级(Y) ←———●———————●———————→ 细节执行(Z)
                       /    |    \
                      /     |     \
                     /      |      \
                    ▼_______▼_______▼
              功能性(W)      V      创新性(U)
                      
实际分数：[哲学:X, 视觉:Y, 细节:Z, 功能:W, 创新:U]
```

**R-mode 输出格式（强制模板）：**
```markdown
## 5 维度评审结果

### 雷达图
[内联 SVG 雷达图 或 ASCII 雷达图]

### 评分详情

|| 维度 | 分数 | 关键发现 |
||------|------|---------|
|| 哲学一致性 | X/10 | ... |
|| 视觉层级 | X/10 | ... |
|| 细节执行 | X/10 | ... |
|| 功能性 | X/10 | ... |
|| 创新性 | X/10 | ... |
| **总分** | **X/10** | |

## 反 AI Slop 自检

|| 检查项 | 状态 | 备注 |
|--------|------|------|
|| 紫渐变 | ✅ OK / ⚠️ 命中 | 原因 + 建议 |
...

**Step R3: 反 AI Slop 自检**

详见 `references/anti-ai-slop.md`（含完整清单 + 交付前自检流程 + 设计原则 #0）。

快速版本（内嵌，用于评审输出）：

|| 元素 | 判断 |
|------|------|------|
| 激进紫渐变 / Emoji 作图标 / 圆角卡片+左 border / 系统字体做 display | **品牌 spec 写了才能用** |
| SVG 画人/物 / CSS 剪影代替产品图 / 编造 stats | **永远禁止** |
| 赛博霓虹 / 无意义 icon 每处配 | **品牌 spec 写了或承载差异化信息才能用** |

**判断原则**：「品牌 spec 里明写了才能用」是唯一破例理由。没有 spec 就默认禁用。

**Step R4: 问题分级 + 输出**

| 等级 | 定义 | 示例 |
|------|------|------|
| P0 Critical | 必须修，影响核心功能 | 交互逻辑错误、功能缺失 |
| P1 High | 重要，应该修 | 层级不清晰、色值不规范 |
| P2 Medium | 建议修，不紧急 | 微调间距、字体换更好的 |
| P3 Low | 可选，不影响功能 | 美化细节 |

**输出格式：**
```markdown
## 5 维度评审结果

| 维度 | 分数 | 关键发现 |
|------|------|---------|
| 哲学一致性 | X/10 | ... |
| 视觉层级 | X/10 | ... |
| 细节执行 | X/10 | ... |
| 功能性 | X/10 | ... |
| 创新性 | X/10 | ... |

## 反 AI Slop 自检

| 检查项 | 状态 | 备注 |
|--------|------|------|
| 紫渐变 | ✅ OK / ⚠️ 命中 | 原因 + 建议 |

## Keep / Fix / Quick Wins

**Keep（做得好）：**
- ...

**Fix（需修复）：**
- P0: ...
- P1: ...

**Quick Wins（5 分钟能做的事）：**
- ...
```

> ⚠️ **[检查点 R]** 评审完成后，暂停展示结果，等用户确认 `ok` 再提交。如果用户还有补充信息，据此调整评审结论。

> ⚠️ **评审的是设计，不是设计师**。描述问题时用"这个元素显得"而非"你做的"。

---

## 边界条件

| 场景 | 处理方式 |
|------|---------|
| 用户只说了功能名，没说细节 | 假设最小可用方案，标注"[待确认]" |
| 设计与现有实现冲突 | 标注冲突点，建议与 frontend developer 对齐 |
| 需求超出当前项目范围 | 标注"[新 feature]"，建议开独立 ticket |
| 需要用户做决策 | 输出选项，格式：`选项1 / 选项2 / 选项3` |
| 原型文件超过 800 行 | 拆分为多个 HTML 模块文件 |
| **SSE 连接断开** | 显示"连接中断，正在重连..."状态，自动触发 reconnect 逻辑 |
| **SSE 事件乱序** | 前端按 `eventId` 字段排序，丢弃重复 eventId |
| **SSE 消费者失败** | 提供手动刷新按钮，不自动重试（避免状态不一致） |

---

## 自审清单（提交前必须检查）

在提交 YAML 规格或 HTML 原型之前，逐项检查：

```
□ YAML 规格
  □ 每个 store 有 state[] 声明
  □ 每个 store 有 mutations[] 声明（可选但推荐）
  □ 有 sse_events[] 声明
  □ 有 interactions[]（至少 3 个核心交互）
  □ 有 design_tokens（颜色/字体/动效）
  □ 组件有 states[]（至少 4 种：default/selected/error/loading）
  □ 无硬编码颜色值（全部用 token）
  □ 无 scrollIntoView（用其他 DOM 滚动方法）
  □ 有 result[] 数组（记录该 spec 的最终产出文件，含 spec_link + confirmed_by）

□ HTML 原型
  □ 所有交互有响应（hover/click/focus/keyboard）
  □ 文件顶部有 spec 来源注释（<!-- spec: xxx.yaml -->）
  □ 颜色用 CSS 变量而非硬编码（fallback 色值来自 design_tokens）
  □ 生成后用 browser_navigate 打开截图发给用户确认
  □ 保存到 /prototypes/<feature>-prototype.html
  □ patch 后检查：函数不重复、ID 不冲突、init 调用完整

□ Result Tracker（产出追踪模态框）
  □ 工具栏入口带徽章（待确认数量）
  □ 模态框头部有实时统计（已确认/待确认/总计）
  □ 过滤栏：全部/待确认/已确认 + 刷新 + 批量确认
  □ 每条产出有跳转查看（HTML→新标签页/源码→路径信息/YAML→下载）
  □ 逐一确认后自动更新 confirmed_at + 实时刷新列表
  □ localStorage 持久化确认状态，重启不丢失
  □ ESC + 点击遮罩均可关闭模态框

□ 通用
  □ 调用了 spec-frontend skill（读取了 YAML 格式规范）
  □ 调用了 spec-ui skill（读取了设计令牌）
  □ 动效不超过 3 种类型
  □ 组件状态覆盖完整

□ 核心资产协议（涉及具体品牌时必检）
  □ 是否先 WebSearch 验证了品牌存在性和资产 URL（原则#0）——**这条在最前**
  □ 是否有 product-facts.md（含品牌验证结果）
  □ 是否先搜了 Logo/产品图/UI（不只是 grep 色值）
  □ 是否有 brand-spec.md（含 Logo 路径/产品图路径/色板/气质关键词）
  □ Logo 是否作为 `<img>` 引用真实文件（不是 CSS 剪影）
  □ 产品图是否引用真实文件（不是 CSS 剪影/SVG 手画）
  □ 如果 Logo 找不到，是否停了问用户（而非硬做）

□ 增强评审（review 时必检）
  □ 是否输出了 5 维度评审（哲学一致性/视觉层级/细节执行/功能性/创新性）
  □ 是否输出了 **SVG 雷达图或 ASCII 雷达图**（两者必有其一）——**这条新增**
  □ 是否执行了反 AI Slop 检查（见 `references/anti-ai-slop.md`）
  □ 是否输出了 Keep/Fix/Quick Wins
  □ 检查点 R 是否暂停等用户确认
  □ 描述问题时是否用「这个元素显得」而非「你做的」

□ 反 AI Slop（HTML 原型生成时必检）
  □ 无紫渐变（品牌 spec 允许除外）
  □ 无 emoji 图标（品牌 spec 允许除外）
  □ 无 SVG 手画人/物（用真图或 placeholder）
  □ 无 CSS 剪影代替产品图（用真图或 placeholder）
  □ 无系统字体做 display（品牌 spec 允许除外）
  □ 无编造 stats/quotes（留空白或真实数据）
```

---

## 质量标准

1. **HTML 原型**：所有交互都有响应（hover/click），无死链接
2. **YAML 规格**：每个 store 有 state[]，每个组件有 states[]
3. **动效**：不超过 3 种动画类型
4. **颜色**：全部使用 token 或 oklch，无硬编码 #fff/red 等
5. **可访问性**：按钮有 label，表单有 placeholder，图片有 alt
6. **反 AI Slop**：生成原型时，对照以下清单自检，无命中才交付：
   - 无紫渐变（品牌 spec 允许除外）
   - 无 emoji 图标（品牌 spec 允许除外）
   - 无 SVG 手画人/物（用真图）
   - 无 CSS 剪影代替产品图（用真图）
   - 无系统字体做 display（品牌 spec 允许除外）
   - 无编造 stats/quotes（留空白或用真实数据）

---

## 与其他 Skills 的衔接

```
输入 ← coord agent（任务分发）
     ← spec-concept（设计理念）
     ← spec-ui（设计系统 token）
     ← vibex-workbench（项目上下文）

输出 → spec-frontend（YAML 规格）
     → user（HTML 原型）→ 用 browser_navigate 截图确认
     → frontend developer（review 反馈）
```

---

## 实战案例记录

### canvas-workbench HTML 原型（2026-04-20）

**背景**：三栏布局工作台（左侧线程列表/中间对话区/右侧 Canvas），需集成 Result Tracker 产出追踪模态框。

**迭代过程**：
- Step 1: 生成基础三栏布局 + Canvas 节点链路 + 三条路由颜色编码
- Step 2: 加 Result Tracker 按钮 + 模态框（CSS + HTML + JS）
- Step 3: 修 3 个 bug
  1. **重复函数定义**：patch 追加代码段时，`escapeHtml` 在原文件已有定义，导致 JS 报错。检查方法：`search_files` 搜 `function xxx`，确认无重复后再 patch
  2. **ID 对不上**：CSS 类名 `.rt-done` 被误当成元素 ID `id="rt-done"`，JS 查 `getElementById('rtDoneCount')` 找不到 → 统计数字永远是 0。修复：确保 CSS 类名和 JS ID 完全独立
  3. **init 漏调**：`updateFilterCounts()` 在 JS 里定义但 init 没调 → 初始加载时数字为 0。修复：init 中 `loadConfirmedMap()` 后紧跟 `updateFilterCounts(); updateToolbarBadge()`

**关键教训**：HTML prototype 用 patch 增量修改时，每次 patch 后立刻 `browser_navigate` + `browser_console` 检查 JS 报错，比最后一次性检查更高效。

**触发**：用户说"用技能跑出来一版 vibex-workbench 的样式"
**路径**：路径A（HTML 原型）
**输入**：canvas-workbench_uiux.yaml（1123行，已含5种节点类型/toolbar/minimap/design_tokens）
**产出**：/root/vibex-workbench/prototypes/canvas-workbench-prototype.html（30KB，9节点，完整交互）
**耗时**：约 15 分钟（生成+自审+截图确认）
**用户反馈**："不错不错"
**结论**：YAML spec → HTML 原型比跑 npm run dev 更快确认视觉方向，适合快速迭代
