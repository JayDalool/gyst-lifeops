// Early, guarded theme initialization. One owner for both old and new UI tokens.
// Appearance (DD-1): the stored mode is system, light or dark; System follows the OS live.
(function () {
  "use strict";
  var key = "lifeops-theme";
  var media = null;
  try { media = window.matchMedia("(prefers-color-scheme: dark)"); } catch (_) {}
  var mode = "system";
  function normalize(value) { return value === "light" || value === "dark" ? value : "system"; }
  function resolve(value) { return value === "system" ? (media && media.matches ? "dark" : "light") : value; }
  function apply(value, persist) {
    mode = normalize(value);
    var theme = resolve(mode);
    var root = document.documentElement;
    // Legacy dark styles use absence of data-theme; new components use an explicit value.
    if (theme === "light") root.dataset.theme = "light";
    else delete root.dataset.theme;
    root.dataset.gystTheme = theme;
    var meta = document.querySelector('meta[name="theme-color"]');
    if (meta) meta.content = theme === "dark" ? "#0f172a" : "#f8fafc";
    if (persist) { try { localStorage.setItem(key, mode); } catch (_) {} }
    document.querySelectorAll("[data-theme-choice]").forEach(function (choice) { choice.setAttribute("aria-pressed", String(choice.dataset.themeChoice === mode)); });
    document.dispatchEvent(new Event("gyst:themechange"));
  }
  var stored;
  try { stored = localStorage.getItem(key); } catch (_) {}
  apply(stored, false);
  try { if (localStorage.getItem("lifeops-motion") === "reduced") document.documentElement.dataset.motion = "reduced"; } catch (_) {}
  function osChanged() { if (mode === "system") apply("system", false); }
  if (media) {
    if (media.addEventListener) media.addEventListener("change", osChanged);
    else if (media.addListener) media.addListener(osChanged);
  }
  document.addEventListener("DOMContentLoaded", function () {
    document.querySelectorAll("[data-theme-choice]").forEach(function (choice) {
      choice.setAttribute("aria-pressed", String(choice.dataset.themeChoice === mode));
      choice.addEventListener("click", function () { apply(choice.dataset.themeChoice, true); });
    });
  });
  window.addEventListener("storage", function (event) { if (event.key === key || event.key === null) apply(event.newValue, false); });
})();
