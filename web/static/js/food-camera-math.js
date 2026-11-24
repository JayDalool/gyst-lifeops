// Pure maths of the food camera review (Issue #211, GYST-179). No DOM, so the
// same file runs in the browser (window.FoodCameraMath) and under node, where a
// Go test checks it against internal/handlers/testdata/camera_rounding.json.
// The server rounds grams to 0.1 and recomputes a database food's nutrition; this
// is the script's side of that contract, and the table is what keeps them equal.
(function (root) {
  'use strict';
  // num shows a value to 0.1 (what the review fields and the server both use).
  function num(v) {
    return v == null || !isFinite(v) ? '' : String(Math.round(v * 10) / 10);
  }
  // prefill is the nutrient shown for a food with `per` per 100 g at the grams
  // as typed (not rounded). Same operation order as the Go side: (per*g)/100.
  function prefill(per, g) {
    return per == null ? '' : num(per * g / 100);
  }
  var api = {num: num, prefill: prefill};
  if (typeof module !== 'undefined' && module.exports) module.exports = api;
  else root.FoodCameraMath = api;
})(typeof self !== 'undefined' ? self : this);
