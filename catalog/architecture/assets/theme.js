/*
 * Pre-paint theme bootstrap.
 *
 * Runs inline in <head>, before the body renders, so a reader who chose a
 * theme never sees a flash of the other one. Kept separate (and tiny) from the
 * main script for exactly that reason.
 */
(function () {
  try {
    var stored = localStorage.getItem("cqrs-architecture-theme");
    if (stored) {
      document.documentElement.setAttribute("data-theme", stored);
    }
  } catch (error) {
    /* storage can be unavailable (private mode, file://); the system theme stands. */
  }
})();
