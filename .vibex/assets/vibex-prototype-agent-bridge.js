/**
 * VibeX prototype ↔ Workbench / Agent 桥接（嵌入 prototype HTML，与页面同源）。
 * 在原型页 head 末尾加入 script 标签，src 指向 .vibex/assets/vibex-prototype-agent-bridge.js
 * Agent 可调用 window.vibexPrototypeBridge；Workbench 抽屉通过 postMessage 驱动高亮 / onboard。
 */
(function () {
	'use strict';
	var PREFIX = 'vibex-prototype-bridge';
	var VER = 1;

	function vw() {
		return window.innerWidth || document.documentElement.clientWidth || 1;
	}
	function vh() {
		return window.innerHeight || document.documentElement.clientHeight || 1;
	}

	function postToParent(msg) {
		try {
			if (window.parent && window.parent !== window) {
				window.parent.postMessage(
					Object.assign({ source: PREFIX, version: VER }, msg),
					'*'
				);
			}
		} catch (e) {
			/* ignore */
		}
	}

	function measure(sel) {
		if (!sel || typeof sel !== 'string') return null;
		var el = document.querySelector(sel);
		if (!el) return null;
		var r = el.getBoundingClientRect();
		return { sel: sel, top: r.top, left: r.left, width: r.width, height: r.height };
	}

	function highlightSelectors(selectors) {
		var rects = [];
		(selectors || []).forEach(function (sel) {
			var m = measure(sel);
			if (m) rects.push(m);
		});
		postToParent({
			kind: 'highlight-rects',
			rects: rects,
			viewport: { width: vw(), height: vh() },
			onboard: null,
		});
	}

	function runOnboard(steps) {
		steps = steps || [];
		var i = 0;
		function clearTimer() {
			if (window.__vibexOnboardTimer) {
				clearTimeout(window.__vibexOnboardTimer);
				window.__vibexOnboardTimer = 0;
			}
		}
		function step() {
			if (i >= steps.length) {
				clearTimer();
				postToParent({ kind: 'onboard-end', rects: [], viewport: { width: vw(), height: vh() }, onboard: null });
				return;
			}
			var s = steps[i] || {};
			var rects = [];
			if (s.target) {
				var m = measure(s.target);
				if (m) rects.push(m);
			}
			postToParent({
				kind: 'onboard-step',
				rects: rects,
				viewport: { width: vw(), height: vh() },
				onboard: {
					title: String(s.title || ''),
					body: s.body != null ? String(s.body) : '',
					step: i + 1,
					total: steps.length,
				},
			});
			i += 1;
			var delay = typeof s.ms === 'number' ? s.ms : 3400;
			clearTimer();
			window.__vibexOnboardTimer = window.setTimeout(step, delay);
		}
		clearTimer();
		step();
	}

	function clearAll() {
		if (window.__vibexOnboardTimer) {
			clearTimeout(window.__vibexOnboardTimer);
			window.__vibexOnboardTimer = 0;
		}
		postToParent({ kind: 'clear', rects: [], viewport: { width: vw(), height: vh() }, onboard: null });
	}

	window.vibexPrototypeBridge = {
		highlight: highlightSelectors,
		onboard: runOnboard,
		clear: clearAll,
	};

	window.addEventListener('message', function (ev) {
		var d = ev.data;
		if (!d || d.source !== 'vibex-workbench' || d.version !== VER) return;
		if (d.kind === 'highlight') highlightSelectors(d.selectors || []);
		else if (d.kind === 'onboard') runOnboard(d.steps || []);
		else if (d.kind === 'clear') clearAll();
	});

	postToParent({ kind: 'ready', rects: [], viewport: { width: vw(), height: vh() }, onboard: null });
})();
