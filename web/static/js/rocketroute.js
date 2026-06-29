(function () {
  "use strict";

  // viewBox of the route SVG; node %-positions are derived from path points in
  // these user units. preserveAspectRatio="meet" keeps the scale uniform, so the
  // tangent angle needs no anisotropy correction.
  var VB_W = 1000;
  var VB_H = 300;

  // The flight is always shorter than this, so the FALLBACK_MS guard below only
  // ever fires when rAF genuinely stalls (backgrounded/throttled tab) — never
  // truncating a healthy animation.
  var MAX_FLIGHT_MS = 760;
  var MIN_FLIGHT_MS = 520;
  // Hard safety net: if nothing has navigated this long after a click, force it.
  // Must exceed MAX_FLIGHT_MS so good flights complete on their own first.
  var FALLBACK_MS = 900;

  // One controller per .rocket-route instance, keyed off the route element so a
  // delegated click can always find fresh, never-stale state. State (timers,
  // navigated flag) lives on the controller — never globally, never permanent.
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

  function buildController(route) {
    var track = route.querySelector(".route-track");
    var path = route.querySelector(".route-base");
    var rocket = route.querySelector(".rocket");
    var nodeList = route.querySelectorAll(".route-node");
    if (!track || !path || !rocket || !nodeList.length) return null;

    var nodes = Array.prototype.slice.call(nodeList);
    // Old browsers without getTotalLength still navigate — the click handler
    // detects this and falls straight through to the href.
    var canAnimate = typeof path.getTotalLength === "function";
    var total = canAnimate ? path.getTotalLength() : 0;

    // Lay each node precisely on the path. getPointAtLength returns user-space
    // coords independent of rendered pixel size, so these percentages are
    // resolution-stable — no recompute on resize needed.
    var lengths = [];
    var restLen = 0;
    if (canAnimate) {
      nodes.forEach(function (node, i) {
        var pct = parseFloat(node.dataset.pathPct || "0");
        var len = (total * pct) / 100;
        lengths[i] = len;
        var pt = path.getPointAtLength(len);
        node.style.left = pctLeft(pt) + "%";
        node.style.top = pctTop(pt) + "%";
        if (node.classList.contains("is-current")) restLen = len;
      });
    }

    var rafId = null;
    var fallbackTimer = null;
    var navigated = false;

    function clearTimers() {
      if (rafId !== null) {
        window.cancelAnimationFrame(rafId);
        rafId = null;
      }
      if (fallbackTimer !== null) {
        window.clearTimeout(fallbackTimer);
        fallbackTimer = null;
      }
    }

    // reset clears every scrap of in-flight/stuck state so a click always works
    // again. It is idempotent, never disables links, and never strips href.
    // Called on each click (self-healing), on pagehide (before bfcache freeze so
    // no timer thaws and navigates the user back forward), and on pageshow.
    function reset() {
      clearTimers();
      navigated = false;
      route.dataset.launching = "";
      rocket.classList.remove("flying", "launching");
      rocket.classList.add("ready");
      nodes.forEach(function (n) {
        n.classList.remove("targeted", "flying", "launching");
        delete n.dataset.launching;
        n.style.pointerEvents = ""; // defensive: never leave clicks blocked
      });
      if (canAnimate) {
        try {
          placeRocket(rocket, path, restLen, total);
        } catch (e) {
          /* layout only; ignore */
        }
      }
    }

    function navigate(url) {
      if (navigated) return; // navigate exactly once per click
      navigated = true;
      window.location.href = url;
    }

    function flyTo(index, url) {
      var targetLen = lengths[index];
      var start = restLen;
      var dist = Math.abs(targetLen - start);
      var duration = Math.min(MAX_FLIGHT_MS, Math.max(MIN_FLIGHT_MS, dist * 1.5));
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
          // Defensive: if the animation throws mid-flight, still honour the click.
          rafId = null;
          navigate(url);
        }
      }

      rocket.classList.remove("ready");
      rocket.classList.add("flying");
      rafId = window.requestAnimationFrame(frame);
    }

    // handleClick is self-healing: it NEVER ignores a click because a previous
    // flight is "still running". It always resets first, then starts fresh.
    function handleClick(node, e) {
      var index = nodes.indexOf(node);
      var url = node.dataset.url || node.getAttribute("href");
      if (!url) return; // href missing: do nothing, let nothing happen
      // No-JS / reduced-motion path is the native <a href>: don't preventDefault,
      // let the browser navigate normally.
      if (reducedMotion() || !canAnimate || index < 0) return;

      e.preventDefault();
      // Wipe any stale state from a prior click or a bad bfcache restore.
      reset();
      route.dataset.launching = "1";
      node.classList.add("targeted");

      // Outer safety net: guarantees navigation even if rAF never runs.
      fallbackTimer = window.setTimeout(function () {
        fallbackTimer = null;
        navigate(url);
      }, FALLBACK_MS);

      try {
        flyTo(index, url);
      } catch (err) {
        // Animation could not start — navigate immediately rather than dead-end.
        navigate(url);
      }
    }

    reset(); // start from a known-clean state
    var controller = { route: route, reset: reset, handleClick: handleClick };
    controllers.push(controller);
    return controller;
  }

  function controllerForRoute(route) {
    for (var i = 0; i < controllers.length; i++) {
      if (controllers[i].route === route) return controllers[i];
    }
    return null;
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

  // Event delegation: a single document-level listener means there is no
  // per-node closure state that can go stale across a bfcache restore. The node
  // is resolved fresh from the DOM on every click.
  function onDocumentClick(e) {
    var node = e.target.closest
      ? e.target.closest("[data-route-node], .route-node")
      : null;
    if (!node) return;
    var routeEl = node.closest(".rocket-route");
    if (!routeEl) return;
    var controller = controllerForRoute(routeEl);
    if (!controller) return; // route not wired (e.g. very old browser): let <a> work
    controller.handleClick(node, e);
  }

  // Pause the route's continuous CSS animations (flow pulse + node rings) while
  // it is scrolled off-screen, so nothing repaints when the user can't see it.
  // Cheap, transform/opacity-only animations resume the instant it returns.
  function observeVisibility(routes) {
    if (typeof IntersectionObserver !== "function") return;
    var io = new IntersectionObserver(
      function (entries) {
        entries.forEach(function (entry) {
          entry.target.classList.toggle("route-paused", !entry.isIntersecting);
        });
      },
      { rootMargin: "120px" }
    );
    Array.prototype.forEach.call(routes, function (route) {
      io.observe(route);
    });
  }

  function init() {
    controllers = [];
    var routes = document.querySelectorAll(".rocket-route");
    Array.prototype.forEach.call(routes, buildController);
    observeVisibility(routes);
  }

  // --- Top-level listeners (registered immediately, never inside
  // DOMContentLoaded, so a bfcache restore can never miss them) ---

  document.addEventListener("click", onDocumentClick, false);

  // pagehide fires right before the page is frozen into the bfcache. Resetting
  // here clears the launching flag, .flying class, and — critically — every
  // pending timer, so nothing thaws on restore and yanks the user forward.
  window.addEventListener("pagehide", function () {
    resetRocketRoutes();
  });

  // pageshow fires on normal load AND on back/forward bfcache restore (where
  // DOMContentLoaded does not). Resetting again here makes the route self-heal
  // even if it returned from the cache in a bad state, so the next click works.
  window.addEventListener("pageshow", function () {
    resetRocketRoutes();
  });

  // When a tab returns to visibility, clear any genuinely stuck flight.
  document.addEventListener("visibilitychange", function () {
    if (document.visibilityState === "visible") {
      resetRocketRoutes();
    }
  });

  // Debug-safe self-heal check (manual): in the console run
  //   __rocketRouteSelfTest()
  // It forces a route into the old "stuck" state (dataset.launching="1" +
  // .flying), then asserts the next simulated click still resets and proceeds.
  window.__rocketRouteSelfTest = function () {
    if (!controllers.length) return "no rocket routes on this page";
    var c = controllers[0];
    var route = c.route;
    var node = route.querySelector(".route-node");
    if (!node) return "no route node";
    var rocket = route.querySelector(".rocket");
    var url = node.dataset.url || node.getAttribute("href");

    // Poison the state exactly the way a bad bfcache restore would.
    route.dataset.launching = "1";
    node.dataset.launching = "1";
    if (rocket) rocket.classList.add("flying");

    // reset() is the first thing handleClick does on every click, so passing it
    // proves a stale/stuck route can never block the next click.
    c.reset();
    var healed =
      route.dataset.launching === "" &&
      !node.dataset.launching &&
      (!rocket || !rocket.classList.contains("flying"));
    return healed
      ? "PASS: stale state cleared; next click would navigate to " + url
      : "FAIL: state still stuck";
  };

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
