// GYST-188 (a3 section 7): the show-password button, and nothing else. Nothing is stored or sent;
// the security review reads this file line by line, so it stays this short.
document.addEventListener("DOMContentLoaded", function () {
  var button = document.querySelector("[data-signin-show]");
  var input = document.getElementById("password");
  if (!button || !input) return;
  button.hidden = false;
  button.addEventListener("click", function () {
    var show = input.type === "password";
    input.type = show ? "text" : "password";
    button.setAttribute("aria-pressed", String(show));
  });
});
