// sidepanel.js - 侧边栏 UI 逻辑

let extractedData = null;

const btnExtract = document.getElementById('btnExtract');
const btnCopyHTML = document.getElementById('btnCopyHTML');
const btnExport = document.getElementById('btnExport');
const statusDot = document.getElementById('statusDot');
const statusText = document.getElementById('statusText');
const previewSection = document.getElementById('previewSection');
const extractedHTML = document.getElementById('extractedHTML');
const elTag = document.getElementById('elTag');
const elSource = document.getElementById('elSource');

btnExtract.addEventListener('click', async () => {
  setStatus('active', '选区模式：点击网页元素');
  // 通知 content script 开始选区
  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
  chrome.tabs.sendMessage(tab.id, { type: 'START_SELECTION' });
});

btnCopyHTML.addEventListener('click', () => {
  if (!extractedData) return;
  navigator.clipboard.writeText(extractedData.html).then(() => {
    btnCopyHTML.textContent = '已复制 ✓';
    setTimeout(() => btnCopyHTML.textContent = '复制 HTML', 1500);
  });
});

btnExport.addEventListener('click', async () => {
  if (!extractedData) return;
  // TODO: 打开 workbench 导入流程
  // 目前提示用户手动复制粘贴
  statusText.textContent = '请在 workbench 原型抽屉导入';
});

// 监听 background 的响应
chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.type === 'EXTRACT_REGION') {
    onExtracted(message.data);
  }
});

async function onExtracted(data) {
  extractedData = data;
  setStatus('idle', '已提取');
  previewSection.style.display = 'block';
  elTag.textContent = `<${data.tag}>`;
  elSource.textContent = new URL(data.sourceUrl).hostname;
  extractedHTML.textContent = data.html.substring(0, 500) + (data.html.length > 500 ? '...' : '');
}

function setStatus(state, text) {
  statusDot.className = 'status-dot ' + state;
  statusText.textContent = text;
}

// ESC 取消选区
document.addEventListener('keydown', async (e) => {
  if (e.key === 'Escape') {
    const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
    chrome.tabs.sendMessage(tab.id, { type: 'CANCEL_SELECTION' });
    setStatus('idle', '已取消');
  }
});
