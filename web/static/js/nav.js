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

  // The decision: viewport heights and the focused control's kind in, "open" or "closed" out.
  // A missing visual viewport height (no support) is always "closed".
  function kbState(layoutHeight, visualHeight, kind) {
    if (typeof layoutHeight !== "number" || typeof visualHeight !== "number" || !(visualHeight > 0)) return "closed";
    return layoutHeight - visualHeight > KEYBOARD_MIN && kind === "text" ? "open" : "closed";
  }
  root.GYSTNav = { kbState: kbState, focusKind: focusKind };
  if (!root.document) return;

  var doc = root.document, vv = root.visualViewport;
  function update() {
    var state = vv ? kbState(root.innerHeight, vv.height, focusKind(doc.activeElement)) : "closed";
    if (state === "open") doc.documentElement.setAttribute("data-gyst-kb", "open");
    else doc.documentElement.removeAttribute("data-gyst-kb");
  }
  if (vv) {
    vv.addEventListener("resize", update);
    // Focus can move while the keyboard stays up; the new focus is read after the move settles.
    doc.addEventListener("focusin", update);
    doc.addEventListener("focusout", function () { root.setTimeout(update, 0); });
    update();
  }

  doc.addEventListener("click", function (event) {
    var plus = event.target && event.target.closest ? event.target.closest("[data-nav-add]") : null;
    if (!plus) return;
    var capture = doc.querySelector('[data-launcher-mode="capture"]');
    if (capture) capture.click();
  });
})(globalThis);
