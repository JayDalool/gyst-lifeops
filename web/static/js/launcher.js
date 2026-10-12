/* Launcher v2 (GYST-102): Go mode searches GYST, Capture mode adds a task.
   Go never creates anything. Results are real links built with
   createElement/textContent (no HTML strings). Every key handler except the
   Cmd/Ctrl-K opener lives on the dialog, so it is inert while the dialog is
   closed (the Galaxy page shares this layout and keeps Escape and the arrows). */
(function () {
  "use strict";
  var root = typeof window !== "undefined" ? window : globalThis;
  var RECENTS_KEY = "gyst-launcher-recents", RECENTS_MAX = 5, DEBOUNCE_MS = 120;

  // A link is accepted only as a rooted internal path, or (for external apps)
  // an http(s) URL with a host. Anything else (//host, javascript:, data:) is dropped.
  function safeHref(href, external) {
    if (typeof href !== "string" || href.length > 2000 || /[\u0000-\u001f\u007f\\\s]/.test(href)) return "";
    if (external) return /^https?:\/\/[^\/?#@:]+(:\d+)?([\/?#]\S*)?$/i.test(href) ? href : "";
    return href.charAt(0) === "/" && href.charAt(1) !== "/" ? href : "";
  }

  function cleanRecents(list) {
    var out = [], seen = {};
    if (!Array.isArray(list)) return out;
    for (var i = 0; i < list.length && out.length < RECENTS_MAX; i++) {
      var e = list[i];
      if (!e || typeof e.label !== "string" || !e.label.trim()) continue;
      var href = safeHref(e.href, false);
      if (!href || seen[href]) continue;
      seen[href] = true;
      out.push({ label: e.label.trim().slice(0, 100), href: href });
    }
    return out;
  }

  function addRecent(list, entry) {
    return cleanRecents([entry].concat(Array.isArray(list) ? list : []));
  }

  // '+' as the first character switches to Capture and is not part of the title.
  function parseInput(value, mode) {
    value = String(value == null ? "" : value);
    if (mode === "go" && value.charAt(0) === "+") return { mode: "capture", text: value.slice(1).replace(/^\s+/, "") };
    return { mode: mode, text: value };
  }

  // Enter opens the selected result in Go and never creates; it submits only from Capture.
  // While results are stale (the query changed and the answer has not arrived)
  // Enter never opens an old option: it asks for the current query and opens
  // that query's top hit when it arrives ("defer").
  function enterAction(mode, hasActive, title, pending) {
    if (mode === "capture") return String(title || "").trim() ? "submit" : "none";
    if (pending) return "defer";
    return hasActive ? "open" : "none";
  }

  function step(index, delta, count) {
    if (!count) return -1;
    return ((index < 0 ? (delta > 0 ? -1 : 0) : index) + delta + count) % count;
  }

  // The tomorrow/today and !high/!low chips the old quick-task box understood.
  function parseCapture(text, today, tomorrow) {
    var title = String(text || ""), due = "", priority = "";
    if (/\btomorrow\b/i.test(title)) { due = tomorrow; title = title.replace(/\btomorrow\b/ig, "").trim(); }
    else if (/\btoday\b/i.test(title)) { due = today; title = title.replace(/\btoday\b/ig, "").trim(); }
    if (/!high\b/i.test(title)) { priority = "high"; title = title.replace(/!high\b/ig, "").trim(); }
    if (/!low\b/i.test(title)) { priority = "low"; title = title.replace(/!low\b/ig, "").trim(); }
    return { title: title, due: due, priority: priority };
  }

  root.GystLauncher = { safeHref: safeHref, cleanRecents: cleanRecents, addRecent: addRecent, parseInput: parseInput, enterAction: enterAction, step: step, parseCapture: parseCapture, RECENTS_MAX: RECENTS_MAX };

  if (typeof document === "undefined") return;

  function init() {
    var palette = document.querySelector("[data-command-palette]");
    if (!palette) return;
    var input = palette.querySelector("[data-command-search]");
    var results = palette.querySelector("[data-command-results]");
    var form = palette.querySelector("[data-quick-task]");
    var title = palette.querySelector("[data-quick-title]");
    var date = palette.querySelector("[data-quick-date]");
    var priority = palette.querySelector("[data-quick-priority]");
    var returnTo = palette.querySelector("[data-command-return]");
    var status = palette.querySelector("[data-launcher-status]");
    var closeButton = palette.querySelector("[data-command-close]");
    if (!input || !results || !form || !title || !date || !priority) return;
    var modeButtons = Array.prototype.slice.call(palette.querySelectorAll("[data-launcher-mode]"));
    var panels = Array.prototype.slice.call(palette.querySelectorAll("[data-launcher-panel]"));
    var HELP = status ? status.textContent : "";

    var mode = "go", options = [], active = -1, timer = 0, controller = null, seq = 0, trigger = null, pending = false, openAfter = false;

    // Today and tomorrow are the server's local days in the app timezone
    // (data-today / data-tomorrow); the browser date is only a fallback.
    function isoDate(offset) {
      var server = offset === 1 ? palette.dataset.tomorrow : palette.dataset.today;
      if (offset >= 0 && offset <= 1 && server) return server;
      var value = new Date();
      value.setDate(value.getDate() + offset);
      return value.getFullYear() + "-" + String(value.getMonth() + 1).padStart(2, "0") + "-" + String(value.getDate()).padStart(2, "0");
    }

    function readRecents() {
      try { return cleanRecents(JSON.parse(localStorage.getItem(RECENTS_KEY) || "[]")); } catch (e) { return []; }
    }
    function saveRecent(label, href) {
      try { localStorage.setItem(RECENTS_KEY, JSON.stringify(addRecent(readRecents(), { label: label, href: href }))); } catch (e) {}
    }
    function setStatus(text) { if (status) status.textContent = text; }

    function setActive(next) {
      if (options[active]) { options[active].classList.remove("active"); options[active].setAttribute("aria-selected", "false"); }
      active = next;
      var el = options[active];
      if (el) {
        el.classList.add("active");
        el.setAttribute("aria-selected", "true");
        input.setAttribute("aria-activedescendant", el.id);
        if (el.scrollIntoView) el.scrollIntoView({ block: "nearest" });
      } else {
        input.setAttribute("aria-activedescendant", "");
      }
    }

    function clearResults() {
      while (results.firstChild) results.removeChild(results.firstChild);
      options = [];
      active = -1;
      input.setAttribute("aria-activedescendant", "");
    }

    function addGroup(label, items, remember) {
      if (!items.length) return;
      var group = document.createElement("div");
      group.setAttribute("role", "group");
      group.className = "launcher-group";
      var heading = document.createElement("div");
      heading.className = "launcher-group-label";
      heading.setAttribute("aria-hidden", "true");
      heading.textContent = label;
      group.setAttribute("aria-label", label);
      group.appendChild(heading);
      items.forEach(function (item) {
        var a = document.createElement("a");
        a.id = "launcher-option-" + options.length;
        a.className = "launcher-item";
        a.setAttribute("role", "option");
        a.setAttribute("aria-selected", "false");
        a.setAttribute("tabindex", "-1");
        a.href = item.href;
        if (item.external) { a.target = "_blank"; a.rel = "noopener noreferrer"; }
        a.dataset.recent = remember && !item.external ? "1" : "";
        var name = document.createElement("span");
        name.className = "launcher-item-label";
        name.textContent = item.label;
        a.appendChild(name);
        if (item.meta) {
          var meta = document.createElement("span");
          meta.className = "launcher-item-meta";
          meta.textContent = item.meta;
          a.appendChild(meta);
        }
        group.appendChild(a);
        options.push(a);
      });
      results.appendChild(group);
    }

    function cleanGroups(data) {
      var out = [];
      if (!data || !Array.isArray(data.groups)) return out;
      data.groups.forEach(function (g) {
        if (!g || typeof g.label !== "string" || !Array.isArray(g.items)) return;
        var items = [];
        g.items.slice(0, 6).forEach(function (it) {
          if (!it || typeof it.label !== "string") return;
          var external = it.external === true;
          var href = safeHref(it.href, external);
          if (href) items.push({ label: it.label, meta: typeof it.meta === "string" ? it.meta : "", href: href, external: external });
        });
        if (items.length) out.push({ label: g.label, items: items });
      });
      return out;
    }

    function render(groups, query) {
      clearResults();
      if (!query) {
        var recents = readRecents();
        addGroup("Recents", recents.map(function (r) { return { label: r.label, meta: "", href: r.href, external: false }; }), true);
      }
      groups.forEach(function (g) { addGroup(g.label, g.items, true); });
      input.setAttribute("aria-expanded", options.length ? "true" : "false");
      if (options.length) {
        setActive(0);
        setStatus(HELP);
      } else {
        setStatus(query ? "No matches for “" + query + "”. Start with + to add a task." : HELP);
      }
    }

    function search(query) {
      if (controller) controller.abort();
      controller = typeof AbortController !== "undefined" ? new AbortController() : null;
      var mine = ++seq;
      pending = true;
      var init = { credentials: "same-origin", headers: { Accept: "application/json" } };
      if (controller) init.signal = controller.signal;
      fetch("/api/search?q=" + encodeURIComponent(query), init).then(function (response) {
        if (response.status === 401) { var auth = new Error("auth"); auth.auth = true; throw auth; }
        if (!response.ok) throw new Error("search " + response.status);
        return response.json();
      }).then(function (data) {
        if (mine !== seq || mode !== "go") return;
        pending = false;
        render(cleanGroups(data), query);
        if (openAfter) { openAfter = false; if (options[0]) options[0].click(); }
      }).catch(function (err) {
        if (mine !== seq || mode !== "go" || (err && err.name === "AbortError")) return;
        pending = false; openAfter = false;
        render([], query);
        setStatus(err && err.auth ? "Your session ended. Reload the page to sign in." : "Search is unavailable right now.");
      });
    }

    function runSearch(immediate) {
      clearTimeout(timer);
      var query = input.value.trim();
      if (immediate || !query) search(query);
      else timer = setTimeout(function () { search(query); }, DEBOUNCE_MS);
    }

    function applyCapture() {
      var parsed = parseCapture(input.value, isoDate(0), isoDate(1));
      title.value = parsed.title;
      if (parsed.due) date.value = parsed.due;
      if (parsed.priority) priority.value = parsed.priority;
    }

    function setMode(next, focus) {
      if (next === mode) { if (focus) input.focus(); return; }
      mode = next;
      pending = false; openAfter = false;
      modeButtons.forEach(function (b) { b.setAttribute("aria-pressed", b.dataset.launcherMode === mode ? "true" : "false"); });
      panels.forEach(function (p) { p.hidden = p.dataset.launcherPanel !== mode; });
      if (mode === "capture") {
        clearTimeout(timer);
        seq++;
        if (controller) controller.abort();
        input.setAttribute("aria-expanded", "false");
        input.removeAttribute("aria-controls");
        input.setAttribute("aria-label", "Task title");
        input.placeholder = "What needs doing?";
        applyCapture();
      } else {
        input.setAttribute("aria-controls", "launcher-results");
        input.setAttribute("aria-label", "Search GYST");
        input.placeholder = "Search GYST, or + to add a task";
        runSearch(true);
      }
      if (focus) input.focus();
    }

    function open(event) {
      trigger = event && event.currentTarget && event.currentTarget.focus ? event.currentTarget : document.activeElement;
      returnTo.value = location.pathname + location.search + location.hash;
      date.value = isoDate(0);
      priority.value = "medium";
      if (!palette.open) palette.showModal();
      mode = "capture"; // force the reset below
      pending = false; openAfter = false;
      input.value = ""; title.value = "";
      setMode("go", false);
      input.focus();
    }
    function close() { if (palette.open) palette.close(); }

    palette.addEventListener("close", function () {
      pending = false; openAfter = false;
      clearTimeout(timer);
      seq++;
      if (controller) controller.abort();
      if (trigger && trigger.isConnected && trigger.focus) trigger.focus();
    });
    document.querySelectorAll("[data-command-open]").forEach(function (button) { button.addEventListener("click", open); });
    if (closeButton) closeButton.addEventListener("click", close);
    modeButtons.forEach(function (b) { b.addEventListener("click", function () { setMode(b.dataset.launcherMode, true); }); });

    input.addEventListener("input", function () {
      var parsed = parseInput(input.value, mode);
      if (parsed.mode !== mode) { input.value = parsed.text; setMode(parsed.mode, false); }
      if (mode === "capture") applyCapture();
      else { setActive(-1); pending = true; openAfter = false; runSearch(false); }
    });

    input.addEventListener("keydown", function (event) {
      if (event.isComposing) return;
      if (event.key === "Tab" && !event.shiftKey && !event.ctrlKey && !event.metaKey && !event.altKey) {
        event.preventDefault();
        setMode(mode === "go" ? "capture" : "go", true);
      } else if (mode === "go" && !pending && event.key === "ArrowDown") { event.preventDefault(); setActive(step(active, 1, options.length)); }
      else if (mode === "go" && !pending && event.key === "ArrowUp") { event.preventDefault(); setActive(step(active, -1, options.length)); }
      else if (event.key === "Enter") {
        var action = enterAction(mode, !!options[active], title.value, pending);
        event.preventDefault();
        if (action === "open") options[active].click();
        else if (action === "defer") { if (input.value.trim()) { openAfter = true; runSearch(true); } }
        else if (action === "submit") form.requestSubmit();
      }
    });

    // Escape and result clicks live on the dialog, so nothing runs while it is closed.
    palette.addEventListener("keydown", function (event) {
      if (event.key === "Escape") { event.preventDefault(); close(); }
    });
    results.addEventListener("click", function (event) {
      var a = event.target && event.target.closest ? event.target.closest("a.launcher-item") : null;
      if (!a) return;
      if (a.dataset.recent === "1") {
        var label = a.querySelector(".launcher-item-label");
        saveRecent(label ? label.textContent : a.textContent, a.getAttribute("href"));
      }
      var next = options.indexOf(a);
      if (next >= 0) setActive(next);
    });
    palette.addEventListener("click", function (event) { if (event.target === palette) close(); });
    document.addEventListener("keydown", function (event) {
      if ((event.ctrlKey || event.metaKey) && event.key && event.key.toLowerCase() === "k") {
        event.preventDefault();
        if (palette.open) close(); else open();
      }
    });
  }

  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init);
  else init();
})();
