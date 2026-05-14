// background.ts — Proto Spec background service worker
// 作为侧边栏 / content script 的消息 relay 中心
import type { Browser } from 'wxt/browser';

type SidePanelApi = {
  setPanelBehavior: (opts: { openPanelOnActionClick: boolean }) => Promise<void>;
};

type BrowserWithSidePanel = typeof browser & { sidePanel?: SidePanelApi };

/** Workbench 前端所在 tab 的 host（用于 relay）。 */
const WORKBENCH_HOST_PATTERNS = [
  '*://wails.localhost:*/*',
  '*://localhost:34115/*',
];

export default defineBackground(() => {
  let activeTabId: number | null = null;

  async function enableSidePanelOnToolbarClick() {
    const sidePanel = (browser as BrowserWithSidePanel).sidePanel;
    if (!sidePanel?.setPanelBehavior) return;
    try {
      await sidePanel.setPanelBehavior({ openPanelOnActionClick: true });
    } catch (e) {
      console.warn('[Proto Spec BG] sidePanel.setPanelBehavior failed', e);
    }
  }

  void enableSidePanelOnToolbarClick();

  browser.runtime.onInstalled.addListener(() => {
    console.log('[Proto Spec BG] Extension installed');
    void enableSidePanelOnToolbarClick();
  });

  // 监听 tab 激活
  browser.tabs.onActivated.addListener(async (info) => {
    activeTabId = info.tabId;
  });

  // 监听来自侧边栏等 UI 的消息
  browser.runtime.onMessage.addListener((msg, sender, sendResponse) => {
    handleMessage(msg, sender).then(sendResponse);
    return true;
  });

  async function handleMessage(msg: any, sender: Browser.runtime.MessageSender) {
    const { type, payload } = msg;

    switch (type) {
      // 侧边栏 → Content: 转发 Extension → Page 指令
      case 'ext:send':
        return await sendToPage(payload);

      // 请求当前 tab ID
      case 'ext:getActiveTab':
        return { tabId: activeTabId };

      /**
       * Content（页面 postMessage 中继）或 overlay 内联逻辑 → 广播给侧栏等 extension pages。
       * 页面 runtime 用户事件：elem:click、runtime:ready 等，payload 形状为 { type, data }。
       */
      case 'popup:receive': {
        const forward = {
          ...(payload && typeof payload === 'object' ? payload : { payload }),
          tabId: sender.tab?.id,
        };
        try {
          await browser.runtime.sendMessage({
            type: 'popup:receive',
            payload: forward,
          });
        } catch {
          /* 侧栏未打开等场景下可无接收方 */
        }
        return { ok: true };
      }

      /**
       * ext:relayToFrontend — 扩展侧栏/background 主动将数据中继到 Workbench WebView。
       * 1. 先在 content script（运行在 Workbench WebView 中）中查找目标 tab
       * 2. 通过 tabs.sendMessage('relay:toFrontend') 触发 content script postMessage
       * 3. content script 再 window.postMessage 到 Workbench 前端
       *
       * payload: { threadId, kind, data }
       */
      case 'ext:relayToFrontend': {
        return await relayToWorkbench(payload);
      }

      /**
       * ext:writeSessionContext — 将 Workbench session 上下文写入 storage
       *（当用户在 Workbench 点击「新标签打开」时，URL 带 _vibex_ctx 参数，
       *  content script 解析后调用此 handler 将上下文固化到 storage，供侧栏读取）
       */
      case 'ext:writeSessionContext': {
        if (payload && typeof payload === 'object') {
          await browser.storage.local.set({ vibexWorkbenchSession: payload });
        }
        return { ok: true };
      }

      /**
       * ext:getSessionContext — 读取 Workbench session 上下文
       */
      case 'ext:getSessionContext': {
        const result = await browser.storage.local.get('vibexWorkbenchSession');
        return { session: result.vibexWorkbenchSession ?? null };
      }

      default:
        console.warn('[Proto Spec BG] Unknown message type:', type);
        return { error: 'unknown type' };
    }
  }

  /**
   * 找到 Workbench WebView tab 并发送 relay 消息。
   * payload: { threadId, kind, data }
   */
  async function relayToWorkbench(payload: any): Promise<{ ok: boolean; error?: string }> {
    try {
      const tabs = await browser.tabs.query({ url: WORKBENCH_HOST_PATTERNS });
      const targetTab = tabs.find(t => t.id && t.id !== activeTabId) ?? tabs[0];
      if (!targetTab?.id) {
        return { ok: false, error: 'Workbench tab not found' };
      }
      await browser.tabs.sendMessage(targetTab.id, {
        type: 'relay:toFrontend',
        payload,
      });
      return { ok: true };
    } catch (err: any) {
      if (err.message?.includes('Receiving end does not exist')) {
        return { ok: false, error: 'Workbench content script not ready' };
      }
      return { ok: false, error: err.message };
    }
  }

  /**
   * 监听 storage 变化，作为扩展侧栏 → content script → Workbench 的备用 relay 触发器。
   * 流程：侧栏写 storage → bg 监听变化 → tabs.sendMessage 触发 content script relay
   */
  browser.storage.onChanged.addListener((changes, area) => {
    if (area !== 'local') return;

    // 一次性事件：extToFrontendEvent
    const evt = changes['extToFrontendEvent'];
    if (evt?.newValue) {
      relayToWorkbench(evt.newValue).catch(() => {});
      browser.storage.local.remove('extToFrontendEvent').catch(() => {});
    }

    // Workbench session 上下文写入（来自 content script）
    if (changes['vibexWorkbenchSession']?.newValue) {
      // 上下文已写入 storage，侧栏可自行读取；无需额外操作
      console.log('[Proto Spec BG] vibexWorkbenchSession updated:', changes['vibexWorkbenchSession'].newValue);
    }
  });

  async function sendToPage(payload: any): Promise<any> {
    // 获取当前 tab
    if (!activeTabId) {
      const tabs = await browser.tabs.query({ active: true, currentWindow: true });
      if (tabs[0]?.id) {
        activeTabId = tabs[0].id;
      } else {
        return { error: 'no active tab' };
      }
    }

    try {
      // 向 content script 发送消息 → runtime 处理
      const response = await browser.tabs.sendMessage(activeTabId, {
        type: payload.type,
        payload: payload.data,
      });
      return response;
    } catch (err: any) {
      if (err.message?.includes('Receiving end does not exist')) {
        return { error: 'content not ready, please refresh the page' };
      }
      return { error: err.message };
    }
  }
});
