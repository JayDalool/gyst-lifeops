/* Launcher v2 (GYST-102, GYST-103): Go mode searches GYST, Do mode runs a fixed
   list of commands, Capture mode adds a task. Go never creates anything. Do
   mode only links, applies the appearance choice, or shows a confirm step that
   holds a real <form method=post> to an allowlisted action; nothing posts
   before Confirm and the client never writes with fetch. Results are real links built with
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

  // Do mode: the only writes the launcher can ever offer. Anything else the
  // server sends is dropped, so a hostile response cannot aim a form elsewhere.
  var POST_ACTION = /^\/tasks\/\d+\/complete$|^\/health\/water$|^\/health\/training\/sync$/;
  var DATE_FIELD = /^\d{4}-\d{2}-\d{2}$/;
  var MODES = ["go", "do", "capture"];

  function safePostAction(action) {
    return typeof action === "string" && action.length < 80 && POST_ACTION.test(action) ? action : "";
  }

  // Hidden fields: only occurrence_date, only YYYY-MM-DD.
  function cleanFields(fields) {
    var out = {};
    if (fields && typeof fields === "object" && !Array.isArray(fields) && typeof fields.occurrence_date === "string" && DATE_FIELD.test(fields.occurrence_date)) {
      out.occurrence_date = fields.occurrence_date;
    }
    return out;
  }

  function text(value, max) { return typeof value === "string" ? value.slice(0, max) : ""; }

  // Returns the commands the dialog may show: links with a safe internal href,
  // posts with an allowlisted action (or disabled, with a reason), nothing else.
  function cleanCommands(data) {
    var out = [];
    if (!data || !Array.isArray(data.commands)) return out;
    data.commands.slice(0, 20).forEach(function (c) {
      if (!c || typeof c.label !== "string" || !c.label.trim()) return;
      var cmd = { label: text(c.label, 120), meta: text(c.meta, 60), kind: "", href: "", action: "", fields: {}, confirm: text(c.confirm, 200), disabledReason: text(c.disabledReason, 120) };
      if (cmd.disabledReason) { cmd.kind = "disabled"; out.push(cmd); return; }
      if (c.kind === "link") {
        cmd.href = safeHref(c.href, false);
        if (!cmd.href) return;
        cmd.kind = "link";
      } else if (c.kind === "post") {
        cmd.action = safePostAction(c.action);
        if (!cmd.action) return;
        cmd.fields = cleanFields(c.fields);
        cmd.kind = "post";
      } else return;
      out.push(cmd);
    });
    return out;
  }

  // The confirm step: a real form, built with createElement only, or null for an action off the allowlist.
  function buildConfirmForm(doc, cmd) {
    var action = cmd && cmd.kind === "post" ? safePostAction(cmd.action) : "";
    if (!action) return null;
    var form = doc.createElement("form");
    form.setAttribute("method", "post");
    form.setAttribute("action", action);
    form.className = "launcher-confirm-form";
    var fields = cleanFields(cmd.fields);
    Object.keys(fields).forEach(function (name) {
      var hidden = doc.createElement("input");
      hidden.setAttribute("type", "hidden");
      hidden.setAttribute("name", name);
      hidden.setAttribute("value", fields[name]);
      form.appendChild(hidden);
    });
    var message = doc.createElement("p");
    message.className = "launcher-confirm-text";
    message.textContent = text(cmd.confirm, 200) || text(cmd.label, 120);
    form.appendChild(message);
    var row = doc.createElement("div");
    row.className = "launcher-confirm-actions";
    var confirm = doc.createElement("button");
    confirm.setAttribute("type", "submit");
    confirm.className = "btn";
    confirm.dataset.doConfirmSubmit = "1";
    confirm.textContent = "Confirm";
    var cancel = doc.createElement("button");
    cancel.setAttribute("type", "button");
    cancel.className = "btn quiet";
    cancel.dataset.doCancel = "1";
    cancel.textContent = "Cancel";
    row.appendChild(confirm);
    row.appendChild(cancel);
    form.appendChild(row);
    return form;
  }

  function nextMode(mode) { return MODES[(MODES.indexOf(mode) + 1) % MODES.length]; }

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

  // '+' as the first character switches to Capture and is not part of the title;
  // '>' switches to Do.
  function parseInput(value, mode) {
    value = String(value == null ? "" : value);
    if (mode === "go" && value.charAt(0) === ">") return { mode: "do", text: value.slice(1).replace(/^\s+/, "") };
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

  root.GystLauncher = { safeHref: safeHref, cleanRecents: cleanRecents, addRecent: addRecent, parseInput: parseInput, enterAction: enterAction, step: step, parseCapture: parseCapture, safePostAction: safePostAction, cleanFields: cleanFields, cleanCommands: cleanCommands, buildConfirmForm: buildConfirmForm, nextMode: nextMode, RECENTS_MAX: RECENTS_MAX };

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
    var appearance = palette.querySelector("[data-do-appearance]");
    var confirmBox = palette.querySelector("[data-do-confirm]");
    if (!input || !results || !form || !title || !date || !priority) return;
    var modeButtons = Array.prototype.slice.call(palette.querySelectorAll("[data-launcher-mode]"));
    var panels = Array.prototype.slice.call(palette.querySelectorAll("[data-launcher-panel]"));
    var HELP = status ? status.textContent : "";

    var mode = "go", options = [], active = -1, timer = 0, controller = null, seq = 0, trigger = null, pending = false, openAfter = false, confirming = false, appearanceShown = false;
    var APPEARANCE_TERMS = "appearance theme system light dark";

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

    function appearanceMatches(query) {
      var hay = APPEARANCE_TERMS;
      return query.toLowerCase().split(/\s+/).filter(Boolean).every(function (w) { return hay.indexOf(w) >= 0; });
    }

    function addCommands(cmds) {
      if (!cmds.length) return;
      var group = document.createElement("div");
      group.setAttribute("role", "group");
      group.className = "launcher-group";
      group.setAttribute("aria-label", "Commands");
      var heading = document.createElement("div");
      heading.className = "launcher-group-label";
      heading.setAttribute("aria-hidden", "true");
      heading.textContent = "Commands";
      group.appendChild(heading);
      cmds.forEach(function (cmd) {
        var item;
        if (cmd.kind === "link") {
          item = document.createElement("a");
          item.href = cmd.href;
        } else {
          item = document.createElement("div");
          if (cmd.kind === "post") item.dataset.doPost = "1";
          else item.setAttribute("aria-disabled", "true");
        }
        item.id = "launcher-option-" + options.length;
        item.className = "launcher-item";
        item.setAttribute("role", "option");
        item.setAttribute("aria-selected", "false");
        item.setAttribute("tabindex", "-1");
        item.dataset.recent = "";
        item.gystCommand = cmd;
        var name = document.createElement("span");
        name.className = "launcher-item-label";
        name.textContent = cmd.label;
        item.appendChild(name);
        var metaText = cmd.kind === "disabled" ? cmd.disabledReason : cmd.meta;
        if (metaText) {
          var meta = document.createElement("span");
          meta.className = "launcher-item-meta";
          meta.textContent = metaText;
          item.appendChild(meta);
        }
        group.appendChild(item);
        options.push(item);
      });
      results.appendChild(group);
    }

    function showAppearance(show) {
      appearanceShown = !!show;
      if (!appearance) return;
      appearance.hidden = !show;
      if (!show) return;
      // The three theme buttons stay where the layout put them (theme.js owns
      // them); they join the arrow-key list so the keyboard can reach them.
      Array.prototype.forEach.call(appearance.querySelectorAll ? appearance.querySelectorAll("[data-theme-choice]") : [], function (b) {
        b.id = "launcher-appearance-" + options.length;
        options.push(b);
      });
    }

    function renderDo(cmds, query) {
      clearResults();
      addCommands(cmds);
      showAppearance(!query || appearanceMatches(query));
      input.setAttribute("aria-expanded", options.length ? "true" : "false");
      if (options.length) {
        setActive(0);
        setStatus(HELP);
      } else {
        setStatus("No commands match “" + query + "”.");
      }
    }

    // Confirm step: swaps the list for a real form. Nothing is sent until Confirm is activated.
    function askConfirm(cmd) {
      var form = buildConfirmForm(document, cmd);
      if (!form || !confirmBox) { setStatus("That command is not available."); return; }
      while (confirmBox.firstChild) confirmBox.removeChild(confirmBox.firstChild);
      confirmBox.appendChild(form);
      confirming = true;
      results.hidden = true;
      if (appearance) appearance.hidden = true;
      confirmBox.hidden = false;
      form.addEventListener("submit", function () {
        // Block a double tap; the page navigates away with the response.
        setTimeout(function () { var b = form.querySelector ? form.querySelector("[data-do-confirm-submit]") : null; if (b) b.disabled = true; }, 0);
      });
      var go = form.querySelector ? form.querySelector("[data-do-confirm-submit]") : null;
      if (go && go.focus) go.focus();
    }

    function endConfirm(focusInput) {
      if (!confirming) return;
      confirming = false;
      if (confirmBox) {
        while (confirmBox.firstChild) confirmBox.removeChild(confirmBox.firstChild);
        confirmBox.hidden = true;
      }
      results.hidden = false;
      if (appearance && mode === "do") appearance.hidden = !appearanceShown;
      if (focusInput) input.focus();
    }

    function search(query) {
      if (controller) controller.abort();
      controller = typeof AbortController !== "undefined" ? new AbortController() : null;
      var mine = ++seq, asked = mode;
      pending = true;
      var init = { credentials: "same-origin", headers: { Accept: "application/json" } };
      if (controller) init.signal = controller.signal;
      fetch("/api/search?" + (asked === "do" ? "mode=do&" : "") + "q=" + encodeURIComponent(query), init).then(function (response) {
        if (response.status === 401) { var auth = new Error("auth"); auth.auth = true; throw auth; }
        if (!response.ok) throw new Error("search " + response.status);
        return response.json();
      }).then(function (data) {
        if (mine !== seq || mode !== asked || query !== input.value.trim()) return;
        pending = false;
        if (asked === "do") renderDo(cleanCommands(data), query);
        else render(cleanGroups(data), query);
        if (openAfter) {
          openAfter = false;
          // Only an internal link is opened on a deferred Enter (a new tab opened
          // after a fetch is not a user gesture); an external top hit is just selected.
          if (options[0] && options[0].target !== "_blank") options[0].click();
        }
      }).catch(function (err) {
        if (mine !== seq || mode !== asked || (err && err.name === "AbortError")) return;
        pending = false; openAfter = false;
        if (asked === "do") renderDo([], query); else render([], query);
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
      endConfirm(false);
      mode = next;
      pending = false; openAfter = false;
      modeButtons.forEach(function (b) { b.setAttribute("aria-pressed", b.dataset.launcherMode === mode ? "true" : "false"); });
      // The "go" panel (results list, status line) is shared by Go and Do.
      panels.forEach(function (p) { p.hidden = p.dataset.launcherPanel !== (mode === "do" ? "go" : mode); });
      if (appearance) appearance.hidden = true;
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
        input.setAttribute("aria-label", mode === "do" ? "Filter commands" : "Search GYST");
        input.placeholder = mode === "do" ? "Filter commands" : "Search GYST, or > for commands, + to add a task";
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
      endConfirm(false);
      input.value = ""; title.value = "";
      setMode("go", false);
      input.focus();
    }
    function close() { if (palette.open) palette.close(); }

    palette.addEventListener("close", function () {
      pending = false; openAfter = false;
      endConfirm(false);
      clearTimeout(timer);
      seq++;
      if (controller) controller.abort();
      if (trigger && trigger.isConnected && trigger.focus) trigger.focus();
    });
    document.querySelectorAll("[data-command-open]").forEach(function (button) { button.addEventListener("click", open); });
    if (closeButton) closeButton.addEventListener("click", close);
    modeButtons.forEach(function (b) { b.addEventListener("click", function () { setMode(b.dataset.launcherMode, true); }); });

    input.addEventListener("input", function () {
      endConfirm(false); // typing leaves the confirm step and shows the list again
      var parsed = parseInput(input.value, mode);
      if (parsed.mode !== mode) { input.value = parsed.text; setMode(parsed.mode, false); if (mode === "do") return; }
      if (mode === "capture") applyCapture();
      else {
        // Anything already asked for or in flight is now about an older query.
        seq++;
        if (controller) controller.abort();
        setActive(-1); pending = true; openAfter = false; runSearch(false);
      }
    });

    input.addEventListener("keydown", function (event) {
      if (event.isComposing) return;
      // While a confirm is showing, the hidden list must not react.
      if (confirming && (event.key === "Enter" || event.key === "ArrowDown" || event.key === "ArrowUp")) { event.preventDefault(); return; }
      if (event.key === "Tab" && !event.shiftKey && !event.ctrlKey && !event.metaKey && !event.altKey) {
        event.preventDefault();
        setMode(nextMode(mode), true);
      } else if (mode !== "capture" && !pending && event.key === "ArrowDown") { event.preventDefault(); setActive(step(active, 1, options.length)); }
      else if (mode !== "capture" && !pending && event.key === "ArrowUp") { event.preventDefault(); setActive(step(active, -1, options.length)); }
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
      if (event.key !== "Escape") return;
      event.preventDefault();
      if (confirming) endConfirm(true); else close();
    });
    if (confirmBox) confirmBox.addEventListener("click", function (event) {
      var t = event.target;
      if (t && t.dataset && t.dataset.doCancel === "1") { event.preventDefault(); endConfirm(true); }
    });
    results.addEventListener("click", function (event) {
      var a = event.target && event.target.closest ? event.target.closest(".launcher-item") : null;
      if (!a) return;
      if (a.dataset.doPost === "1") { event.preventDefault(); askConfirm(a.gystCommand); return; }
      if (a.getAttribute("aria-disabled") === "true") {
        event.preventDefault();
        setStatus(a.gystCommand && a.gystCommand.disabledReason ? a.gystCommand.disabledReason : "Not available right now.");
        return;
      }
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
