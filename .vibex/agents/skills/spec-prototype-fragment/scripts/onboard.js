/**
 * spec-prototype-fragment · onboard.js
 * ─────────────────────────────────────────────────────────────────────────────
 * 派生原型行为脚手架脚本。
 *
 * ⚠ 重要：派生原型必须从 vibex-workbench-mvp.html 出发。
 *   CSS 变量和样式必须从 MVP 复制，禁止自己发明颜色/字体/布局。
 *
 * MVP CSS 变量参考：
 *   --wb-bg-panel:   #0f1117   （面板背景）
 *   --wb-bg-panel-2: #151822    （次级面板）
 *   --wb-text:       #c0caf5   （主文字）
 *   --wb-text-sec:   #787c99   （次级文字）
 *   --wb-accent:     #72d6d0   （主题强调色）
 *   --accent-green:   #87cf8a   （成功绿）
 *   --accent-red:     #e16d75   （错误红）
 *   --accent-orange:   #f09a6a   （警告橙）
 *   --accent-yellow:   #efc66b   （黄色）
 *
 * 使用方式：
 *   1. 先从 MVP 中复制对应区域的 HTML + CSS 到派生原型
 *   2. 复制整个脚本到原型 HTML 的 <script> 标签内
 *   3. 调整 CONFIG（面板 ID、按钮 ID、行为函数）
 *   4. 重写 _syncOpen() / _syncPreview() / _syncLoading() / _syncDone() / _syncError()
 *
 * 本脚手架确保：
 *   ✓ 原型从 idle 初态开始，弹窗/遮罩初始隐藏
 *   ✓ 所有 panel 通过 class="hidden" 统一管理（不用 hidden 属性）
 *   ✓ done / error 面板正确展示（classList.remove('hidden')）
 *   ✓ 所有弹窗状态均有关闭路径回到 idle
 *   ✓ 所有按钮均有 addEventListener 绑定
 *
 * 四态设计规范（spec-prototype-fragment §0.3）：
 *   idle    — 初态：触发按钮等待点击，遮罩/弹窗隐藏
 *   active  — 边界态：用户已触发，显示弹窗/表单/进度（可分多个 phase）
 *   success — 结束态：操作成功，显示完成 UI，点「完成」回到 idle
 *   error   — 错误态：操作失败，显示错误提示，点「重试」或「取消」回到 idle
 *
 * 扩展四态的中间步骤（如本脚本的 open / preview / loading）：
 *   这些都是 active 的子步骤，遵循同样的展示/隐藏规则。
 * ─────────────────────────────────────────────────────────────────────────────
 */

/* ═══════════════════════════════════════════════════════════════════════════
   CONFIG — 在此处调整本原型的面板和按钮配置
   ═══════════════════════════════════════════════════════════════════════════ */

var CONFIG = {
  overlayId:    'overlay',    // 遮罩层 ID
  panels: {
    // phase 名 → panel DOM ID
    // 每个 phase 都需要一个对应的 panel
    // 注意：done 和 error 的 panel 必须存在，且 syncUI 中用 remove('hidden')
    open:    'ph-open',
    preview: 'ph-preview',
    loading: 'ph-loading',
    done:    'ph-done',
    error:   'ph-error',
  },
  buttons: {
    // 触发按钮（从 idle 进入 active 的第一步）
    trigger: 'btn-init',
    // 各 phase 的关闭路径
    // 格式：buttonId → 目标 phase
    // 这里列出所有 → idle 的关闭按钮
    closeToIdle: [
      'btn-cancel',       // open 阶段的「取消」
      'btn-cancel-open',  // open 阶段别名
      'btn-cancel-loading', // loading 阶段「取消」（中止操作）
      'btn-cancel-error', // error 阶段「取消」
      'btn-finish',       // done 阶段「完成」
      'btn-x',            // 弹窗「×」
    ],
    // 各 phase 之间的导航按钮
    navigate: {
      'btn-next':  'preview',  // open → preview
      'btn-back':  'open',     // preview → open
      'btn-gen':   'loading',  // preview → loading
      'btn-retry': 'open',     // error → open
    },
  },
  // 行为函数（可选）
  // onPhaseEnter(phase): 进入某 phase 时调用，返回 true 则继续默认行为
  // onPhaseExit(phase):  离开某 phase 时调用
  onPhaseEnter: null,
  onPhaseExit:  null,
  // loading 阶段的行为函数（返回 Promise 或调用 callback）
  // runLoading: function(doneCallback) { ... doneCallback(true); /* 或 false */ }
  runLoading: null,
  // loading 随机错误率（0-1），设为 0 则不模拟错误
  loadingErrorRate: 0.1,
};

