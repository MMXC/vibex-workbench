import { defineConfig } from 'wxt';

export default defineConfig({
  manifest: {
    name: 'VibeX Prototype Extractor',
    version: '0.1.0',
    description: '从浏览器网页提取原型片段，导入 VibeX spec 治理体系',
    permissions: ['activeTab', 'storage'],
    host_permissions: [],
    side_panel: {
      default_path: 'sidepanel.html',
    },
    action: {
      default_title: '打开 VibeX 原型提取器',
    },
  },
});
