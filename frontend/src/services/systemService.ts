import { apiRequest } from "@/services/api"

export async function forceSync(): Promise<{ synced: boolean }> {
  return apiRequest<{ synced: boolean }>("/sync/force", {
    method: "POST",
  })
}

export async function checkpointSQLite(): Promise<{ checkpointed: boolean }> {
  return apiRequest<{ checkpointed: boolean }>("/system/sqlite/checkpoint", {
    method: "POST",
  })
}