/* ═══════════════════════════════════════════════════════════════════════════
   STATE MACHINE — 状态机核心（不建议修改）
   ═══════════════════════════════════════════════════════════════════════════ */

/** 全局当前 phase，初始为 idle（初态） */
var _phase = 'idle';

/**
 * 设置当前 phase（所有状态切换的唯一入口）
 * @param {string} p 目标 phase
 * @param {object} opts 可选参数 { skipExit, skipEnter }
 */
function setPhase(p, opts) {
  opts = opts || {};
  if (_phase === p) return; // 已在目标状态，不重复处理

  // onPhaseExit 钩子
  if (!opts.skipExit && CONFIG.onPhaseExit) {
    CONFIG.onPhaseExit(_phase);
  }

  var prev = _phase;
  _phase = p;

  // onPhaseEnter 钩子
  if (!opts.skipEnter && CONFIG.onPhaseEnter) {
    var proceed = CONFIG.onPhaseEnter(p);
    if (proceed === false) {
      _phase = prev; // 钩子拒绝，保持原状态
      return;
    }
  }

  syncUI();
}

/**
 * 获取当前 phase
 * @returns {string}
 */
function getPhase() {
  return _phase;
}

/* ═══════════════════════════════════════════════════════════════════════════
   syncUI — 状态 → UI 同步（按需重写各 phase 块）
   ═══════════════════════════════════════════════════════════════════════════ */

/**
 * syncUI 是状态机的核心渲染函数。
 * 规则：
 *   1. phase === 'idle' 时：隐藏 overlay，恢复初态 UI
 *   2. 其他 phase：显示 overlay，隐藏所有 panels，再展示当前 panel
 *   3. 每个非 idle 的 case 块：显示自己的 panel，用 classList.remove('hidden')
 *      隐藏其他 panels，用 classList.add('hidden')
 *   4. 永远不要在 done / error 阶段写 panels.xxx.hidden = true / add('hidden')
 *      —— 这两个阶段必须展示对应的成功/错误面板
 *   5. 永远不要在 idle 阶段直接展示 overlay
 */
function syncUI() {
  var overlay = document.getElementById(CONFIG.overlayId);

  /* ── idle：隐藏弹窗，恢复初态 ── */
  if (_phase === 'idle') {
    overlay.classList.add('hidden');
    return;
  }

  /* ── 显示遮罩 ── */
  overlay.classList.remove('hidden');

  /* 隐藏所有 panels */
  var panelIds = Object.keys(CONFIG.panels);
  panelIds.forEach(function(key) {
    var el = document.getElementById(CONFIG.panels[key]);
    if (el) el.classList.add('hidden');
  });

  /* ══════════════════════════════════════════════════════
     各 phase UI 渲染 — 重写这里以自定义你的 UI
     规则：显示当前 panel → classList.remove('hidden')
           隐藏其他 panel → classList.add('hidden')
  ══════════════════════════════════════════════════════ */

  /* ── open ── */
  if (_phase === 'open') {
    var el = document.getElementById(CONFIG.panels.open);
    if (el) el.classList.remove('hidden');
    // 重写这里：自定义 open 阶段的 UI
    _syncOpen();
  }
  /* ── preview ── */
  else if (_phase === 'preview') {
    var el = document.getElementById(CONFIG.panels.preview);
    if (el) el.classList.remove('hidden');
    // 重写这里：自定义 preview 阶段的 UI
    _syncPreview();
  }
  /* ── loading ── */
  else if (_phase === 'loading') {
    var el = document.getElementById(CONFIG.panels.loading);
    if (el) el.classList.remove('hidden');
    // 重写这里：自定义 loading 阶段的 UI
    _syncLoading();
  }
  /* ── done（成功）—— 展示成功面板，禁止隐藏 ── */
  else if (_phase === 'done') {
    var el = document.getElementById(CONFIG.panels.done);
    if (el) el.classList.remove('hidden'); // ✓ 必须展示
    // 重写这里：自定义 done 阶段的 UI
    _syncDone();
  }
  /* ── error（错误）—— 展示错误面板，禁止隐藏 ── */
  else if (_phase === 'error') {
    var el = document.getElementById(CONFIG.panels.error);
    if (el) el.classList.remove('hidden'); // ✓ 必须展示
    // 重写这里：自定义 error 阶段的 UI
    _syncError();
  }
}

