(function () {
  "use strict";

  // ----- Mission completion flourish -------------------------------------
  // Completion is a real POST that reloads the page, so this is just a brief
  // client-side moment of feedback before the navigation happens.
  function wireMissionFlourish() {
    var forms = document.querySelectorAll("form[data-mission-complete]");
    Array.prototype.forEach.call(forms, function (form) {
      form.addEventListener("submit", function () {
        var card = form.closest(".mission-node");
        if (card) card.classList.add("is-completing");
      });
    });
  }

  // ----- Campaign star map -----------------------------------------------
  function wireStarmap() {
    var map = document.querySelector(".starmap");
    if (!map) return; // page has no map; nothing to do

    // Center the active destination on load (instant, no scroll animation).
    var active = map.querySelector("[data-map-active]");
    if (active) {
      var prev = map.style.scrollBehavior;
      map.style.scrollBehavior = "auto";
      var target = active.offsetLeft - map.clientWidth / 2 + active.clientWidth / 2;
      map.scrollLeft = Math.max(0, target);
      map.style.scrollBehavior = prev || "";
    }

    // Smoothly pan to a node when it is focused or clicked.
    function panTo(node) {
      var target = node.offsetLeft - map.clientWidth / 2 + node.clientWidth / 2;
      map.scrollTo({ left: Math.max(0, target), behavior: "smooth" });
    }
    Array.prototype.forEach.call(map.querySelectorAll(".map-node"), function (node) {
      node.addEventListener("focus", function () {
        panTo(node);
      });
    });

    // Drag to pan with the mouse/pointer.
    var dragging = false;
    var startX = 0;
    var startScroll = 0;
    var moved = false;

    map.addEventListener("pointerdown", function (e) {
      // ignore clicks on interactive children so links still work
      dragging = true;
      moved = false;
      startX = e.clientX;
      startScroll = map.scrollLeft;
      map.classList.add("is-panning");
    });
    map.addEventListener("pointermove", function (e) {
      if (!dragging) return;
      var delta = e.clientX - startX;
      if (Math.abs(delta) > 4) moved = true;
      map.scrollLeft = startScroll - delta;
    });
    function endDrag() {
      dragging = false;
      map.classList.remove("is-panning");
    }
    map.addEventListener("pointerup", endDrag);
    map.addEventListener("pointerleave", endDrag);
    // Suppress the click that follows a real drag so we don't navigate.
    map.addEventListener(
      "click",
      function (e) {
        if (moved) {
          e.preventDefault();
          e.stopPropagation();
          moved = false;
        }
      },
      true
    );

    // Keyboard panning when the map container itself is focused.
    map.addEventListener("keydown", function (e) {
      if (e.target !== map) return;
      var step = map.clientWidth * 0.6;
      if (e.key === "ArrowRight") {
        map.scrollBy({ left: step, behavior: "smooth" });
        e.preventDefault();
      } else if (e.key === "ArrowLeft") {
        map.scrollBy({ left: -step, behavior: "smooth" });
        e.preventDefault();
      }
    });
  }

  // ----- Roadmap Map/List toggle ----------------------------------------
  // CSS handles this via :has(); this adds a class-based fallback for any
  // engine without :has() support and keeps both in sync.
  function wireViewToggle() {
    var views = document.querySelector(".roadmap-views");
    if (!views) return;
    var radios = views.querySelectorAll(".view-radio");
    function sync() {
      var list = document.getElementById("view-list");
      views.classList.toggle("show-list", !!(list && list.checked));
    }
    Array.prototype.forEach.call(radios, function (radio) {
      radio.addEventListener("change", sync);
    });
    sync();
  }

  function init() {
    wireMissionFlourish();
    wireStarmap();
    wireViewToggle();
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
