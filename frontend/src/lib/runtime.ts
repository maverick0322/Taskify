declare global {
  interface Window {
    __TAURI_INTERNALS__?: unknown
  }
}

export function isTauriRuntime() {
  if (typeof window === "undefined") {
    return false;
  }

  if (Boolean(window.__TAURI_INTERNALS__)) {
    return true;
  }

  return (
    window.location.protocol === "tauri:" ||
    window.location.hostname === "tauri.localhost"
  );
}