/* ═══════════════════════════════════════════════════════════════════════════
   PHASE SYNC 钩子 — 每个 phase 的 UI 渲染逻辑
   复制到此处后，重写这些函数的内容即可
   ═══════════════════════════════════════════════════════════════════════════ */

/**
 * open 阶段同步
 * 触发：用户点击触发按钮
 * 退出：点「下一步」→「preview」；点「取消」/「×」/遮罩→「idle」
 */
function _syncOpen() {
  // 示例：设置 header 标题
  var hdr = document.getElementById('hdr-title');
  if (hdr) hdr.textContent = '初始化脚手架';

  // 示例：显示 open 阶段的按钮
  _showBtns(['btn-cancel-open', 'btn-next']);
  _hideBtns(['btn-back', 'btn-gen', 'btn-cancel-loading', 'btn-finish', 'btn-retry', 'btn-cancel-error']);
}

/**
 * preview 阶段同步
 * 触发：open → 点「下一步」
 * 退出：点「开始生成」→「loading」；点「← 上一步」→「open」；关闭→「idle」
 */
function _syncPreview() {
  var hdr = document.getElementById('hdr-title');
  if (hdr) hdr.textContent = '预览 → 确认';

  // 示例：刷新预览内容
  if (typeof refreshTree === 'function') refreshTree();

  _showBtns(['btn-back', 'btn-gen']);
  _hideBtns(['btn-cancel-open', 'btn-next', 'btn-cancel-loading', 'btn-finish', 'btn-retry', 'btn-cancel-error']);
}

/**
 * loading 阶段同步
 * 触发：preview → 点「开始生成」
 * 退出：完成（成功）→「done」；完成（失败）→「error」；点「取消」→「idle」
 */
function _syncLoading() {
  var hdr = document.getElementById('hdr-title');
  if (hdr) hdr.textContent = '正在生成…';

  _showBtns(['btn-cancel-loading']);
  _hideBtns(['btn-cancel-open', 'btn-next', 'btn-back', 'btn-gen', 'btn-finish', 'btn-retry', 'btn-cancel-error']);

  // 执行 loading 行为
  if (CONFIG.runLoading) {
    CONFIG.runLoading(function(success) {
      setPhase(success ? 'done' : 'error');
    });
  } else {
    // 默认：模拟带随机错误的行为
    _runDefaultLoading();
  }
}

/**
 * done 阶段同步
 * 触发：loading 完成（success）
 * 退出：点「完成」→「idle」
 */
function _syncDone() {
  var hdr = document.getElementById('hdr-title');
  if (hdr) hdr.textContent = '脚手架生成完成';

  // 隐藏 titlebar 的 loading 指示，显示完成指示
  var statePill = document.getElementById('state-pill');
  var donePill  = document.getElementById('done-pill');
  if (statePill) statePill.classList.add('hidden');
  if (donePill)  donePill.classList.remove('hidden');

  _showBtns(['btn-finish']);
  _hideBtns(['btn-cancel-open', 'btn-next', 'btn-back', 'btn-gen', 'btn-cancel-loading', 'btn-retry', 'btn-cancel-error']);
}

/**
 * error 阶段同步
 * 触发：loading 完成（failure）
 * 退出：点「重试」→「open」；点「取消」→「idle」
 */
function _syncError(msg) {
  var hdr = document.getElementById('hdr-title');
  if (hdr) hdr.textContent = '生成失败';

  var statePill = document.getElementById('state-pill');
  if (statePill) statePill.classList.add('hidden');

  // 显示错误信息
  var errMsg = document.getElementById('err-msg');
  if (errMsg && msg) errMsg.textContent = msg;

  _showBtns(['btn-retry', 'btn-cancel-error']);
  _hideBtns(['btn-cancel-open', 'btn-next', 'btn-back', 'btn-gen', 'btn-cancel-loading', 'btn-finish']);
}

