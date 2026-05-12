import { defineContentScript } from 'wxt/sandbox';

export default defineContentScript({
  matches: ['<all_urls>'],
  css: [],
  main(ctx) {
    let selecting = false;

    ctx.addListener((window as any).chrome.runtime.onMessage, (msg: any, _sender: any, sendResponse: any) => {
      if (msg.type === 'START_SELECTION') { startSelect(); sendResponse({ ok: true }); }
      if (msg.type === 'CANCEL_SELECTION') { stopSelect(); sendResponse({ ok: true }); }
      return false;
    });

    document.addEventListener('mouseover', onHover, true);
    document.addEventListener('mouseout', onHoverOut, true);
    document.addEventListener('click', onClick, true);
    document.addEventListener('keydown', (e: KeyboardEvent) => { if (e.key === 'Escape') stopSelect(); }, true);

    function startSelect() {
      selecting = true;
      document.body.style.cursor = 'crosshair';
    }

    function stopSelect() {
      selecting = false;
      document.body.style.cursor = '';
      document.querySelectorAll('.vibex-highlight').forEach(el => el.classList.remove('vibex-highlight'));
    }

    function onHover(e: Event) {
      if (!selecting) return;
      (e.target as HTMLElement).classList.add('vibex-highlight');
    }

    function onHoverOut(e: Event) {
      if (!selecting) return;
      (e.target as HTMLElement).classList.remove('vibex-highlight');
    }

    function onClick(e: Event) {
      if (!selecting) return;
      e.preventDefault();
      e.stopPropagation();
      const el = e.target as HTMLElement;
      stopSelect();
      (window as any).chrome.runtime.sendMessage({ type: 'ELEMENT_SELECTED', html: el.outerHTML, sourceUrl: location.href });
    }
  },
});
