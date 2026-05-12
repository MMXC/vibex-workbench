import { defineBackground } from 'wxt/sandbox';

export default defineBackground(() => {
  let session: any = null;

  browser.runtime.onMessage.addListener((msg: any, _sender: any, sendResponse: any) => {
    if (msg.type === 'START_SELECTION') {
      session = { id: crypto.randomUUID(), status: 'extracting', selectedHTML: '', sourceUrl: '', timestamp: Date.now(), rounds: [] };
      sendResponse({ ok: true, sessionId: session.id });
      return;
    }
    if (msg.type === 'ELEMENT_SELECTED') {
      if (!session) return sendResponse({ ok: false, error: 'No active session' });
      session.selectedHTML = msg.html;
      session.sourceUrl = msg.sourceUrl;
      session.status = 'pending_confirmation';
      session.rounds.push({ round: 1, question: '这个区域的语义功能是什么？预期的交互行为？', answer: '', confirmed: false, at: new Date().toISOString() });
      sendResponse({ ok: true, session });
      return;
    }
    if (msg.type === 'ANSWER_ROUND') {
      if (!session) return sendResponse({ ok: false, error: 'No session' });
      const r: any = session.rounds[session.rounds.length - 1];
      if (r) { r.answer = msg.answer; r.confirmed = true; }
      if (session.rounds.length >= 3) {
        session.status = 'confirmed';
        session.derivedYaml = buildYaml(session);
      } else {
        session.rounds.push({ round: session.rounds.length + 1, question: '下一个关键设计点？', answer: '', confirmed: false, at: new Date().toISOString() });
      }
      sendResponse({ ok: true, session });
      return;
    }
    if (msg.type === 'GET_SESSION') { sendResponse(session); return; }
    if (msg.type === 'EXPORT') {
      if (!session || session.status !== 'confirmed') return sendResponse({ ok: false, error: 'Not ready' });
      sendResponse({ ok: true, html: session.selectedHTML, yaml: session.derivedYaml });
      return;
    }
    return false;
  });

  function buildYaml(s: any): string {
    const qa = s.rounds.map((r: any) => `    - Q: ${r.question}\n      A: ${r.answer}`).join('\n');
    return `spec:
  level: L3
  name: extracted-region
  title: 提取原型片段
  parent: vibex-prototype-driven-spec
  status: draft
prototype:
  file: .vibex/specs/prototypes/extracted.html
  source_url: "${s.sourceUrl}"
io_contract:
  inputs: []
  outputs: []
behaviors:
${qa}
`;
  }
});
