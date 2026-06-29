import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";

import { isTauriRuntime } from "@/lib/runtime";
import { resolveApiBaseUrl } from "@/services/api";
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

    const scheduleReconnect = () => {
      if (disposed) {
        return;
      }
      const delay = Math.min(1000 * 2 ** reconnectAttempts, 10000);
      reconnectAttempts += 1;
      reconnectTimer = setTimeout(connect, delay);
    };

    const connect = () => {
      if (disposed) {
        return;
      }

      socket = new WebSocket(resolveRemoteRealtimeUrl(accessToken));
      socket.onopen = () => {
        reconnectAttempts = 0;
      };
      socket.onmessage = (event) => {
        try {
          const payload = JSON.parse(event.data) as RemoteRealtimeEvent;
          if (payload.type === "sync_update") {
            void queryClient.invalidateQueries();
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
    };

    connect();

    return () => {
      disposed = true;
      if (reconnectTimer) {
        clearTimeout(reconnectTimer);
      }
      socket?.close();
    };
  }, [accessToken, queryClient]);
}
