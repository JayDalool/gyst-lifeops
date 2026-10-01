// Completion toast (#25). After a completion POST the server redirects back
// with celebrate_* query params; this script turns them into a small toast
// ("Done · title · +N XP · Undo") and, when a level was crossed, a short
// level-up card. Every param is untrusted: text is only ever set through
// textContent, and the Undo target must match a strict same-origin allowlist.
(function () {
  "use strict";

  var KEYS = ["celebrate", "celebrate_title", "celebrate_xp", "celebrate_level", "celebrate_undo", "celebrate_date"];
  var UNDO_RE = /^\/(tasks\/\d+|projects\/\d+\/items\/\d+|courses\/\d+\/blocks\/\d+)\/undo$/;
  var DATE_RE = /^\d{4}-\d{2}-\d{2}$/;
  var TOAST_MS = 5000;
  var LEVELUP_MS = 1500;
  var MAX_TITLE = 60;

  function parse(search) {
    var params = new URLSearchParams(search || "");
    var kind = params.get("celebrate");
    if (kind !== "task" && kind !== "project") return null;
    var title = (params.get("celebrate_title") || "").trim();
    var points = Array.from(title);
    if (points.length > MAX_TITLE) title = points.slice(0, MAX_TITLE).join("") + "…";
    if (!title) title = kind === "project" ? "Project complete" : "Mission complete";
    var xp = parseInt(params.get("celebrate_xp"), 10);
    if (!(xp > 0 && xp <= 1000000)) xp = 0;
    var level = parseInt(params.get("celebrate_level"), 10);
    if (!(level >= 2 && level <= 1000000)) level = 0;
    var undo = params.get("celebrate_undo") || "";
    if (!UNDO_RE.test(undo)) undo = "";
    var date = params.get("celebrate_date") || "";
    if (!DATE_RE.test(date)) date = "";
    return { kind: kind, title: title, xp: xp, level: level, undo: undo, date: date };
  }

  // The current URL without the celebrate_* params (path, other params, hash).
  function cleanUrl(loc) {
    var params = new URLSearchParams(loc.search || "");
    KEYS.forEach(function (key) { params.delete(key); });
    var query = params.toString();
    return loc.pathname + (query ? "?" + query : "") + (loc.hash || "");
  }

  function el(doc, tag, className, text) {
    var node = doc.createElement(tag);
    if (className) node.className = className;
    if (text !== undefined) node.textContent = text;
    return node;
  }

  function hidden(doc, name, value) {
    var input = doc.createElement("input");
    input.type = "hidden";
    input.name = name;
    input.value = value;
    return input;
  }

  function build(doc, data, returnTo) {
    var toast = el(doc, "div", "lifeops-toast completion-toast");
    toast.setAttribute("role", "status");
    toast.setAttribute("aria-live", "polite");
    toast.appendChild(el(doc, "span", "completion-toast-title", "Done · " + data.title));
    if (data.xp > 0) toast.appendChild(el(doc, "span", "completion-toast-xp", "+" + data.xp + " XP"));
    if (data.undo) {
      var form = doc.createElement("form");
      form.method = "post";
      form.setAttribute("method", "post");
      form.setAttribute("action", data.undo);
      if (data.date) form.appendChild(hidden(doc, "occurrence_date", data.date));
      form.appendChild(hidden(doc, "return_to", returnTo));
      var button = el(doc, "button", "", "Undo");
      button.type = "submit";
      form.appendChild(button);
      form.addEventListener("submit", function () {
        // Disable after the submit has been dispatched so a double tap cannot post twice.
        setTimeout(function () { button.disabled = true; }, 0);
      });
      toast.appendChild(form);
    }
    return toast;
  }

  function buildLevelUp(doc, data) {
    var overlay = el(doc, "div", "levelup-event");
    overlay.setAttribute("role", "status");
    var card = el(doc, "div", "levelup-card");
    card.appendChild(el(doc, "span", "levelup-caption", "Level up"));
    card.appendChild(el(doc, "strong", "levelup-level", "Level " + data.level));
    if (data.xp > 0) card.appendChild(el(doc, "span", "levelup-xp", "+" + data.xp + " XP"));
    overlay.appendChild(card);
    return overlay;
  }

  function baseMs() {
    try {
      var v = getComputedStyle(document.documentElement).getPropertyValue("--vn-base").trim();
      var n = parseFloat(v);
      if (isNaN(n)) return 240;
      return /ms$/.test(v) ? n : n * 1000;
    } catch (_) { return 240; }
  }

  function showToast(data, returnTo) {
    var toast = build(document, data, returnTo);
    var timer = null;
    function remove() { if (toast.parentNode) toast.parentNode.removeChild(toast); }
    function hide() {
      toast.classList.remove("is-visible");
      setTimeout(remove, baseMs());
    }
    function start() { clearTimeout(timer); timer = setTimeout(hide, TOAST_MS); }
    function pause() { clearTimeout(timer); }
    toast.addEventListener("pointerenter", pause);
    toast.addEventListener("focusin", pause);
    toast.addEventListener("pointerleave", start);
    toast.addEventListener("focusout", start);
    document.body.appendChild(toast);
    requestAnimationFrame(function () { toast.classList.add("is-visible"); });
    start();
  }

  function showLevelUp(data) {
    var overlay = buildLevelUp(document, data);
    var fadeTimer = null, removeTimer = null, done = false;
    function remove() {
      if (done) return;
      done = true;
      clearTimeout(fadeTimer); clearTimeout(removeTimer);
      ["pointerdown", "click", "keydown"].forEach(function (name) { document.removeEventListener(name, dismiss, true); });
      if (overlay.parentNode) overlay.parentNode.removeChild(overlay);
    }
    function dismiss() {
      if (done) return;
      clearTimeout(fadeTimer);
      overlay.classList.remove("is-visible");
      removeTimer = setTimeout(remove, baseMs());
    }
    ["pointerdown", "click", "keydown"].forEach(function (name) { document.addEventListener(name, dismiss, true); });
    document.body.appendChild(overlay);
    requestAnimationFrame(function () { overlay.classList.add("is-visible"); });
    // Focus is neither moved nor trapped. Total lifetime is LEVELUP_MS.
    fadeTimer = setTimeout(dismiss, Math.max(0, LEVELUP_MS - baseMs()));
    removeTimer = setTimeout(remove, LEVELUP_MS);
  }

  function run() {
    var data = parse(location.search);
    if (!data) return;
    var returnTo = cleanUrl(location);
    try { history.replaceState(null, "", returnTo); } catch (_) {}
    showToast(data, returnTo);
    if (data.level > 0) showLevelUp(data);
  }

  window.GystCompletionToast = { parse: parse, build: build, buildLevelUp: buildLevelUp, cleanUrl: cleanUrl };

  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", run);
  else run();
})();
