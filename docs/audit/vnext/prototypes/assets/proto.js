/* GYST vNext prototype runtime — NON-PRODUCTION. Shell, icons, sheets, theme. Fictional data only. */
(function () {
  "use strict";
  var P = {
    home: '<path d="M3 10.5 12 3l9 7.5"/><path d="M5 9.5V20h5v-6h4v6h5V9.5"/>',
    folder: '<path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2Z"/>',
    heart: '<path d="M12 20s-7-4.4-7-10a4 4 0 0 1 7-2.6A4 4 0 0 1 19 10c0 5.6-7 10-7 10Z"/>',
    book: '<path d="M4 5.5A2.5 2.5 0 0 1 6.5 3H20v15H6.5A2.5 2.5 0 0 0 4 20.5Z"/><path d="M4 20.5A2.5 2.5 0 0 1 6.5 18H20v3H6.5"/>',
    more: '<circle cx="5" cy="12" r="1.3"/><circle cx="12" cy="12" r="1.3"/><circle cx="19" cy="12" r="1.3"/>',
    search: '<circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/>',
    plus: '<path d="M12 5v14M5 12h14"/>',
    chev: '<path d="m9 6 6 6-6 6"/>',
    back: '<path d="m15 6-6 6 6 6"/>',
    check: '<path d="m5 12 5 5 9-10"/>',
    bell: '<path d="M6 16V11a6 6 0 1 1 12 0v5l2 2H4Z"/><path d="M10 20a2 2 0 0 0 4 0"/>',
    alert: '<path d="M12 3 2 20h20Z"/><path d="M12 10v4M12 17h.01"/>',
    circleAlert: '<circle cx="12" cy="12" r="9"/><path d="M12 8v5M12 16h.01"/>',
    pr: '<circle cx="6" cy="6" r="2.2"/><circle cx="6" cy="18" r="2.2"/><circle cx="18" cy="18" r="2.2"/><path d="M6 8.2v7.6M18 15.8V10a3 3 0 0 0-3-3h-4"/><path d="m13 5-2 2 2 2"/>',
    branch: '<circle cx="6" cy="5" r="2"/><circle cx="6" cy="19" r="2"/><circle cx="18" cy="8" r="2"/><path d="M6 7v10M18 10c0 4-6 3-11 7"/>',
    code: '<path d="m8 8-4 4 4 4M16 8l4 4-4 4M13.5 5l-3 14"/>',
    bot: '<rect x="4" y="8" width="16" height="11" rx="3"/><path d="M12 4v4M9 13h.01M15 13h.01M2 13v2M22 13v2"/>',
    server: '<rect x="3" y="4" width="18" height="7" rx="2"/><rect x="3" y="13" width="18" height="7" rx="2"/><path d="M7 7.5h.01M7 16.5h.01"/>',
    zap: '<path d="M13 3 4 14h7l-1 7 9-11h-7Z"/>',
    grid: '<rect x="4" y="4" width="7" height="7" rx="2"/><rect x="13" y="4" width="7" height="7" rx="2"/><rect x="4" y="13" width="7" height="7" rx="2"/><rect x="13" y="13" width="7" height="7" rx="2"/>',
    target: '<circle cx="12" cy="12" r="8"/><circle cx="12" cy="12" r="4"/><circle cx="12" cy="12" r=".6"/>',
    settings: '<circle cx="12" cy="12" r="3"/><path d="M12 2v3M12 19v3M4.2 4.2l2.1 2.1M17.7 17.7l2.1 2.1M2 12h3M19 12h3M4.2 19.8l2.1-2.1M17.7 6.3l2.1-2.1"/>',
    clock: '<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>',
    moon: '<path d="M20 14.5A8 8 0 1 1 9.5 4a6.5 6.5 0 0 0 10.5 10.5Z"/>',
    sun: '<circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4"/>',
    x: '<path d="M6 6l12 12M18 6 6 18"/>',
    play: '<path d="M7 5v14l11-7Z"/>',
    stop: '<rect x="6" y="6" width="12" height="12" rx="2"/>',
    refresh: '<path d="M20 12a8 8 0 1 1-2.3-5.7M20 4v5h-5"/>',
    ext: '<path d="M14 4h6v6M20 4l-9 9M18 14v5a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V7a1 1 0 0 1 1-1h5"/>',
    info: '<circle cx="12" cy="12" r="9"/><path d="M12 11v5M12 8h.01"/>',
    dumbbell: '<path d="M6 7v10M18 7v10M3 10v4M21 10v4M6 12h12"/>',
    flame: '<path d="M12 21c-4 0-6-2.7-6-6 0-4 4-6 4-10 3 2 8 5 8 10 0 3.3-2 6-6 6Z"/><path d="M12 21c-1.7 0-3-1.2-3-3 0-2 3-3.5 3-3.5s3 1.5 3 3.5c0 1.8-1.3 3-3 3Z"/>',
    steps: '<path d="M8 3c2 0 3 2 3 5s-1 5-3 5-3-2-3-5 1-5 3-5ZM16 9c2 0 3 2 3 5s-1 5-3 5-3-2-3-5 1-5 3-5ZM6 16h4M14 21h4"/>',
    scale: '<rect x="3" y="3" width="18" height="18" rx="4"/><path d="M8 9a5 5 0 0 1 8 0l-3 3"/>',
    utensils: '<path d="M5 3v8a2 2 0 0 0 2 2v8M9 3v8M7 3v6M17 3c-2 1-3 3-3 6v3h3v9"/>',
    calendar: '<rect x="3" y="5" width="18" height="16" rx="3"/><path d="M3 10h18M8 3v4M16 3v4"/>',
    cap: '<path d="m2 9 10-5 10 5-10 5Z"/><path d="M6 11v5c3 2.5 9 2.5 12 0v-5M22 9v5"/>',
    layers: '<path d="m12 3 9 5-9 5-9-5Z"/><path d="m3 13 9 5 9-5"/>',
    route: '<circle cx="6" cy="19" r="2"/><circle cx="18" cy="5" r="2"/><path d="M8 19h7a3.5 3.5 0 0 0 0-7H9a3.5 3.5 0 0 1 0-7h7"/>',
    orbit: '<circle cx="12" cy="12" r="3"/><ellipse cx="12" cy="12" rx="10" ry="4.5" transform="rotate(-25 12 12)"/>',
    activity: '<path d="M3 12h4l3-8 4 16 3-8h4"/>',
    database: '<ellipse cx="12" cy="5.5" rx="8" ry="3"/><path d="M4 5.5v13c0 1.7 3.6 3 8 3s8-1.3 8-3v-13M4 12c0 1.7 3.6 3 8 3s8-1.3 8-3"/>',
    shield: '<path d="M12 3 4 6v6c0 5 3.5 8 8 9 4.5-1 8-4 8-9V6Z"/>',
    offline: '<path d="M2 8a15 15 0 0 1 4-2.6M9.5 4.2A15 15 0 0 1 22 8M5 12a10 10 0 0 1 3.5-2.3M16 10a10 10 0 0 1 3 2M8.5 15.5a5 5 0 0 1 7 0M12 19h.01M3 3l18 18"/>',
    archive: '<rect x="3" y="4" width="18" height="5" rx="1.5"/><path d="M5 9v10a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1V9M10 13h4"/>',
    list: '<path d="M9 6h11M9 12h11M9 18h11M4.5 6h.01M4.5 12h.01M4.5 18h.01"/>',
    command: '<path d="M9 6a3 3 0 1 0-3 3h12a3 3 0 1 0-3-3v12a3 3 0 1 0 3-3H6a3 3 0 1 0 3 3Z"/>',
    edit: '<path d="M4 20h4L19 9l-4-4L4 16Z"/>',
    tool: '<path d="M14.5 6.5a4 4 0 0 0 5 5L12 19a2.1 2.1 0 1 1-3-3l7.5-7.5a4 4 0 0 1-2-2Z"/>',
    cpu: '<rect x="6" y="6" width="12" height="12" rx="2"/><path d="M9 2v4M15 2v4M9 18v4M15 18v4M2 9h4M2 15h4M18 9h4M18 15h4"/>',
    disk: '<rect x="3" y="4" width="18" height="16" rx="3"/><path d="M3 14h18M7 17h.01"/>',
    moonstar: '<path d="M14 3a7 7 0 1 0 7 7 5.5 5.5 0 0 1-7-7Z"/>',
    sparkle: '<path d="M12 3v4M12 17v4M3 12h4M17 12h4M6 6l2.5 2.5M15.5 15.5 18 18M6 18l2.5-2.5M15.5 8.5 18 6"/>',
    link: '<path d="M10 14a4 4 0 0 0 5.7 0l3-3a4 4 0 0 0-5.7-5.7l-1 1M14 10a4 4 0 0 0-5.7 0l-3 3a4 4 0 0 0 5.7 5.7l1-1"/>',
    user: '<circle cx="12" cy="8" r="4"/><path d="M4 21a8 8 0 0 1 16 0"/>',
    lock: '<rect x="4" y="10" width="16" height="11" rx="2"/><path d="M8 10V7a4 4 0 0 1 8 0v3"/>',
    minus: '<path d="M5 12h14"/>',
    ship: '<path d="M12 2c3 3 4 7 3.5 11L12 16l-3.5-3C8 9 9 5 12 2Z"/><path d="M8.5 13 5 16l3.5 1M15.5 13 19 16l-3.5 1M11 19l1 3 1-3"/>'
  };
  var sprite = '<svg xmlns="http://www.w3.org/2000/svg" style="display:none" aria-hidden="true">' +
    Object.keys(P).map(function (k) { return '<symbol id="i-' + k + '" viewBox="0 0 24 24">' + P[k] + '</symbol>'; }).join("") + "</svg>";
  document.body.insertAdjacentHTML("afterbegin", sprite);
  function ic(n, cls) { return '<svg class="i ' + (cls || "") + '" aria-hidden="true"><use href="#i-' + n + '"/></svg>'; }
  window.GYST = { ic: ic };

  /* ---------- Theme (prototype toggle, stored per viewer) ---------- */
  var root = document.documentElement;
  try { var saved = localStorage.getItem("gyst-proto-theme"); if (saved) root.setAttribute("data-theme", saved); } catch (e) {}
  function toggleTheme() {
    var dark = root.getAttribute("data-theme") === "dark" || (!root.getAttribute("data-theme") && matchMedia("(prefers-color-scheme: dark)").matches);
    var next = dark ? "light" : "dark";
    root.setAttribute("data-theme", next);
    try { localStorage.setItem("gyst-proto-theme", next); } catch (e) {}
  }

  /* ---------- Shell ---------- */
  var tab = document.body.getAttribute("data-tab") || "today";
  var area = document.body.getAttribute("data-area") || "";
  var tabs = [
    ["today", "Today", "home", "today.html"],
    ["projects", "Projects", "folder", "projects.html"],
    ["health", "Health", "heart", "health.html"],
    ["learn", "Learn", "cap", "learn.html"],
    ["more", "More", "more", "more.html"]
  ];
  var attnCount = document.body.getAttribute("data-attn") || "3";
  var tabHTML = '<nav class="tabbar" aria-label="Primary">' + tabs.map(function (t) {
    var badge = (t[0] === "today" && attnCount !== "0") ? '<span class="tab-badge" aria-label="' + attnCount + ' items need attention">' + attnCount + "</span>" : "";
    return '<a href="' + t[3] + '"' + (t[0] === tab ? ' aria-current="page"' : "") + ">" + ic(t[2]) + "<span>" + t[1] + "</span>" + badge + "</a>";
  }).join("") + "</nav>";
  function side(href, label, icon, key, extra, cls) {
    return '<a class="' + (cls || "") + '" href="' + href + '"' + (key === area || (!area && key === tab) ? ' aria-current="page"' : "") + ">" + (icon ? ic(icon) : "") + "<span>" + label + "</span>" + (extra || "") + "</a>";
  }
  var sideHTML = '<nav class="sidebar" aria-label="Primary">' +
    '<div class="brand"><span class="o" aria-hidden="true"></span>GYST</div>' +
    '<a class="find" href="launcher.html">' + ic("search", "i-16") + "<span>Search or run…</span><kbd>⌘K</kbd></a>" +
    side("today.html", "Today", "home", "today", attnCount !== "0" ? '<span class="badge" aria-label="' + attnCount + ' attention items">' + attnCount + "</span>" : "") +
    side("projects.html", "Projects", "folder", "projects") +
    (tab === "projects" ?
      side("development.html", "Development", "", "development", "", "sub") +
      side("galaxy.html", "Galaxy", "", "galaxy", "", "sub") +
      side("projects.html#roadmaps", "Roadmaps", "", "roadmaps", "", "sub") : "") +
    side("health.html", "Health", "heart", "health") +
    side("learn.html", "Learn", "cap", "learn") +
    '<div class="grp">Operate</div>' +
    side("system.html", "System", "server", "system", (document.body.getAttribute("data-sys") || "crit") === "ok" ? '<span class="dot ok" aria-label="Healthy"></span>' : '<span class="dot crit" aria-label="1 problem"></span>') +
    side("apps.html", "Apps", "grid", "apps") +
    '<div class="spacer"></div>' +
    side("more.html", "More", "more", "more") +
    '<a href="index.html">' + ic("layers") + "<span>Prototype index</span></a>" +
    "</nav>";
  var app = document.querySelector(".app");
  if (app) { app.insertAdjacentHTML("afterbegin", sideHTML); app.insertAdjacentHTML("beforeend", tabHTML); }

  /* topbar title fades in after scrolling past the page header */
  var topbar = document.querySelector(".topbar");
  if (topbar) {
    var onScroll = function () { topbar.classList.toggle("scrolled", window.scrollY > 48); };
    addEventListener("scroll", onScroll, { passive: true }); onScroll();
  }

  /* ---------- Sheets, dialogs, toast ---------- */
  var scrim = document.createElement("div"); scrim.className = "scrim"; document.body.appendChild(scrim);
  var lastFocus = null;
  function openLayer(el) { lastFocus = document.activeElement; el.classList.add("open"); el.hidden = false; scrim.classList.add("open"); var f = el.querySelector("button, a, input"); if (f) setTimeout(function () { f.focus(); }, 60); }
  function closeLayers() { document.querySelectorAll(".sheet.open, .dialog.open").forEach(function (el) { el.classList.remove("open"); }); scrim.classList.remove("open"); if (lastFocus) lastFocus.focus(); }
  document.addEventListener("click", function (e) {
    var o = e.target.closest("[data-open]");
    if (o) { e.preventDefault(); var el = document.getElementById(o.getAttribute("data-open")); if (el) openLayer(el); return; }
    var cl = e.target.closest("[data-close]"); if (cl) { e.preventDefault(); closeLayers(); if (cl.dataset.toast) toast(cl.dataset.toast, cl.dataset.undo); return; }
    var t = e.target.closest("[data-toggle-theme]"); if (t) { toggleTheme(); return; }
    var tg = e.target.closest("[data-press]"); if (tg) { tg.setAttribute("aria-pressed", tg.getAttribute("aria-pressed") === "true" ? "false" : "true"); if (tg.dataset.toast) toast(tg.dataset.toast); return; }
    var seg = e.target.closest(".seg button, .filters .filter[data-single]");
    if (seg) { var parent = seg.parentElement; parent.querySelectorAll("[aria-pressed]").forEach(function (b) { b.setAttribute("aria-pressed", String(b === seg)); }); if (seg.dataset.show) showGroup(seg.dataset.group, seg.dataset.show); return; }
    var ts = e.target.closest("[data-toast]"); if (ts) { e.preventDefault(); toast(ts.dataset.toast, ts.dataset.undo); }
  });
  scrim.addEventListener("click", closeLayers);
  document.addEventListener("keydown", function (e) {
    if (e.key === "Escape") closeLayers();
    if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") { e.preventDefault(); location.href = "launcher.html"; }
  });
  document.addEventListener("click", function (e) { if (e.target.closest("[data-launch]")) location.href = "launcher.html"; });
  function showGroup(group, name) { document.querySelectorAll('[data-group="' + group + '"][data-name]').forEach(function (el) { el.hidden = el.getAttribute("data-name") !== name; }); }
  var toastEl = document.createElement("div"); toastEl.className = "toast"; toastEl.setAttribute("role", "status"); document.body.appendChild(toastEl);
  var tt;
  function toast(msg, undo) { toastEl.innerHTML = "<span>" + msg + "</span>" + (undo ? '<button type="button">Undo</button>' : ""); toastEl.classList.add("show"); clearTimeout(tt); tt = setTimeout(function () { toastEl.classList.remove("show"); }, 2600); }
  window.GYST.toast = toast;

  /* ---------- Desktop inspector selection ---------- */
  document.querySelectorAll("[data-inspect]").forEach(function (row) {
    row.addEventListener("click", function (e) {
      if (!matchMedia("(min-width: 1280px)").matches) return; /* phone/tablet follow the link to the detail page */
      e.preventDefault(); e.stopPropagation();
      document.querySelectorAll("[data-inspect]").forEach(function (r) { r.removeAttribute("aria-current"); });
      row.setAttribute("aria-current", "true");
      var id = row.getAttribute("data-inspect");
      document.querySelectorAll(".inspector [data-for]").forEach(function (p) { p.hidden = p.getAttribute("data-for") !== id; });
    });
  });
})();
