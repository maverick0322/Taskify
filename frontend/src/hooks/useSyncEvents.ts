import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";

import { API_BASE_URL } from "@/services/api";
import { useAuthStore } from "@/store/useAuthStore";

export function useSyncEvents() {
  const accessToken = useAuthStore((state) => state.accessToken);
  const queryClient = useQueryClient();

  useEffect(() => {
    if (!accessToken || typeof EventSource === "undefined") {
      return;
    }

    const eventUrl = new URL(`${API_BASE_URL}/sync/events`);
    eventUrl.searchParams.set("token", accessToken);

    const eventSource = new EventSource(eventUrl.toString());
    const handleSyncUpdated = () => {
      void queryClient.invalidateQueries();
    };

    eventSource.addEventListener("sync_updated", handleSyncUpdated);
    eventSource.onerror = (error) => {
      console.warn("sync events stream error", error);
    };

    return () => {
      eventSource.removeEventListener("sync_updated", handleSyncUpdated);
      eventSource.close();
    };
  }, [accessToken, queryClient]);
}
