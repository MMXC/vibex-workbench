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
4. 选择 `extensions/prototype-extractor/dist`

## 使用流程

```
vibex-workbench 原型抽屉
    → 点击「浏览器完善区域」
    → Chrome 打开 sidepanel
    → 在目标网页点击「选择区域」
    → 选中满意区块
    → 复制 HTML 或导出
    → workbench 预览原型
    → agent 基于原型 grill-me 澄清 → 生成 spec
```

## 项目结构

```
src/
  background.js   # Service Worker（会话管理）
  content.js       # Content Script（选区交互）
  content.css      # 选区高亮样式
  sidepanel.html   # 侧边栏 UI
  sidepanel.js     # 侧边栏逻辑
```
