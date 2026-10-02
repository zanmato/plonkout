/**
 * Keeps the page behind a modal still. Modals can stack, so the page only
 * scrolls again when the last one lets go.
 */
let holders = 0;

function apply() {
  const overflow = holders > 0 ? "hidden" : "";
  document.documentElement.style.overflow = overflow;
  document.body.style.overflow = overflow;
}

/** Locks the page and returns the function that releases this hold. */
export function lockScroll(): () => void {
  holders++;
  apply();
  let released = false;
  return () => {
    if (released) return;
    released = true;
    holders--;
    apply();
  };
}
