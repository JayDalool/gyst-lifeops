// The + beside the phone tab bar (GYST-178, 13B DD-18). The button carries data-command-open, so
// launcher.js opens the dialog and returns focus to the + when it closes. This file only asks the
// launcher for its Capture mode, through the mode button's own data attribute. The listener is on
// document so that it runs after the launcher's listener on the button, whatever the load order.
//
// It also tells the stylesheet when the on-screen keyboard is up (Codex P2 on PR 270): a fixed bar would
// ride up over the field on iOS, but hiding it on focus alone also hides the only primary navigation for a
// hardware keyboard and leaves it hidden after the keyboard is dismissed. The keyboard counts as up only
// when the visual viewport is much shorter than the layout viewport AND a text-entry control has focus.
// Without visualViewport nothing is ever hidden. The state is data-gyst-kb="open" on <html>.
(function (root) {
  "use strict";
  var KEYBOARD_MIN = 120; // px of height the keyboard takes; address-bar changes are far smaller
  var TEXT_TYPES = ["", "text", "search", "email", "url", "tel", "password", "number"];

  // What kind of control has focus: "text" (takes typing, so shows a keyboard) or "other".
  function focusKind(el) {
    if (!el || !el.tagName) return "other";
    var tag = String(el.tagName).toLowerCase();
    if (tag === "textarea") return "text";
    if (tag === "input") return TEXT_TYPES.indexOf(String(el.type || "").toLowerCase()) >= 0 ? "text" : "other";
    if (tag === "select") return "other";
    return el.isContentEditable ? "text" : "other";
  }

  // The decision: viewport heights, the visual viewport's scale and the focused control's kind in, "open" or
  // "closed" out. Pinch zoom shrinks visualViewport.height by the scale too (Codex P2), so the height is
  // multiplied back before it is compared; a missing or non-finite scale counts as 1. A missing visual
  // viewport height (no support) is always "closed".
  function kbState(layoutHeight, visualHeight, scale, kind) {
    if (typeof layoutHeight !== "number" || typeof visualHeight !== "number" || !(visualHeight > 0)) return "closed";
    var zoom = typeof scale === "number" && isFinite(scale) && scale > 0 ? scale : 1;
    return layoutHeight - visualHeight * zoom > KEYBOARD_MIN && kind === "text" ? "open" : "closed";
  }

  // Safety net for quick add (Codex P2): when the keyboard state goes from open to closed and focus is on the document
  // itself (the hidden + could not take it), and quick add was opened from the +, focus goes back to the +.
  // Never while any control has focus, so a field the keyboard is still serving keeps it.
  function refocusPlus(prev, next, activeIsDocument, fromPlus) {
    return prev === "open" && next === "closed" && activeIsDocument === true && fromPlus === true;
  }

  // All pages search (GYST-187, spec a1 section 7). The query is trimmed, capped at 100 characters (as the
  // server's NormalizeSearchQuery caps it) and lower-cased, then looked for inside each tile's data-search,
  // which the server wrote from the same text its own filter uses. Returns the indexes of the matches; an
  // empty query matches nothing, because the page then shows its groups instead of results.
  function norm(query) {
    return Array.from(String(query == null ? "" : query).trim()).slice(0, 100).join("").trim().toLowerCase();
  }
  function filterPages(query, haystacks) {
    var q = norm(query), found = [];
    if (!q) return found;
    for (var i = 0; i < haystacks.length; i++) if (String(haystacks[i]).indexOf(q) >= 0) found.push(i);
    return found;
  }
  function liveText(count) { return count === 0 ? "No page matches" : count === 1 ? "1 page" : count + " pages"; }

  root.GYSTNav = { kbState: kbState, focusKind: focusKind, refocusPlus: refocusPlus, filterPages: filterPages, norm: norm, liveText: liveText };
  if (!root.document) return;

  var doc = root.document, vv = root.visualViewport;
  var plusButton = null; // the + that opened quick add, while that focus has not moved elsewhere
  var current = "closed";
  function inside(el, tag) { for (; el; el = el.parentNode) if (el.tagName && String(el.tagName).toLowerCase() === tag) return true; return false; }
  function setState(state) {
    if (state === "open") doc.documentElement.setAttribute("data-gyst-kb", "open");
    else doc.documentElement.removeAttribute("data-gyst-kb");
    current = state;
  }
  function update() {
    var prev = current;
    setState(vv ? kbState(root.innerHeight, vv.height, vv.scale, focusKind(doc.activeElement)) : "closed");
    var active = doc.activeElement;
    if (refocusPlus(prev, current, !active || active === doc.body || active === doc.documentElement, !!plusButton) && plusButton.isConnected && plusButton.focus) plusButton.focus();
  }
  if (vv) {
    vv.addEventListener("resize", update);
    // Focus can move while the keyboard stays up; the new focus is read after the move settles.
    doc.addEventListener("focusin", function (event) {
      if (plusButton && event.target !== plusButton && !inside(event.target, "dialog")) plusButton = null;
      update();
    });
    doc.addEventListener("focusout", function () { root.setTimeout(update, 0); });
    // The launcher restores focus to its trigger in the dialog's own close handler. The bar is display: none while the
    // keyboard state is open, and a hidden button cannot take focus, so the state is cleared first. close does not bubble,
    // but a capture listener on the document runs before the dialog's own listeners.
    doc.addEventListener("close", function () { setState("closed"); }, true);
    update();
  }

  // The filter only reads and shows what the server drew; it makes no request and never changes the address.
  // Without script the same field is a GET form, so Enter reloads /more?q= with the same Results group.
  var field = doc.querySelector('[data-launch-filter]');
  if (field) {
    var page = field.closest ? field.closest('.gyst-launch') : null;
    var groups = page && page.querySelector('[data-launch-groups]');
    var results = page && page.querySelector('[data-launch-results]');
    var list = results && results.querySelector('[data-launch-list]');
    var heading = results && results.querySelector('.gyst-launch-sec');
    var count = results && results.querySelector('[data-launch-count]');
    var empty = results && results.querySelector('[data-launch-empty]');
    var shown = page && page.querySelector('[data-launch-query]');
    var live = page && page.querySelector('[data-launch-live]');
    if (groups && results && list && heading && count && empty && shown && live) {
      var liveTimer = 0;
      var showFor = function (query) {
        var q = norm(query), seen = {}, tiles = [];
        Array.prototype.forEach.call(groups.querySelectorAll('.gyst-tile'), function (a) {
          var href = a.getAttribute("href");
          if (!seen[href]) { seen[href] = true; tiles.push(a); } // Pinned repeats a page its group already draws
        });
        var hit = q ? filterPages(q, tiles.map(function (a) { return a.getAttribute("data-search"); })) : [];
        while (list.firstChild) list.removeChild(list.firstChild);
        hit.forEach(function (i) { list.appendChild(tiles[i].parentNode.cloneNode(true)); });
        groups.hidden = q !== "";
        results.hidden = q === "";
        heading.hidden = list.hidden = hit.length === 0;
        empty.hidden = hit.length > 0;
        count.textContent = String(hit.length);
        shown.textContent = String(query).trim();
        root.clearTimeout(liveTimer);
        liveTimer = root.setTimeout(function () { live.textContent = q ? liveText(hit.length) : ""; }, 300);
      };
      field.addEventListener("input", function () { showFor(field.value); });
      field.addEventListener("keydown", function (event) {
        if (event.key === "Escape") {
          field.value = "";
          showFor("");
          event.preventDefault();
        } else if (event.key === "Enter") {
          var first = norm(field.value) ? list.querySelector('a') : null;
          if (first) { event.preventDefault(); root.location.assign(first.getAttribute("href")); }
        }
      });
      if (norm(field.value)) showFor(field.value); // a restored field (back from a page) shows its results
    }
  }

  doc.addEventListener("click", function (event) {
    var plus = event.target && event.target.closest ? event.target.closest("[data-nav-add]") : null;
    if (!plus) return;
    plusButton = plus;
    var capture = doc.querySelector('[data-launcher-mode="capture"]');
    if (capture) capture.click();
  });
})(globalThis);
