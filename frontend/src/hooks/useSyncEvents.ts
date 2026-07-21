import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";

import { isTauriRuntime } from "@/lib/runtime";
import {
  ensureValidAccessToken,
  normalizeApiError,
  resolveApiBaseUrl,
  restoreOrRefreshSession,
} from "@/services/api";
import { invalidateRealtimeQueries } from "@/services/realtime";
import { useAuthStore } from "@/store/useAuthStore";
import { useDesktopSyncStore } from "@/store/useDesktopSyncStore";

export function useSyncEvents() {
  const accessToken = useAuthStore((state) => state.accessToken);
  const logout = useAuthStore((state) => state.logout);
  const setDesktopSyncStatus = useDesktopSyncStore((state) => state.setStatus);
  const queryClient = useQueryClient();

  useEffect(() => {
    if (
      !isTauriRuntime() ||
      !accessToken ||
      typeof EventSource === "undefined"
    ) {
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
      eventSource?.removeEventListener(
        "sync_status_connected",
        handleSyncStatusConnected,
      );
      eventSource?.removeEventListener(
        "sync_status_pending",
        handleSyncStatusPending,
      );
      eventSource?.removeEventListener(
        "sync_status_error",
        handleSyncStatusError,
      );
      eventSource?.close();
      eventSource = null;
    };

    const handleSyncUpdated = () => {
      setDesktopSyncStatus("connected", null);
      void invalidateRealtimeQueries(queryClient);
    };

    const handleSyncStatusConnected = () => {
      setDesktopSyncStatus("connected", null);
    };

    const handleSyncStatusPending = () => {
      setDesktopSyncStatus(
        "pending",
        "La sincronización remota sigue pendiente. Seguimos trabajando con tus datos locales.",
      );
    };

    const handleSyncStatusError = () => {
      setDesktopSyncStatus(
        "error",
        "No pudimos sincronizar con la nube por ahora. Tus datos locales siguen disponibles.",
      );
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
        eventSource.addEventListener(
          "sync_status_connected",
          handleSyncStatusConnected,
        );
        eventSource.addEventListener(
          "sync_status_pending",
          handleSyncStatusPending,
        );
        eventSource.addEventListener(
          "sync_status_error",
          handleSyncStatusError,
        );
        eventSource.onerror = (error) => {
          console.warn("sync events stream error", error);
          closeEventSource();
          setDesktopSyncStatus(
            "pending",
            "Reconectando la sincronización remota…",
          );
          scheduleReconnect();
        };
      } catch (error) {
        const normalizedError = normalizeApiError(error);
        if (normalizedError.status === 401) {
          const restoredAccessToken = await restoreOrRefreshSession().catch(
            () => null,
          );
          if (restoredAccessToken) {
            scheduleReconnect();
            return;
          }

          closeEventSource();
          setDesktopSyncStatus(
            "offline",
            "La sesión local expiró. Inicia sesión nuevamente para reanudar la sincronización.",
          );
          await logout();
          return;
        }
        setDesktopSyncStatus(
          "pending",
          "La sincronización remota sigue pendiente. Seguimos trabajando con tus datos locales.",
        );
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
  }, [accessToken, logout, queryClient, setDesktopSyncStatus]);
}
