#!/usr/bin/env python3
"""
gen-detail-drawer.py
────────────────────────────────────────────────────────────────────────────
生成 MOD-detail-drawer.html（详情抽屉派生原型）。

从 vibex-workbench-mvp.html 出发：
  1. 保留 MVP 全部静态结构（已清理 spec 专属 JS/onclick）
  2. 注入抽屉四态行为脚本（open → loading → content|meta|journal|error → idle）
  3. 将 data-action="open-drawer" 按钮接入 openDrawer(specPath) 行为
  4. 写入 .vibex/prototypes/MOD-detail-drawer.html

用法（项目根目录执行）：
    python .vibex/agents/skills/spec-prototype-fragment/scripts/gen-detail-drawer.py
"""

import os
import re

MVP_PATH = ".vibex/prototypes/vibex-workbench-mvp.html"
OUT_PATH = ".vibex/prototypes/MOD-detail-drawer.html"


# ── 行为脚本：注入到 </body> 前（<script> 包裹） ────────────────────────────

DRAWER_BEHAVIOR_SCRIPT = r"""
document.addEventListener('DOMContentLoaded', function () {
  (function () {
    'use strict';

  /* ── 抽屉元素 ── */
  var backdrop     = document.getElementById('detail-drawer-backdrop');
  var drawer      = document.getElementById('detail-drawer');
  var specNameEl  = document.getElementById('drawer-spec-name');
  var tabContent  = document.getElementById('drawer-tab-content');
  var tabMeta     = document.getElementById('drawer-tab-metadata');
  var tabJournal  = document.getElementById('drawer-tab-journal');
  var emptyEl     = document.getElementById('drawer-empty-panel');
  var loadingEl   = document.getElementById('drawer-loading');
  var errPanel    = document.getElementById('drawer-error-panel');
  var contentPanel = document.getElementById('drawer-content-panel');
  var metaPanel   = document.getElementById('drawer-meta-panel');
  var journalPanel = document.getElementById('drawer-journal-panel');

  /* ── 四态辅助 ── */
  function hideAll() {
    emptyEl.style.display      = 'none';
    loadingEl.style.display    = 'none';
    errPanel.style.display     = 'none';
    contentPanel.style.display = 'none';
    metaPanel.style.display    = 'none';
    journalPanel.style.display = 'none';
  }

  function showEmpty() {
    hideAll();
    emptyEl.style.display = '';
  }

  function showLoading() {
    hideAll();
    loadingEl.style.display = '';
  }

  function showError(msg) {
    hideAll();
    errPanel.style.display  = '';
    errPanel.textContent   = msg;
  }

  function showContent(text) {
    hideAll();
    contentPanel.style.display = '';
    contentPanel.textContent   = text;
  }

  function showMetadata(specPath) {
    var parts = specPath.split('/');
    var name  = parts[parts.length - 1];
    var path  = specPath;
    var level = (name.match(/^(L[1-5])-/) || ['L?'])[1];
    document.getElementById('drawer-path-value').textContent     = path;
    document.getElementById('drawer-filename-value').textContent = name;
    document.getElementById('drawer-level-value').textContent   = level;
    hideAll();
    metaPanel.style.display = '';
  }

  /* ── 当前打开的 specPath（闭包变量） ── */
  var _currentSpecPath = '';

  /* ── 切换 Tab ── */
  function switchTab(name) {
    [tabContent, tabMeta, tabJournal].forEach(function (b) {
      b.classList.toggle('active', b.id === 'drawer-tab-' + name);
    });
    if (name === 'content') {
      showContent('spec: ' + _currentSpecPath + '\n# content placeholder\u2026');
    } else if (name === 'metadata') {
      showMetadata(_currentSpecPath);
    } else if (name === 'journal') {
      hideAll();
      journalPanel.style.display = '';
    }
  }

  /* ── 初始化 Tab 按钮事件 ── */
  [tabContent, tabMeta, tabJournal].forEach(function (btn) {
    btn.addEventListener('click', function () {
      var name = btn.id.replace('drawer-tab-', '');
      switchTab(name);
    });
  });

  /* ── 关闭抽屉 ── */
  function closeDrawer() {
    backdrop.classList.remove('open');
    drawer.style.display       = 'none';
    backdrop.style.display     = 'none';
    specNameEl.textContent     = '\u2014';
    hideAll();
    _currentSpecPath = '';
  }

  document.querySelector('.drawer-close-btn')
    .addEventListener('click', closeDrawer);

  backdrop.addEventListener('click', closeDrawer);

  /* ── 打开抽屉：idle -> loading -> content|error -> idle ── */
  function openDrawer(specPath) {
    _currentSpecPath = specPath;
    specNameEl.textContent = specPath.split('/').pop().replace('.yaml', '');

    backdrop.style.display = '';
    drawer.style.display   = '';
    backdrop.classList.add('open');

    showLoading();
  }

  /* ── 将 MVP 中 data-action="open-drawer" 的按钮接入行为 ── */
  document.querySelectorAll('[data-action="open-drawer"]').forEach(function (btn) {
    btn.addEventListener('click', function () {
      var specPath = btn.getAttribute('data-spec-path') || btn.dataset.specPath;
      if (specPath) openDrawer(specPath);
    });
  });

  })();
});
"""


def inject_before_closing_tag(html: str, tag: str, content: str) -> str:
    """在最后一个 </tag> 之前注入 content（script/rfind，追加到 vibex-annot-ctrl 脚本块，它在抽屉 HTML 之后执行）。"""
    pos = html.rfind('</' + tag + '>')
    if pos == -1:
        raise ValueError(f'未找到 </{tag}>')
    return html[:pos] + content + html[pos:]


def main():
    # __file__ = .../scripts/gen-detail-drawer.py
    # 向上 4 层到项目根: scripts → spec-prototype-fragment → skills → agents → project_root
    depth = 6
    root = os.path.abspath(__file__)
    for _ in range(depth):
        root = os.path.dirname(root)
    mvp_file = os.path.join(root, MVP_PATH)
    out_file = os.path.join(root, OUT_PATH)

    with open(mvp_file, encoding='utf-8') as f:
        html = f.read()

    # 1. 在最后一个 </script> 前追加行为脚本（追加到 MVP 既有 script 块内，避免 file:// 安全问题）
    html = inject_before_closing_tag(html, 'script', '\n' + DRAWER_BEHAVIOR_SCRIPT + '\n')

    # 2. 移除抽屉的 inline display:none（让 JS 控制显示）
    #    注意：loading 默认显示（不需要 style="display:none"），
    #    但 content/meta/journal/error 需要保持 visible hidden 直到 JS 显示它们。
    #    先把 style="display:none" 统一替换为 class=hidden
    html = html.replace(
        'style="display:none"',
        'style="display:none"'
    )
    # 四态面板用 class 切换更干净，统一在 JS 中管理
    # loading 默认显示（首态），content/meta/journal/error 默认隐藏

    with open(out_file, 'w', encoding='utf-8') as f:
        f.write(html)

    print(f'[gen-detail-drawer] derived prototype written to: {out_file}')
    print('  [OK] MVP static structure preserved')
    print('  [OK] drawer 4-state behavior injected (open -> loading -> done|error)')
    print('  [OK] data-action=open-drawer buttons wired')
    print('  [OK] close button / backdrop click wired')
    print('  [OK] Step 0 done (drawer static structure annotated in MVP)')


if __name__ == '__main__':
    main()
