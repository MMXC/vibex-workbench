// background.js - Service Worker
// 负责：存储协调、文件写入协调（通过 chrome.downloads 或消息传递）

// 当前提取会话状态
let currentSession = null;

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.type === 'EXTRACT_REGION') {
    handleExtractRegion(message.data, sendResponse);
    return true; // 异步响应
  }
  if (message.type === 'SAVE_TO_WORKSPACE') {
    handleSaveToWorkspace(message.data, sendResponse);
    return true;
  }
  if (message.type === 'GET_SESSION') {
    sendResponse(currentSession);
    return false;
  }
});

async function handleExtractRegion(data, sendResponse) {
  // TODO: 调用 AI 分析 DOM 结构，生成提取方案
  // 目前返回简化版
  currentSession = {
    ...data,
    status: 'pending_confirmation',
    timestamp: Date.now()
  };
  sendResponse({ success: true, session: currentSession });
}

async function handleSaveToWorkspace(data, sendResponse) {
  // TODO: 通过 File System Access API 或 clipboard 导出
  // 用户需要在 workbench 中手动粘贴/导入
  sendResponse({ success: true, path: data.targetPath });
}
