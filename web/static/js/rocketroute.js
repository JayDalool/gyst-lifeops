(function () {
  "use strict";

  // viewBox of the route SVG; node %-positions are derived from path points in
  // these user units. preserveAspectRatio="meet" keeps the scale uniform, so the
  // tangent angle needs no anisotropy correction.
  var VB_W = 1000;
  var VB_H = 300;

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

    placeRocket(rocket, path, restLen, total);
    rocket.classList.add("ready");

    function flyTo(targetLen, url) {
      var start = restLen;
      var dist = Math.abs(targetLen - start);
      var duration = Math.min(1500, Math.max(560, dist * 1.5));
      var startTime = null;
      var lastTrail = 0;

      function frame(ts) {
        if (startTime === null) startTime = ts;
        var t = Math.min(1, (ts - startTime) / duration);
        var len = start + (targetLen - start) * easeInOut(t);
        placeRocket(rocket, path, len, total);
        if (ts - lastTrail > 55) {
          spawnTrail(track, rocket);
          lastTrail = ts;
        }
        if (t < 1) {
          window.requestAnimationFrame(frame);
        } else {
          window.location.href = url;
        }
      }
      rocket.classList.remove("ready");
      rocket.classList.add("flying");
      window.requestAnimationFrame(frame);
    }

    Array.prototype.forEach.call(nodes, function (node, i) {
      node.addEventListener("click", function (e) {
        var url = node.dataset.url || node.getAttribute("href");
        if (!url) return;
        // No-JS fallback is the <a href>; reduced-motion users also navigate
        // straight away without the flight.
        if (reducedMotion()) return;
        e.preventDefault();
        if (route.dataset.launching === "1") return; // already flying
        route.dataset.launching = "1";
        node.classList.add("targeted");
        flyTo(lengths[i], url);
      });
    });
  }

  function init() {
    var routes = document.querySelectorAll(".rocket-route");
    Array.prototype.forEach.call(routes, wireRoute);
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
