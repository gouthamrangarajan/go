function focusTrap(querySelector) {
  const activeElementsSelector =
    'a[href]:not(:disabled), button:not(:disabled), select:not(:disabled), textarea:not(:disabled), input:not(:disabled):not([type="hidden"]), [tabindex]:not([tabindex="-1"])';

  const abortController = new AbortController();
  const container = document.querySelector(querySelector);
  if (!container) {
    return abortController;
  }
  const focusableElements = container.querySelectorAll(activeElementsSelector);
  if (focusableElements.length === 0) {
    if (container.focus) {
      container.focus();
    }
  } else {
    focusableElements[0].focus();
  }
  window.addEventListener(
    "keydown",
    (event) => {
      if (event.key == "Tab") {
        const container = document.querySelector(querySelector);
        if (!container) {
          return;
        }
        const focusableElements = container.querySelectorAll(
          activeElementsSelector
        );
        if (focusableElements.length === 0) {
          if (container.focus) {
            event.preventDefault();
            container.focus();
          }
          return;
        } else {
          const firstElement = focusableElements[0];
          const lastElement = focusableElements[focusableElements.length - 1];
          if (event.shiftKey) {
            if (document.activeElement === firstElement) {
              event.preventDefault();
              lastElement.focus();
            }
          } else {
            if (document.activeElement === lastElement) {
              event.preventDefault();
              firstElement.focus();
            }
          }
        }
      }
    },
    { signal: abortController.signal }
  );
  window.addEventListener(
    "focus",
    (event) => {
      const container = document.querySelector(querySelector);
      if (!container) {
        return;
      }
      const focusableElements = container.querySelectorAll(
        activeElementsSelector
      );
      if (container.focus) {
        event.preventDefault();
        container.focus();
        return;
      } else if (focusableElements.length > 0) {
        event.preventDefault();
        focusableElements[0].focus();
        return;
      }
    },
    { signal: abortController.signal, capture: true }
  );
  return abortController;
}
