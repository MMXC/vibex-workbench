# VibeX Prototype Extractor

Chrome Extension，从浏览器网页提取原型片段，导入 VibeX spec 治理体系。

## 安装

```bash
cd extensions/prototype-extractor
npm install
npm run build
```

然后在 Chrome 中：
1. 打开 `chrome://extensions/`
2. 开启「开发者模式」
3. 点击「加载已解压的扩展程序」
4. 选择 `extensions/prototype-extractor/.output/chrome-mv3`

## 使用流程

```
vibex-workbench 原型抽屉
    → 点击「浏览器完善区域」
    → Chrome 打开 sidepanel
    → 在目标网页点击「开始选区」
    → agent 基于选区逐轮 grill-me 澄清（Q&A 循环）
    → 用户确认 3 轮后导出
    → 复制 HTML + YAML 粘贴回 workbench
```

## 项目结构

```
entrypoints/
  background.ts       # Service Worker（会话管理 + AI 调度）
  content.ts          # Content Script（选区交互、高亮）
  sidepanel.html      # 侧边栏 UI（grill-me 风格 Q&A 对话）
  sidepanel.sidepanel.html  # WXT 入口
.wxt/                 # WXT 内部文件
.output/chrome-mv3/   # 构建输出（加载到 Chrome）
```

## Sidepanel UI（grill-me 风格）

- Header：状态标签（空闲 / 选区中 / 澄清中 / 已完成）
- Conversation：R{n} 编号的 Q&A 轮次卡片
- Answer area：pending 轮次的 Q 输入 + A 回答区
- Export area：confirmed 后显示 HTML 预览 + YAML，可一键复制
