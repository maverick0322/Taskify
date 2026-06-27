import { invoke } from "@tauri-apps/api/core";

import { isTauriRuntime } from "@/lib/runtime";

export type StoredSession = {
  accessToken: string;
  refreshToken: string;
};

type TauriStoredSession = {
  access_token: string;
  refresh_token: string;
};

export async function loadStoredSession(): Promise<StoredSession | null> {
  if (!isTauriRuntime()) {
    const accessToken = localStorage.getItem("accessToken");
    const refreshToken = localStorage.getItem("refreshToken");

    if (!accessToken || !refreshToken) {
      return null;
    }

    return { accessToken, refreshToken };
  }

  const session = await invoke<TauriStoredSession | null>("get_secure_session");
  if (!session?.access_token || !session?.refresh_token) {
    return null;
  }

  return {
    accessToken: session.access_token,
    refreshToken: session.refresh_token,
  };
}

export async function persistSession(session: StoredSession): Promise<void> {
  if (!isTauriRuntime()) {
    localStorage.setItem("accessToken", session.accessToken);
    localStorage.setItem("refreshToken", session.refreshToken);
    return;
  }

  await invoke("set_secure_session", {
    accessToken: session.accessToken,
    refreshToken: session.refreshToken,
  });
}

export async function clearStoredSession(): Promise<void> {
  if (!isTauriRuntime()) {
    localStorage.removeItem("accessToken");
    localStorage.removeItem("refreshToken");
    return;
  }

  await invoke("clear_secure_session");
}
