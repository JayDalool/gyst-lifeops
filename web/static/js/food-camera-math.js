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
  // prefillEst is the nutrient shown for an AI estimate that holds `v` for
  // `estG` grams, at the grams `g` as typed. Same operation order as the Go
  // side (services.CameraScale): (v*g)/estG, then to 0.1. GYST-180/GYST-192:
  // the server compares what the form posts with its own figure to the last
  // digit, so this must not be rewritten as v * (g / estG).
  function prefillEst(v, estG, g) {
    return v == null || !(estG > 0) || !(g > 0) ? '' : num(v * g / estG);
  }
  // estimateHolds is the server's "still the AI's estimate" rule for a typed
  // field: the number typed (as a number, so 350 and 350.0 agree) equals the
  // estimate scaled to the grams and rounded to 0.1, within 1e-6 like
  // services.nutrientsEqual.
  function estimateHolds(text, v, estG, g) {
    var typed = parseAmount(text);
    if (v == null || !(estG > 0) || !(g > 0) || isNaN(typed)) return false;
    return Math.abs(typed - Math.round(v * g / estG * 10) / 10) <= 1e-6;
  }
  // whole is a calorie figure as the server prints it (formatThousands):
  // rounded to a whole number with thousands commas.
  function whole(v) {
    var r = Math.round(v);
    if (r === 0) r = 0; // never "-0"
    var s = String(Math.abs(r)).replace(/\B(?=(\d{3})+(?!\d))/g, ',');
    return r < 0 ? '\u2212' + s : s;
  }
  // estimateNote is the day-total label of the Go side (handlers.estimateNote):
  // "Includes AI estimates: 2 items, about 410 kcal"; kcal null leaves it out.
  function estimateNote(items, kcal) {
    if (!(items > 0)) return '';
    var n = 'Includes AI estimates: ' + items + (items === 1 ? ' item' : ' items');
    if (kcal != null) n += ', about ' + whole(kcal) + ' kcal';
    return n;
  }
  // parseAmount reads a typed amount: digits with an optional decimal part,
  // comma or point. NaN for anything else (including blank).
  function parseAmount(v) {
    var s = String(v).trim().replace(',', '.');
    return /^\d+(\.\d+)?$/.test(s) ? parseFloat(s) : NaN;
  }
  // typedRatio is the per-gram ratio of a nutrient the owner typed, or null
  // when there are no grams yet or the text is not a valid amount.
  function typedRatio(text, g) {
    var v = parseAmount(text);
    return g > 0 && v >= 0 ? v / g : null;
  }
  // captureRatios adds a ratio for each field that has a typed value but none
  // yet (nutrients entered before the grams), using the grams as they are now,
  // i.e. before a stepper or the portion slider changes them. Existing ratios
  // are kept. GYST-179: never done on each keystroke of the grams, which would
  // fix the ratio at "1" of "150".
  function captureRatios(ratio, texts, g) {
    var out = {}, k;
    for (k in ratio) out[k] = ratio[k];
    if (!(g > 0)) return out;
    for (k in texts) {
      if (out[k] == null) { var r = typedRatio(texts[k], g); if (r != null) out[k] = r; }
    }
    return out;
  }
  // followValues is what the ratio fields show at these grams, only for
  // fields that have a ratio: a field without one is left as the owner typed it.
  function followValues(ratio, g) {
    var out = {};
    if (!(g > 0)) return out;
    for (var k in ratio) if (ratio[k] != null) out[k] = num(ratio[k] * g);
    return out;
  }
  // latest hands out request ids so only the newest answer is used; invalidate
  // drops every answer still in flight.
  function latest() {
    var n = 0;
    return {next: function () { return ++n; }, invalidate: function () { n++; }, isCurrent: function (id) { return id === n; }};
  }
  var api = {num: num, prefill: prefill, prefillEst: prefillEst, estimateHolds: estimateHolds, whole: whole, estimateNote: estimateNote, parseAmount: parseAmount, typedRatio: typedRatio, captureRatios: captureRatios, followValues: followValues, latest: latest};
  if (typeof module !== 'undefined' && module.exports) module.exports = api;
  else root.FoodCameraMath = api;
})(typeof self !== 'undefined' ? self : this);
