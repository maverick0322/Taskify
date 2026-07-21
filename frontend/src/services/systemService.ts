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
}): Promise<{
  initialSyncCompleted?: boolean;
}> {
  if (!isTauriRuntime()) {
    return {};
  }

  return apiRequest<{
    connected: boolean;
    initialSyncCompleted?: boolean;
  }>("/sync/session/login", {
    method: "POST",
    body: JSON.stringify(credentials),
    timeoutMs: 60000,
    timeoutMessage:
      "La sincronización inicial está tardando demasiado. Intenta de nuevo.",
  });
}

export async function restoreDesktopSyncSession(): Promise<{
  restored?: boolean;
  initialSyncCompleted?: boolean;
  syncState?: "connected" | "pending" | "offline" | "error";
}> {
  if (!isTauriRuntime()) {
    return {};
  }

  return apiRequest<{ restored: boolean; initialSyncCompleted?: boolean }>(
    "/sync/session/restore",
    {
      method: "POST",
      timeoutMs: 60000,
      timeoutMessage:
        "La restauración de la sincronización está tardando demasiado. Intenta de nuevo.",
    },
  );
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
