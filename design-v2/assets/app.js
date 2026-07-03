/* ==========================================================================
   LifeOps design prototype v2 — map + small interactions.
   ~200 lines, no dependencies, no continuous rAF loops. The only script-driven
   animation is the marker moving when a node is clicked; everything else is a
   short CSS transition on hover/press.
   ========================================================================== */
(function () {
  "use strict";

  var VB_W = 1000;

  function reducedMotion() {
    return (
      window.matchMedia("(prefers-reduced-motion: reduce)").matches ||
      document.documentElement.dataset.motion === "reduced"
    );
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

  /* ---------------- map ---------------- */
  function initMap(map) {
    var VB_H = +(map.dataset.vbh || 320);
    var state = { path: null, total: 0, len: 0, busy: false };

    var svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
    svg.setAttribute("class", "routes");
    svg.setAttribute("viewBox", "0 0 " + VB_W + " " + VB_H);
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
      var q = state.path.getPointAtLength(Math.min(state.total, len + 2));
      var w = map.clientWidth, h = map.clientHeight;
      var x = (p.x / VB_W) * w, y = (p.y / VB_H) * h;
      var deg = Math.atan2((q.y - p.y) * h / VB_H, (q.x - p.x) * w / VB_W) * 180 / Math.PI;
      marker.style.transform = "translate(" + x + "px," + y + "px) rotate(" + deg + "deg)";
    }

    function build() {
      var ns = nodes();
      if (ns.length < 2) return;
      var pts = ns.map(function (n) {
        n.style.left = (+n.dataset.x / VB_W) * 100 + "%";
        n.style.top = (+n.dataset.y / VB_H) * 100 + "%";
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
      var html = "<b>" + d.title + "</b>";
      if (d.status) html += '<div class="row"><span>' + d.status + "</span></div>";
      if (d.next) html += '<div class="row"><span>Next</span><span>' + d.next + "</span></div>";
      if (d.due) html += '<div class="row"><span>Due</span><span>' + d.due + "</span></div>";
      if (d.progress) {
        html += '<div class="row"><span>Progress</span><span>' + d.progress + "%</span></div>" +
                '<div class="progress"><span style="width:' + d.progress + '%"></span></div>';
      }
      tip.innerHTML = html;
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
      var dur = Math.min(900, Math.max(450, Math.abs(target - from)));
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

  /* ---------------- roadmap page: sidebar switches the dataset ------------- */
  function initRoadmapSwitcher() {
    var host = document.querySelector("[data-roadmap-host]");
    var dataEl = document.querySelector("script[data-roadmaps]");
    if (!host || !dataEl) return;
    var sets = JSON.parse(dataEl.textContent);
    var map = host.querySelector(".map");
    var api = null;

    function render(i) {
      var set = sets[i];
      map.querySelectorAll(".node").forEach(function (n) { n.remove(); });
      set.nodes.forEach(function (n) {
        var b = document.createElement("button");
        b.className = "node " + (n.state || "");
        b.dataset.x = n.x; b.dataset.y = n.y;
        b.dataset.title = n.title;
        if (n.status) b.dataset.status = n.status;
        if (n.next) b.dataset.next = n.next;
        if (n.due) b.dataset.due = n.due;
        if (n.progress != null) b.dataset.progress = n.progress;
        if (n.href) b.dataset.href = n.href;
        b.innerHTML = "<i></i>" + (n.state === "current" ? '<span class="lbl">' + n.title + "</span>" : "");
        b.setAttribute("aria-label", n.title + (n.status ? ", " + n.status : ""));
        map.appendChild(b);
      });
      var name = document.querySelector("[data-roadmap-name]");
      var sub = document.querySelector("[data-roadmap-sub]");
      var next = document.querySelector("[data-roadmap-next]");
      if (name) name.textContent = set.name;
      if (sub) sub.textContent = set.sub;
      if (next) next.textContent = set.next;
      if (api) api.rebuild(); else api = initMap(map);
      document.querySelectorAll("[data-roadmap-idx]").forEach(function (b, j) {
        b.setAttribute("aria-current", j === i ? "true" : "false");
        b.classList.toggle("active", j === i);
      });
    }

    document.querySelectorAll("[data-roadmap-idx]").forEach(function (b, i) {
      b.addEventListener("click", function () { render(i); });
    });
    render(0);
  }

  /* ---------------- small stuff ---------------- */
  function initMisc() {
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
  }

  try {
    if (localStorage.getItem("lifeops-motion") === "reduced") {
      document.documentElement.dataset.motion = "reduced";
    }
  } catch (e) {}

  function init() {
    /* maps not owned by the roadmap switcher */
    document.querySelectorAll(".map").forEach(function (m) {
      if (!m.closest("[data-roadmap-host]")) initMap(m);
    });
    initRoadmapSwitcher();
    initMisc();
  }

  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init);
  else init();
})();
