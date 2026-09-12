// Early, guarded theme initialization. One owner for both old and new UI tokens.
(function () {
  "use strict";
  var key = "lifeops-theme";
  function apply(theme, persist) {
    theme = theme === "dark" ? "dark" : "light";
    var root = document.documentElement;
    // Legacy dark styles use absence of data-theme; new components use an explicit value.
    if (theme === "light") root.dataset.theme = "light";
    else delete root.dataset.theme;
    root.dataset.gystTheme = theme;
    var meta = document.querySelector('meta[name="theme-color"]');
    if (meta) meta.content = theme === "dark" ? "#0f172a" : "#f8fafc";
    if (persist) { try { localStorage.setItem(key, theme); } catch (_) {} }
    document.querySelectorAll("[data-theme-toggle]").forEach(function (toggle) { toggle.checked = theme === "light"; });
    document.dispatchEvent(new Event("gyst:themechange"));
  }
  var stored;
  try { stored = localStorage.getItem(key); } catch (_) {}
  apply(stored, false);
  document.addEventListener("DOMContentLoaded", function () {
    document.querySelectorAll("[data-theme-toggle]").forEach(function (toggle) {
      toggle.checked = document.documentElement.dataset.gystTheme !== "dark";
      toggle.addEventListener("change", function () { apply(toggle.checked ? "light" : "dark", true); });
    });
  });
  window.addEventListener("storage", function (event) { if (event.key === key) apply(event.newValue, false); });
})();
