import { isTauriRuntime } from "@/lib/runtime";
import { apiRequest } from "@/services/api";

export async function forceSync(): Promise<{ synced: boolean }> {
  return apiRequest<{ synced: boolean }>("/sync/force", {
    method: "POST",
  });
}

export async function checkpointSQLite(): Promise<{ checkpointed: boolean }> {
  return apiRequest<{ checkpointed: boolean }>("/system/sqlite/checkpoint", {
    method: "POST",
  });
}

export async function connectDesktopSyncSession(credentials: {
  email: string;
  password: string;
}): Promise<void> {
  if (!isTauriRuntime()) {
    return;
  }

  await apiRequest<{ connected: boolean }>("/sync/session/login", {
    method: "POST",
    body: JSON.stringify(credentials),
  });
}

export async function clearDesktopSyncSession(): Promise<void> {
  if (!isTauriRuntime()) {
    return;
  }

  await apiRequest<{ cleared: boolean }>("/sync/session/logout", {
    method: "POST",
  });
}

export async function purgeDesktopSQLite(): Promise<void> {
  if (!isTauriRuntime()) {
    return;
  }

  await apiRequest<{ purged: boolean }>("/system/sqlite/purge", {
    method: "POST",
  });
}
