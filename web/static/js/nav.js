// The + beside the phone tab bar (GYST-178, 13B DD-18). The button carries data-command-open, so
// launcher.js opens the dialog and returns focus to the + when it closes. This file only asks the
// launcher for its Capture mode, through the mode button's own data attribute. The listener is on
// document so that it runs after the launcher's listener on the button, whatever the load order.
(function () {
  "use strict";
  document.addEventListener("click", function (event) {
    var plus = event.target && event.target.closest ? event.target.closest("[data-nav-add]") : null;
    if (!plus) return;
    var capture = document.querySelector('[data-launcher-mode="capture"]');
    if (capture) capture.click();
  });
})();
