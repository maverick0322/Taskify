import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";

import { isTauriRuntime } from "@/lib/runtime";
import {
  ensureValidAccessToken,
  normalizeApiError,
  resolveApiBaseUrl,
} from "@/services/api";
import { invalidateRealtimeQueries } from "@/services/realtime";
import { useAuthStore } from "@/store/useAuthStore";

export function useSyncEvents() {
  const accessToken = useAuthStore((state) => state.accessToken);
  const queryClient = useQueryClient();

  useEffect(() => {
    if (!isTauriRuntime() || !accessToken || typeof EventSource === "undefined") {
      return;
    }

    let disposed = false;
    let eventSource: EventSource | null = null;
    let reconnectTimer: ReturnType<typeof setTimeout> | undefined;
    let connectInFlight = false;
    let reconnectScheduled = false;

    const scheduleReconnect = () => {
      if (disposed || reconnectScheduled) {
        return;
      }
      reconnectScheduled = true;
      reconnectTimer = setTimeout(() => {
        reconnectScheduled = false;
        void connect();
      }, 1500);
    };

    const closeEventSource = () => {
      eventSource?.removeEventListener("sync_updated", handleSyncUpdated);
      eventSource?.close();
      eventSource = null;
    };

    const handleSyncUpdated = () => {
      void invalidateRealtimeQueries(queryClient);
    };

    const connect = async () => {
      if (disposed || connectInFlight) {
        return;
      }
      connectInFlight = true;

      try {
        const currentAccessToken = await ensureValidAccessToken();
        if (disposed) {
          return;
        }

        const eventUrl = new URL(`${resolveApiBaseUrl()}/sync/events`);
        eventUrl.searchParams.set("token", currentAccessToken);

        eventSource = new EventSource(eventUrl.toString());
        eventSource.addEventListener("sync_updated", handleSyncUpdated);
        eventSource.onerror = (error) => {
          console.warn("sync events stream error", error);
          closeEventSource();
          scheduleReconnect();
        };
      } catch (error) {
        const normalizedError = normalizeApiError(error);
        if (normalizedError.status === 401) {
          closeEventSource();
          await useAuthStore.getState().logout();
          return;
        }
        scheduleReconnect();
      } finally {
        connectInFlight = false;
      }
    };

    void connect();

    return () => {
      disposed = true;
      if (reconnectTimer) {
        clearTimeout(reconnectTimer);
      }
      closeEventSource();
    };
  }, [accessToken, queryClient]);
}
