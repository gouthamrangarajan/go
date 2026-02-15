function focusTrap(querySelector) {
  const activeElementsSelector =
    'a[href]:not(:disabled), button:not(:disabled), input:not(:disabled):not([type="hidden"]), select:not(:disabled), textarea:not(:disabled), [tabindex]:not([tabindex="-1"])';
  const abortController = new AbortController();
  const abortSignal = abortController.signal;
  const container = document.querySelector(querySelector);
  if (!container) {
    return abortController;
  }
  const focusableElements = container.querySelectorAll(activeElementsSelector);
  if (focusableElements.length === 0 && container.focus) {
    container.focus();
  } else {
    focusableElements[0].focus();
  }
  window.addEventListener(
    "keydown",
    function (event) {
      if (event.key === "Tab") {
        const container = document.querySelector(querySelector);
        if (!container) {
          return;
        }
        const focusableElements = container.querySelectorAll(
          activeElementsSelector
        );
        if (focusableElements.length === 0) {
          if (container.focus) {
            container.focus();
          }
          event.preventDefault();
          return;
        }
        const firstElement = focusableElements[0];
        const lastElement = focusableElements[focusableElements.length - 1];
        if (event.shiftKey) {
          // Shift + Tab
          if (document.activeElement === firstElement) {
            event.preventDefault();
            lastElement.focus();
            return;
          }
        } else {
          // Tab
          if (document.activeElement === lastElement) {
            event.preventDefault();
            firstElement.focus();
            return;
          }
        }
        if (!container.contains(document.activeElement)) {
          event.preventDefault();
          firstElement.focus();
          return;
        }
      }
    },
    { signal: abortSignal }
  );
  window.addEventListener(
    "focus",
    function (event) {
      const container = document.querySelector(querySelector);
      if (!container) {
        return;
      }
      const focusableElements = container.querySelectorAll(
        activeElementsSelector
      );
      if (focusableElements.length === 0) {
        event.preventDefault();
        container.focus();
        return;
      }
      const firstElement = focusableElements[0];
      event.preventDefault();
      firstElement.focus();
    },
    { signal: abortSignal }
  );
  return abortController;
}
