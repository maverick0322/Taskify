import { invoke } from "@tauri-apps/api/core";

import { isTauriRuntime } from "@/lib/runtime";

export type StoredSession = {
  accessToken: string;
  refreshToken: string;
  remoteAccessToken?: string;
  remoteRefreshToken?: string;
};

type TauriStoredSession = {
  access_token: string;
  refresh_token: string;
  remote_access_token?: string | null;
  remote_refresh_token?: string | null;
};

export async function loadStoredSession(): Promise<StoredSession | null> {
  if (!isTauriRuntime()) {
    const accessToken = localStorage.getItem("accessToken");
    const refreshToken = localStorage.getItem("refreshToken");

    if (!accessToken || !refreshToken) {
      return null;
    }

    return {
      accessToken,
      refreshToken,
      remoteAccessToken: localStorage.getItem("remoteAccessToken") || undefined,
      remoteRefreshToken:
        localStorage.getItem("remoteRefreshToken") || undefined,
    };
  }

  const session = await invoke<TauriStoredSession | null>("get_secure_session");
  if (!session?.access_token || !session?.refresh_token) {
    return null;
  }

  return {
    accessToken: session.access_token,
    refreshToken: session.refresh_token,
    remoteAccessToken: session.remote_access_token ?? undefined,
    remoteRefreshToken: session.remote_refresh_token ?? undefined,
  };
}

export async function persistSession(session: StoredSession): Promise<void> {
  if (!isTauriRuntime()) {
    localStorage.setItem("accessToken", session.accessToken);
    localStorage.setItem("refreshToken", session.refreshToken);
    if (session.remoteAccessToken && session.remoteRefreshToken) {
      localStorage.setItem("remoteAccessToken", session.remoteAccessToken);
      localStorage.setItem("remoteRefreshToken", session.remoteRefreshToken);
    } else {
      localStorage.removeItem("remoteAccessToken");
      localStorage.removeItem("remoteRefreshToken");
    }
    return;
  }

  await invoke("set_secure_session", {
    accessToken: session.accessToken,
    refreshToken: session.refreshToken,
    remoteAccessToken: session.remoteAccessToken ?? null,
    remoteRefreshToken: session.remoteRefreshToken ?? null,
  });
}

export async function clearStoredSession(): Promise<void> {
  if (!isTauriRuntime()) {
    localStorage.removeItem("accessToken");
    localStorage.removeItem("refreshToken");
    localStorage.removeItem("remoteAccessToken");
    localStorage.removeItem("remoteRefreshToken");
    return;
  }

  await invoke("clear_secure_session");
}
