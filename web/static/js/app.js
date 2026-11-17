/* ==========================================================================
   Design v2 map + interactions (port of docs/design/v2/assets/app.js).
   Only touches elements marked [data-map-v2], so unported v1 pages are
   unaffected. No continuous rAF loops: the single script animation is the
   marker flight on node click, one short run that terminates.

   Coordinates: node data-x / data-y are percentages of the map box on both
   axes (the server renders LeftPct/TopPct), so the SVG uses a 0-100 viewBox
   stretched to fill.
   ========================================================================== */
(function () {
  "use strict";

  // Delegated handlers replacing the inline on*= attributes, so the CSP can
  // keep script-src 'self' with no 'unsafe-inline'.
  document.addEventListener("change", function (event) {
    var el = event.target;
    if (!el || !el.matches || !el.matches("[data-autosubmit]") || !el.form) return;
    if (el.dataset.autosubmit === "plain") el.form.submit();
    else el.form.requestSubmit();
  });
  document.addEventListener("submit", function (event) {
    var form = event.target;
    if (form && form.matches && form.matches("form[data-confirm]") && !window.confirm(form.dataset.confirm)) {
      event.preventDefault();
    }
  });
  document.addEventListener("click", function (event) {
    var btn = event.target && event.target.closest ? event.target.closest("[data-copy-target]") : null;
    if (!btn) return;
    var el = document.getElementById(btn.dataset.copyTarget);
    if (!el || !navigator.clipboard) return;
    navigator.clipboard.writeText(el.value).then(function () { btn.textContent = "Copied"; });
  });

  var VB = 100;

  function reducedMotion() {
    return (
      window.matchMedia("(prefers-reduced-motion: reduce)").matches ||
      document.documentElement.dataset.motion === "reduced"
    );
  }

	function urlBase64ToUint8Array(value) {
	  var padding = "=".repeat((4 - value.length % 4) % 4);
	  var raw = atob((value + padding).replace(/-/g, "+").replace(/_/g, "/"));
	  return Uint8Array.from(raw, function (character) { return character.charCodeAt(0); });
	}

	function enableWebPush() {
	  if (!("serviceWorker" in navigator) || !("PushManager" in window)) return Promise.resolve(false);
	  return navigator.serviceWorker.ready.then(function (registration) {
		return fetch("/api/push/vapid-key", { headers: { Accept: "application/json" } }).then(function (response) {
		  if (!response.ok) return null;
		  return response.json();
		}).then(function (config) {
		  if (!config) return false;
		  return registration.pushManager.getSubscription().then(function (existing) {
			return existing || registration.pushManager.subscribe({ userVisibleOnly: true, applicationServerKey: urlBase64ToUint8Array(config.publicKey) });
		  }).then(function (subscription) {
			return fetch("/api/push/subscribe", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(subscription) });
		  }).then(function (response) { return response.ok; });
		});
	  }).catch(function () { return false; });
	}

  /* Smooth curve through points (Catmull-Rom -> cubic beziers) */
  function curve(pts) {
    var d = "M" + pts[0].x + " " + pts[0].y;
    for (var i = 0; i < pts.length - 1; i++) {
      var p0 = pts[i - 1] || pts[i], p1 = pts[i], p2 = pts[i + 1], p3 = pts[i + 2] || p2;
      d += " C" + (p1.x + (p2.x - p0.x) / 6).toFixed(1) + " " + (p1.y + (p2.y - p0.y) / 6).toFixed(1) +
           "," + (p2.x - (p3.x - p1.x) / 6).toFixed(1) + " " + (p2.y - (p3.y - p1.y) / 6).toFixed(1) +
           "," + p2.x + " " + p2.y;
    }
    return d;
  }

  var MARKER =
    '<svg viewBox="0 0 22 22" aria-hidden="true">' +
    '<path d="M3.5 11 L18.5 5.5 L14.5 11 L18.5 16.5 Z" fill="currentColor"/></svg>';

  function initMap(map) {
    var state = { path: null, total: 0, len: 0, busy: false };

    var svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
    svg.setAttribute("class", "routes");
    svg.setAttribute("viewBox", "0 0 " + VB + " " + VB);
    svg.setAttribute("preserveAspectRatio", "none");
    svg.setAttribute("aria-hidden", "true");
    map.insertBefore(svg, map.firstChild);

    var marker = document.createElement("div");
    marker.className = "marker";
    marker.innerHTML = MARKER;
    map.appendChild(marker);

    var tip = document.createElement("div");
    tip.className = "tip";
    tip.setAttribute("role", "status");
    map.appendChild(tip);

    function nodes() {
      return Array.prototype.slice.call(map.querySelectorAll(".node"));
    }

    function lengthNear(pt) {
      var best = 0, bd = Infinity;
      for (var i = 0; i <= 300; i++) {
        var l = (state.total * i) / 300;
        var p = state.path.getPointAtLength(l);
        var d = (p.x - pt.x) * (p.x - pt.x) + (p.y - pt.y) * (p.y - pt.y);
        if (d < bd) { bd = d; best = l; }
      }
      return best;
    }

    function placeMarker(len) {
      var p = state.path.getPointAtLength(len);
      var q = state.path.getPointAtLength(Math.min(state.total, len + 1));
      var w = map.clientWidth, h = map.clientHeight;
      var x = (p.x / VB) * w, y = (p.y / VB) * h;
      var deg = Math.atan2((q.y - p.y) * h / VB, (q.x - p.x) * w / VB) * 180 / Math.PI;
      marker.style.transform = "translate(" + x + "px," + y + "px) rotate(" + deg + "deg)";
    }

    function build() {
      var ns = nodes();
      if (ns.length < 2) return;
      var pts = ns.map(function (n) {
        n.style.left = +n.dataset.x + "%";
        n.style.top = +n.dataset.y + "%";
        return { x: +n.dataset.x, y: +n.dataset.y };
      });
      var d = curve(pts);
      svg.innerHTML =
        '<path class="route" vector-effect="non-scaling-stroke" d="' + d + '"/>' +
        '<path class="route-done" vector-effect="non-scaling-stroke" d="' + d + '"/>';
      state.path = svg.lastChild;
      state.total = state.path.getTotalLength();
      var cur = Math.max(0, ns.findIndex(function (n) { return n.classList.contains("current"); }));
      state.len = lengthNear(pts[cur]);
      state.path.style.strokeDasharray = state.len + " " + (state.total + 10);
      placeMarker(state.len);
      wire(ns);
    }

    function showTip(n) {
      var d = n.dataset;
      var html = "<b></b>";
      tip.innerHTML = html;
      tip.firstChild.textContent = d.title || "";
      if (d.status) {
        var row = document.createElement("div");
        row.className = "row";
        var span = document.createElement("span");
        span.textContent = d.status;
        row.appendChild(span);
        tip.appendChild(row);
      }
      var mr = map.getBoundingClientRect(), nr = n.getBoundingClientRect();
      var x = nr.left - mr.left + nr.width / 2;
      var y = nr.top - mr.top;
      tip.style.left = Math.max(8, Math.min(x - 110, mr.width - 230)) + "px";
      var above = y - tip.offsetHeight - 10;
      tip.style.top = (above > 6 ? above : y + nr.height + 10) + "px";
      tip.classList.add("open");
    }
    function hideTip() { tip.classList.remove("open"); }

    /* Marker flight: a single short rAF run, only on click. */
    function fly(target, cb) {
      if (reducedMotion()) { state.len = target; placeMarker(target); if (cb) cb(); return; }
      if (state.busy) return;
      state.busy = true;
      var from = state.len, t0 = performance.now();
      var dur = Math.min(900, Math.max(450, Math.abs(target - from) * 4));
      (function frame(now) {
        var t = Math.min(1, (now - t0) / dur);
        var e = t < 0.5 ? 2 * t * t : 1 - Math.pow(-2 * t + 2, 2) / 2;
        placeMarker(from + (target - from) * e);
        if (t < 1) requestAnimationFrame(frame);
        else { state.len = target; state.busy = false; if (cb) cb(); }
      })(t0);
    }

    function wire(ns) {
      ns.forEach(function (n) {
        n.addEventListener("mouseenter", function () { showTip(n); });
        n.addEventListener("mouseleave", hideTip);
        n.addEventListener("focus", function () { showTip(n); });
        n.addEventListener("blur", hideTip);
        n.addEventListener("click", function () {
          if (n.classList.contains("locked")) { showTip(n); return; }
          fly(lengthNear({ x: +n.dataset.x, y: +n.dataset.y }), function () {
            if (n.dataset.href) window.location.href = n.dataset.href;
          });
        });
      });
    }

    build();
    window.addEventListener("resize", function () { if (state.path) placeMarker(state.len); });

    /* Pause the current-node pulse when the map is offscreen. */
    if ("IntersectionObserver" in window) {
      new IntersectionObserver(function (entries) {
        map.classList.toggle("paused", !entries[0].isIntersecting);
      }).observe(map);
    }

    return { rebuild: build };
  }

  /* ---------------- small stuff (copy buttons, reduced-motion toggle) ------ */
  function initMisc() {
    document.querySelectorAll("[data-next-prayer]").forEach(function (hero) {
      var output = hero.querySelector("[data-prayer-countdown]");
      var target = new Date(hero.dataset.nextPrayer).getTime();
	  var enable = hero.querySelector("[data-prayer-enable]");
	  var test = hero.querySelector("[data-prayer-test]");
	  var alertState = hero.querySelector("[data-prayer-alert-state]");
	  var audio = document.querySelector("[data-prayer-audio]");
	  var audioContext = null;
	  var alertsEnabled = false;
	  var prayerTimes = Array.prototype.slice.call(document.querySelectorAll("[data-prayer-at]"));

	  function chime() {
		var Context = window.AudioContext || window.webkitAudioContext;
		if (!Context) return;
		audioContext = audioContext || new Context();
		if (audioContext.state === "suspended") audioContext.resume();
		[0, 0.18].forEach(function (delay, index) {
		  var oscillator = audioContext.createOscillator();
		  var gain = audioContext.createGain();
		  oscillator.frequency.value = index ? 659.25 : 523.25;
		  gain.gain.setValueAtTime(0.0001, audioContext.currentTime + delay);
		  gain.gain.exponentialRampToValueAtTime(0.16, audioContext.currentTime + delay + 0.02);
		  gain.gain.exponentialRampToValueAtTime(0.0001, audioContext.currentTime + delay + 0.55);
		  oscillator.connect(gain).connect(audioContext.destination);
		  oscillator.start(audioContext.currentTime + delay);
		  oscillator.stop(audioContext.currentTime + delay + 0.58);
		});
	  }
	  function playAlert() {
		if (audio && audio.getAttribute("src")) {
		  audio.currentTime = 0;
		  audio.play().catch(function () { alertState.textContent = "Audio was blocked; use Test sound to enable it."; });
		} else chime();
	  }
	  function checkPrayerAlerts() {
		if (!alertsEnabled) return;
		var now = Date.now();
		prayerTimes.forEach(function (item) {
		  var at = new Date(item.dataset.prayerAt).getTime();
		  var key = item.dataset.prayerAt + ":" + item.dataset.prayerName;
		  if (now < at || now - at > 1500 || sessionStorage.getItem("lifeops-prayer-" + key)) return;
		  sessionStorage.setItem("lifeops-prayer-" + key, "sent");
		  if ("Notification" in window && Notification.permission === "granted") new Notification(item.dataset.prayerName + " prayer", { body: "It is time for " + item.dataset.prayerName + "." });
		  playAlert();
		});
	  }
	  enable.addEventListener("click", function () {
		playAlert();
		if (audio && audio.getAttribute("src")) {
		  audio.muted = true;
		  audio.play().then(function () { audio.pause(); audio.currentTime = 0; audio.muted = false; }).catch(function () { audio.muted = false; });
		}
		var permission = "Notification" in window ? Notification.requestPermission() : Promise.resolve("unsupported");
		permission.then(function (result) {
		  alertsEnabled = true;
		  var push = result === "granted" ? enableWebPush() : Promise.resolve(false);
		  push.then(function (pushEnabled) {
			alertState.textContent = result === "granted" ? (pushEnabled ? "Alerts are enabled, including closed-browser prayer notifications." : "Alerts and sound are enabled while this page is open.") : "Sound is enabled while this page is open; browser notifications are unavailable.";
		  });
		  enable.textContent = "Prayer alerts enabled";
		  enable.disabled = true;
		});
	  });
	  test.addEventListener("click", playAlert);
      function tick() {
        var remaining = Math.max(0, target - Date.now());
        var total = Math.floor(remaining / 1000);
        var hours = Math.floor(total / 3600);
        var minutes = Math.floor((total % 3600) / 60);
        var seconds = total % 60;
        output.textContent = hours + "h " + String(minutes).padStart(2, "0") + "m " + String(seconds).padStart(2, "0") + "s";
		checkPrayerAlerts();
        if (remaining === 0) window.setTimeout(function () { window.location.reload(); }, 1000);
      }
      tick();
      window.setInterval(tick, 1000);
    });

    document.querySelectorAll("[data-copy]").forEach(function (btn) {
      btn.addEventListener("click", function () {
        var el = document.querySelector(btn.dataset.copy);
        var done = function () {
          var t = btn.textContent;
          btn.textContent = "Copied";
          setTimeout(function () { btn.textContent = t; }, 1500);
        };
        if (navigator.clipboard && el) navigator.clipboard.writeText(el.textContent.trim()).then(done);
        else done();
      });
    });

    var motion = document.querySelector("[data-motion-toggle]");
    if (motion) {
      motion.checked = document.documentElement.dataset.motion === "reduced";
      motion.addEventListener("change", function () {
        if (motion.checked) document.documentElement.dataset.motion = "reduced";
        else delete document.documentElement.dataset.motion;
        try { localStorage.setItem("lifeops-motion", motion.checked ? "reduced" : ""); } catch (e) {}
      });
    }

    var navMenus = Array.prototype.slice.call(document.querySelectorAll(".nav-menu"));
    navMenus.forEach(function (menu) {
      menu.addEventListener("toggle", function () {
        if (!menu.open) return;
        navMenus.forEach(function (other) {
          if (other !== menu) other.open = false;
        });
      });
    });
    document.addEventListener("click", function (event) {
      navMenus.forEach(function (menu) {
        if (!menu.contains(event.target)) menu.open = false;
      });
    });
    document.addEventListener("keydown", function (event) {
      if (event.key !== "Escape") return;
      navMenus.forEach(function (menu) {
        if (menu.open && menu.contains(document.activeElement)) menu.querySelector("summary").focus();
        menu.open = false;
      });
    });
  }

	if ("serviceWorker" in navigator) window.addEventListener("load", function () { navigator.serviceWorker.register("/service-worker.js").catch(function () {}); });

  // Navigation may grow with text zoom or safe-area changes. Keep content and
  // the Galaxy inside the actual available rectangle, without fixed pixel guesses.
  function initShellClearance() {
    var header = document.querySelector(".gyst-topbar");
    var nav = document.querySelector(".gyst-primary");
    if (!header || !nav || !("ResizeObserver" in window)) return;
    var observer = new ResizeObserver(function () {
      document.documentElement.style.setProperty("--gyst-header-clearance", Math.ceil(header.getBoundingClientRect().height) + "px");
      document.documentElement.style.setProperty("--gyst-nav-clearance", (window.matchMedia("(min-width: 1024px)").matches ? 0 : Math.ceil(nav.getBoundingClientRect().height)) + "px");
    });
    observer.observe(header);
    observer.observe(nav);
  }

  // §L check-off: animate on the current page, then submit the same form.
  function motionMs(name, fallback) {
    var v = getComputedStyle(document.documentElement).getPropertyValue(name).trim(), n = parseFloat(v);
    return isNaN(n) ? fallback : (/ms$/.test(v) ? n : n * 1000);
  }

  function easeProgress(form) {
    var list = form.closest(".today-checklist");
    if (!list || !/^\/tasks\/\d+\/complete$/.test(form.getAttribute("action"))) return;
    var bar = list.querySelector(".gyst-progress");
    if (!bar) return;
    var d = +list.dataset.done, t = d + (+list.dataset.pending) + (+list.dataset.skipped);
    if (!t) return;
    var from = +bar.value, to = Math.floor((d + 1) * 100 / t), dur = motionMs("--vn-base", 240), t0 = performance.now();
    (function f(now) {
      var k = Math.min(1, (now - t0) / dur);
      bar.value = from + (to - from) * (1 - Math.pow(1 - k, 3));
      if (k < 1) requestAnimationFrame(f);
    })(t0);
  }

  function initCheckOff() {
    document.addEventListener("submit", function (event) {
      var form = event.target;
      // Already on its way: a second toggle during the animation must not post again.
      if (form.dataset.completing) { event.preventDefault(); return; }
      var box = form.querySelector(".today-task-check > input[type=checkbox], .project-task-check > input[type=checkbox], .task-check > input[type=checkbox]");
      if (!box || !box.checked || !/\/complete$/.test(form.getAttribute("action") || "")) return;
      if (reducedMotion()) return;
      event.preventDefault();
      form.dataset.completing = "1";
      // Schedule the real submit first so a failure in the animation can't lose it.
      setTimeout(function () { HTMLFormElement.prototype.submit.call(form); }, motionMs("--vn-base", 240) + 40);
      var row = form.closest("li");
      if (row) row.classList.add("is-completing");
      try { easeProgress(form); } catch (_) {}
    }, true);
    // Back/forward cache restores the page mid-state; undo it.
    window.addEventListener("pageshow", function (e) {
      if (!e.persisted) return;
      document.querySelectorAll("form[data-completing]").forEach(function (f) {
        delete f.dataset.completing;
        var b = f.querySelector("input[type=checkbox]");
        if (b) b.checked = false;
        var r = f.closest("li");
        if (r) r.classList.remove("is-completing");
      });
    });
  }

  // Pages about "today" ([data-day-page="YYYY-MM-DD"], e.g. Health Today)
  // reload when the app's day has changed while the PWA sat in the
  // background, so the water bottle and the day's numbers reset at midnight
  // without a manual refresh (#78). The server sends the instant its day ends
  // (data-day-ends, from GYST's pinned zone data), so the device's own zone
  // rules, which may be stale, never decide the day (#193). Without it, the
  // day is read in the app timezone (data-day-tz) by the browser.
  function initDayReload() {
    var page = document.querySelector("[data-day-page]");
    if (!page) return;
    // One-off notices (?water=added&wfrom=...) belong to the tap that caused
    // them; drop them from the address so a refresh or the day reload
    // doesn't replay the message and the animation.
    try {
      var u = new URL(location.href);
      if (u.searchParams.has("water") || u.searchParams.has("wfrom")) {
        u.searchParams.delete("water");
        u.searchParams.delete("wfrom");
        history.replaceState(history.state, "", u.pathname + u.search + u.hash);
      }
    } catch (_) {}
    var day = page.getAttribute("data-day-page"), tz = page.getAttribute("data-day-tz");
    var ends = Date.parse(page.getAttribute("data-day-ends") || "");
    var loaded = new Date().toDateString();
    function appDay() {
      if (!day || !tz) return null;
      try {
        var p = {};
        new Intl.DateTimeFormat("en-US", { timeZone: tz, year: "numeric", month: "2-digit", day: "2-digit" })
          .formatToParts(new Date()).forEach(function (x) { p[x.type] = x.value; });
        return p.year + "-" + p.month + "-" + p.day;
      } catch (_) { return null; }
    }
    function changed() {
      if (!isNaN(ends)) return Date.now() >= ends;
      var d = appDay();
      return d ? d !== day : new Date().toDateString() !== loaded;
    }
    function check() {
      if (document.visibilityState === "visible" && changed()) location.reload();
    }
    document.addEventListener("visibilitychange", check);
    window.addEventListener("pageshow", function (e) { if (e.persisted) check(); });
  }

  // Offline banner (VN-022, 13 §17): shown while the browser reports offline,
  // hidden again on the online event, without a reload.
  function initOfflineBanner() {
    var banner = document.querySelector("[data-gyst-offline]");
    if (!banner) return;
    function sync() { banner.hidden = navigator.onLine !== false; }
    window.addEventListener("online", sync);
    window.addEventListener("offline", sync);
    window.addEventListener("pageshow", sync);
    sync();
  }

  function init() {
    document.querySelectorAll("[data-map-v2]").forEach(function (m) { initMap(m); });
    initDayReload();
    initMisc();
    initShellClearance();
    initCheckOff();
    initOfflineBanner();
  }

  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init);
  else init();
})();
