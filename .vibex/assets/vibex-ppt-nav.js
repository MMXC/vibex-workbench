/* vibex-ppt-nav.js
 * Click left half => previous slide
 * Click right half => next slide
 * Ignores clicks on interactive controls/links/forms.
 */
(function () {
  if (typeof window === "undefined" || typeof document === "undefined") return;

  function dispatchArrow(key) {
    const ev = new KeyboardEvent("keydown", { key: key, bubbles: true });
    document.dispatchEvent(ev);
    window.dispatchEvent(ev);
  }

  function isInteractive(el) {
    if (!el || !(el instanceof Element)) return false;
    return !!el.closest(
      'a, button, input, textarea, select, label, summary, details, [contenteditable=""], [contenteditable="true"], [data-no-slide-nav], .no-slide-nav'
    );
  }

  var hasDeckRuntime = !!document.querySelector(".deck");
  var fallbackSlides = [];
  var fallbackIdx = 0;

  function fallbackInit() {
    fallbackSlides = Array.prototype.slice.call(document.querySelectorAll(".slide"));
    if (!fallbackSlides.length) return;
    var found = fallbackSlides.findIndex(function (s) { return s.classList.contains("is-active"); });
    fallbackIdx = found >= 0 ? found : 0;
    fallbackRender();
  }

  function fallbackRender() {
    if (!fallbackSlides.length) return;
    for (var i = 0; i < fallbackSlides.length; i++) {
      var active = i === fallbackIdx;
      fallbackSlides[i].classList.toggle("is-active", active);
      fallbackSlides[i].style.display = active ? "" : "none";
    }
    var num = document.querySelector(".slide-number");
    if (num) num.textContent = (fallbackIdx + 1) + " / " + fallbackSlides.length;
  }

  function fallbackGo(delta) {
    if (!fallbackSlides.length) return;
    var n = fallbackIdx + delta;
    if (n < 0) n = 0;
    if (n >= fallbackSlides.length) n = fallbackSlides.length - 1;
    if (n === fallbackIdx) return;
    fallbackIdx = n;
    fallbackRender();
  }

  function onClick(e) {
    if (e.defaultPrevented) return;
    if (isInteractive(e.target)) return;

    const w = Math.max(window.innerWidth || 0, document.documentElement.clientWidth || 0);
    if (!w) return;
    const x = typeof e.clientX === "number" ? e.clientX : 0;
    if (hasDeckRuntime) {
      if (x < w / 2) {
        dispatchArrow("ArrowLeft");
      } else {
        dispatchArrow("ArrowRight");
      }
      return;
    }
    if (x < w / 2) {
      fallbackGo(-1);
    } else {
      fallbackGo(1);
    }
  }

  function onKey(e) {
    if (hasDeckRuntime) return;
    if (e.metaKey || e.ctrlKey || e.altKey) return;
    if (e.key === "ArrowLeft" || e.key === "PageUp") {
      fallbackGo(-1);
      e.preventDefault();
    } else if (e.key === "ArrowRight" || e.key === "PageDown" || e.key === " ") {
      fallbackGo(1);
      e.preventDefault();
    }
  }

  if (!hasDeckRuntime) fallbackInit();
  document.addEventListener("click", onClick, true);
  document.addEventListener("keydown", onKey, true);
})();
