(function () {
  "use strict";

  // viewBox of the route SVG; node %-positions are derived from path points in
  // these user units. preserveAspectRatio="meet" keeps the scale uniform, so the
  // tangent angle needs no anisotropy correction.
  var VB_W = 1000;
  var VB_H = 300;

  // One controller per .rocket-route instance. State (flight, timers) lives here
  // so it can be reset per-instance — never globally and never permanently.
  var controllers = [];

  function reducedMotion() {
    return (
      window.matchMedia &&
      window.matchMedia("(prefers-reduced-motion: reduce)").matches
    );
  }

  function pctLeft(pt) {
    return (pt.x / VB_W) * 100;
  }
  function pctTop(pt) {
    return (pt.y / VB_H) * 100;
  }

  function easeInOut(t) {
    return t < 0.5 ? 2 * t * t : 1 - Math.pow(-2 * t + 2, 2) / 2;
  }

  // Place the ship at a length along the path and aim its nose down the tangent.
  function placeRocket(rocket, path, len, total) {
    var pt = path.getPointAtLength(len);
    var ahead = path.getPointAtLength(Math.min(len + 1, total));
    var angle = (Math.atan2(ahead.y - pt.y, ahead.x - pt.x) * 180) / Math.PI;
    rocket.style.left = pctLeft(pt) + "%";
    rocket.style.top = pctTop(pt) + "%";
    // The icon art carries a static +45deg nose offset, so --angle is the raw
    // direction of travel.
    rocket.style.setProperty("--angle", angle + "deg");
  }

  function spawnTrail(track, rocket) {
    var dot = document.createElement("span");
    dot.className = "rocket-trail";
    dot.style.left = rocket.style.left;
    dot.style.top = rocket.style.top;
    track.appendChild(dot);
    window.setTimeout(function () {
      if (dot.parentNode) dot.parentNode.removeChild(dot);
    }, 700);
  }

  function wireRoute(route) {
    var track = route.querySelector(".route-track");
    var path = route.querySelector(".route-base");
    var rocket = route.querySelector(".rocket");
    var nodes = route.querySelectorAll(".route-node");
    if (!track || !path || !rocket || !nodes.length) return;
    if (typeof path.getTotalLength !== "function") return; // very old browser: links still work

    var total = path.getTotalLength();

    // Lay each node precisely on the path. getPointAtLength returns user-space
    // coords (0..1000 / 0..300) independent of rendered pixel size, so these
    // percentages are resolution-stable — no recompute on resize needed.
    var lengths = [];
    var restLen = 0;
    Array.prototype.forEach.call(nodes, function (node, i) {
      var pct = parseFloat(node.dataset.pathPct || "0");
      var len = (total * pct) / 100;
      lengths[i] = len;
      var pt = path.getPointAtLength(len);
      node.style.left = pctLeft(pt) + "%";
      node.style.top = pctTop(pt) + "%";
      if (node.classList.contains("is-current")) restLen = len;
    });

    var rafId = null;
    var navTimer = null;
    var launchedAt = 0;

    // reset clears any in-flight/stuck state so a click always works again. It
    // is idempotent and never disables links or strips href attributes.
    function reset() {
      if (rafId !== null) {
        window.cancelAnimationFrame(rafId);
        rafId = null;
      }
      if (navTimer !== null) {
        window.clearTimeout(navTimer);
        navTimer = null;
      }
      launchedAt = 0;
      route.dataset.launching = "";
      rocket.classList.remove("flying");
      rocket.classList.add("ready");
      Array.prototype.forEach.call(nodes, function (n) {
        n.classList.remove("targeted");
        n.style.pointerEvents = ""; // defensive: ensure clicks are never blocked
      });
      try {
        placeRocket(rocket, path, restLen, total);
      } catch (e) {
        /* layout only; ignore */
      }
    }

    // resetIfStuck clears a flight that has run far longer than any flight could
    // legitimately take (e.g. a frozen tab) without cancelling a live flight.
    function resetIfStuck() {
      if (route.dataset.launching === "1" && launchedAt && Date.now() - launchedAt > 2500) {
        reset();
      }
    }

    function navigate(url) {
      window.location.href = url;
    }

    function flyTo(targetLen, url) {
      var start = restLen;
      var dist = Math.abs(targetLen - start);
      var duration = Math.min(1500, Math.max(560, dist * 1.5));
      var startTime = null;
      var lastTrail = 0;

      function frame(ts) {
        try {
          if (startTime === null) startTime = ts;
          var t = Math.min(1, (ts - startTime) / duration);
          var len = start + (targetLen - start) * easeInOut(t);
          placeRocket(rocket, path, len, total);
          if (ts - lastTrail > 55) {
            spawnTrail(track, rocket);
            lastTrail = ts;
          }
          if (t < 1) {
            rafId = window.requestAnimationFrame(frame);
          } else {
            rafId = null;
            navigate(url);
          }
        } catch (err) {
          // Defensive: if the animation throws, still honour the click.
          rafId = null;
          navigate(url);
        }
      }

      launchedAt = Date.now();
      rocket.classList.remove("ready");
      rocket.classList.add("flying");
      rafId = window.requestAnimationFrame(frame);
      // Safety net: if rAF stalls (backgrounded/throttled tab) the click must
      // still navigate rather than dead-end. Cleared by reset() on bfcache.
      navTimer = window.setTimeout(function () {
        navTimer = null;
        navigate(url);
      }, duration + 600);
    }

    Array.prototype.forEach.call(nodes, function (node, i) {
      node.addEventListener("click", function (e) {
        var url = node.dataset.url || node.getAttribute("href");
        if (!url) return;
        // No-JS fallback is the <a href>; reduced-motion users navigate straight
        // away without the flight.
        if (reducedMotion()) return;
        e.preventDefault();
        if (route.dataset.launching === "1") return; // a flight is already running
        route.dataset.launching = "1";
        node.classList.add("targeted");
        flyTo(lengths[i], url);
      });
    });

    reset(); // start from a known-clean state
    controllers.push({ reset: reset, resetIfStuck: resetIfStuck });
  }

  function resetRocketRoutes() {
    for (var i = 0; i < controllers.length; i++) {
      try {
        controllers[i].reset();
      } catch (e) {
        /* ignore */
      }
    }
  }

  function resetStuckRocketRoutes() {
    for (var i = 0; i < controllers.length; i++) {
      try {
        controllers[i].resetIfStuck();
      } catch (e) {
        /* ignore */
      }
    }
  }

  function init() {
    controllers = [];
    var routes = document.querySelectorAll(".rocket-route");
    Array.prototype.forEach.call(routes, wireRoute);
  }

  // pageshow fires on normal load AND on back/forward bfcache restore (where
  // DOMContentLoaded does not). Resetting here clears the stale "launching"
  // guard and .flying class that would otherwise dead-end the next click after
  // pressing Back. Reset on every pageshow is safe (event.persisted just tells
  // us it was a bfcache restore).
  window.addEventListener("pageshow", function () {
    resetRocketRoutes();
  });

  // When a tab is restored to visibility, clear any genuinely stuck flight
  // (without cancelling a live one).
  document.addEventListener("visibilitychange", function () {
    if (document.visibilityState === "visible") {
      resetStuckRocketRoutes();
    }
  });

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