/* ═══════════════════════════════════════════════════════════════════════════
   辅助函数
   ═══════════════════════════════════════════════════════════════════════════ */

/** 显示指定的按钮（移除 hidden 类） */
function _showBtns(ids) {
  ids.forEach(function(id) {
    var el = document.getElementById(id);
    if (el) el.classList.remove('hidden');
  });
}

/** 隐藏指定的按钮（添加 hidden 类） */
function _hideBtns(ids) {
  ids.forEach(function(id) {
    var el = document.getElementById(id);
    if (el) el.classList.add('hidden');
  });
}

/** 默认 loading 行为：带随机错误的进度模拟 */
function _runDefaultLoading() {
  var steps = [
    { lbl: '检查目标目录…',        cls: 'done'    },
    { lbl: '读取 spec-templates…',  cls: 'done'    },
    { lbl: '渲染骨架文件…',          cls: 'running' },
    { lbl: '写入磁盘…',              cls: 'pending' },
    { lbl: '生成 Makefile…',         cls: 'pending' },
    { lbl: '完成',                   cls: 'pending' },
  ];
  var list = document.getElementById('step-list');
  if (!list) return;

  function draw() {
    list.innerHTML = steps.map(function(s) {
      return '<div class="step ' + s.cls + '"><span class="dot ' + s.cls + '"></span>' + s.lbl + '</div>';
    }).join('');
  }
  draw();

  var i = 0;
  function tick() {
    if (i < steps.length) {
      steps[i].cls = 'done';
      if (i + 1 < steps.length) steps[i + 1].cls = 'running';
      i++;
      draw();
      if (i < steps.length - 1) setTimeout(tick, 550);
      else {
        setTimeout(function() {
          var isError = CONFIG.loadingErrorRate > 0 && Math.random() < CONFIG.loadingErrorRate;
          if (isError) {
            steps[steps.length - 1].cls = 'err';
            draw();
            var errEl = document.getElementById('err-msg');
            if (errEl) errEl.textContent = '写入目录失败：权限不足。请检查目录权限后重试。';
            setTimeout(function() { setPhase('error'); }, 300);
          } else {
            steps[steps.length - 1].cls = 'done';
            draw();
            setTimeout(function() { setPhase('done'); }, 300);
          }
        }, 400);
      }
    }
  }
  setTimeout(tick, 300);
}

/* ═══════════════════════════════════════════════════════════════════════════
   事件绑定（初始化时自动执行）
   ═══════════════════════════════════════════════════════════════════════════ */

/**
 * initBindings — 调用一次，绑定所有按钮事件
 * 在 DOMContentLoaded 或脚本位于 body 底部时调用
 */
function initBindings() {
  var overlay = document.getElementById(CONFIG.overlayId);

  /* ── 触发按钮：idle → open ── */
  var triggerBtn = document.getElementById(CONFIG.buttons.trigger);
  if (triggerBtn) {
    triggerBtn.addEventListener('click', function() {
      setPhase('open');
    });
  }

  /* ── 关闭到 idle 的按钮 ── */
  CONFIG.buttons.closeToIdle.forEach(function(id) {
    var btn = document.getElementById(id);
    if (btn) {
      btn.addEventListener('click', function() {
        setPhase('idle');
      });
    }
  });

  /* ── 导航按钮 ── */
  var nav = CONFIG.buttons.navigate;
  Object.keys(nav).forEach(function(btnId) {
    var btn = document.getElementById(btnId);
    if (btn) {
      btn.addEventListener('click', function() {
        setPhase(nav[btnId]);
      });
    }
  });

  /* ── 遮罩点击 → idle ── */
  if (overlay) {
    overlay.addEventListener('click', function(e) {
      if (e.target === e.currentTarget) setPhase('idle');
    });
  }
}

/* ── 自动初始化 ── */
(function() {
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', function() {
      initBindings();
      syncUI(); // 确保从 idle 开始
    });
  } else {
    initBindings();
    syncUI();
  }
})();
