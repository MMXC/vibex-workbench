/**
 * workbench-canvas-ui runtime
 * Static HTML + data-bind injection helper
 */
(function () {
  function byPath(obj, path) {
    return path.split('.').reduce((acc, k) => (acc && acc[k] !== undefined ? acc[k] : undefined), obj);
  }

  function renderList(el, value) {
    if (!Array.isArray(value)) return;
    el.innerHTML = '';
    for (const item of value) {
      const li = document.createElement('li');
      li.textContent = String(item);
      el.appendChild(li);
    }
  }

  function renderKV(el, value) {
    if (!value || typeof value !== 'object') return;
    el.innerHTML = '';
    for (const [k, v] of Object.entries(value)) {
      const row = document.createElement('div');
      row.className = 'wc-kv-row';
      row.innerHTML = '<span class="k"></span><code class="v"></code>';
      row.querySelector('.k').textContent = k;
      row.querySelector('.v').textContent = String(v);
      el.appendChild(row);
    }
  }

  function inject(root, payload) {
    const nodes = root.querySelectorAll('[data-bind]');
    for (const node of nodes) {
      const key = node.getAttribute('data-bind');
      const value = byPath(payload, key);
      if (value === undefined || value === null) continue;
      if (Array.isArray(value) && node.tagName === 'OL') {
        renderList(node, value);
        continue;
      }
      if (typeof value === 'object') {
        renderKV(node, value);
        continue;
      }
      node.textContent = String(value);
    }
  }

  window.WorkbenchCanvasUI = {
    inject,
  };
})();
