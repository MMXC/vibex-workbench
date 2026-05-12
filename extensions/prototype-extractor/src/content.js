// content.js - Content Script
// 负责：选区交互、高亮、注入脚本

let selectionMode = false;
let selectedElement = null;
let overlay = null;

// 监听来自 sidepanel 的消息
chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.type === 'START_SELECTION') {
    enableSelectionMode();
    sendResponse({ success: true });
  }
  if (message.type === 'CANCEL_SELECTION') {
    disableSelectionMode();
    sendResponse({ success: true });
  }
  if (message.type === 'GET_SELECTED_HTML') {
    sendResponse({ html: getSelectedHTML() });
  }
});

function enableSelectionMode() {
  selectionMode = true;
  document.addEventListener('mouseover', handleMouseOver);
  document.addEventListener('mouseout', handleMouseOut);
  document.addEventListener('click', handleClick);
  document.body.style.cursor = 'crosshair';
  createOverlay();
}

function disableSelectionMode() {
  selectionMode = false;
  document.removeEventListener('mouseover', handleMouseOver);
  document.removeEventListener('mouseout', handleMouseOut);
  document.removeEventListener('click', handleClick);
  document.body.style.cursor = '';
  removeOverlay();
}

function createOverlay() {
  overlay = document.createElement('div');
  overlay.id = 'vibex-extractor-overlay';
  Object.assign(overlay.style, {
    position: 'fixed', top: 0, left: 0,
    width: '100vw', height: '100vh',
    zIndex: 2147483647,
    pointerEvents: 'none'
  });
  document.body.appendChild(overlay);
}

function removeOverlay() {
  overlay?.remove();
  overlay = null;
}

function handleMouseOver(e) {
  if (!selectionMode) return;
  e.target.style.outline = '2px solid #6366f1';
  e.target.style.outlineOffset = '2px';
}

function handleMouseOut(e) {
  if (!selectionMode) return;
  e.target.style.outline = '';
  e.target.style.outlineOffset = '';
}

function handleClick(e) {
  if (!selectionMode) return;
  e.preventDefault();
  e.stopPropagation();
  selectedElement = e.target;
  disableSelectionMode();
  extractElement(e.target);
}

function getSelectedHTML() {
  if (!selectedElement) return null;
  return selectedElement.outerHTML;
}

async function extractElement(el) {
  // 序列化选中元素的 DOM
  const html = el.outerHTML;
  // 计算样式（简化版，完整版需要 computedStyles）
  const styles = el.getAttribute('style') || '';

  const payload = {
    html,
    styles,
    tag: el.tagName.toLowerCase(),
    sourceUrl: window.location.href,
    timestamp: new Date().toISOString()
  };

  // 发送到 background 处理
  chrome.runtime.sendMessage({ type: 'EXTRACT_REGION', data: payload });
}
