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

type RemoteRealtimeEvent = {
  type?: string;
};

function resolveRemoteRealtimeUrl(accessToken: string) {
  const apiBaseUrl = new URL(resolveApiBaseUrl());
  apiBaseUrl.protocol = apiBaseUrl.protocol === "https:" ? "wss:" : "ws:";
  apiBaseUrl.pathname = "/realtime/ws";
  apiBaseUrl.search = "";
  apiBaseUrl.searchParams.set("token", accessToken);
  return apiBaseUrl.toString();
}

export function useRemoteRealtime() {
  const accessToken = useAuthStore((state) => state.accessToken);
  const queryClient = useQueryClient();

  useEffect(() => {
    if (isTauriRuntime() || !accessToken || typeof WebSocket === "undefined") {
      return;
    }

    let disposed = false;
    let socket: WebSocket | null = null;
    let reconnectTimer: ReturnType<typeof setTimeout> | undefined;
    let reconnectAttempts = 0;
    let connectInFlight = false;

    const scheduleReconnect = () => {
      if (disposed) {
        return;
      }
      const delay = Math.min(1000 * 2 ** reconnectAttempts, 10000);
      reconnectAttempts += 1;
      reconnectTimer = setTimeout(connect, delay);
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

        socket = new WebSocket(resolveRemoteRealtimeUrl(currentAccessToken));
        socket.onopen = () => {
          reconnectAttempts = 0;
        };
        socket.onmessage = (event) => {
          try {
            const payload = JSON.parse(event.data) as RemoteRealtimeEvent;
            if (payload.type === "sync_update") {
              void invalidateRealtimeQueries(queryClient);
            }
          } catch (error) {
            console.warn("remote realtime event parse error", error);
          }
        };
        socket.onerror = () => {
          socket?.close();
        };
        socket.onclose = () => {
          scheduleReconnect();
        };
      } catch (error) {
        const normalizedError = normalizeApiError(error);
        if (normalizedError.status === 401) {
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
      socket?.close();
    };
  }, [accessToken, queryClient]);
}
