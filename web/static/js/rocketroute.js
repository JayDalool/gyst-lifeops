(function () {
  "use strict";

  function reducedMotion() {
    return (
      window.matchMedia &&
      window.matchMedia("(prefers-reduced-motion: reduce)").matches
    );
  }

  function wireRoute(route) {
    var rocket = route.querySelector(".rocket");
    var nodes = route.querySelectorAll(".route-node");
    if (!rocket || !nodes.length) return;

    Array.prototype.forEach.call(nodes, function (node) {
      node.addEventListener("click", function (e) {
        var url = node.dataset.url || node.getAttribute("href");
        if (!url) return;

        // No-JS fallback is the <a href>; here we also bail to a normal
        // navigation when the user prefers reduced motion.
        if (reducedMotion()) return;

        e.preventDefault();
        if (route.dataset.launching === "1") return; // already flying
        route.dataset.launching = "1";

        // Visual centers: a node/rocket positioned with left/top + a
        // translate(-50%,-50%) has its visual centre at offsetLeft/offsetTop.
        var dx = node.offsetLeft - rocket.offsetLeft;
        var dy = node.offsetTop - rocket.offsetTop;

        rocket.classList.add("launching");
        // Keep the -50% centring so the rocket stays centred while it moves.
        rocket.style.transform =
          "translate(calc(-50% + " + dx + "px), calc(-50% + " + dy + "px))";
        node.classList.add("targeted");

        window.setTimeout(function () {
          window.location.href = url;
        }, 760);
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
