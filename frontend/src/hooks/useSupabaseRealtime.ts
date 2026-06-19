import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";

import { isTauriRuntime } from "@/lib/runtime";
import { authenticateSupabaseRealtime } from "@/services/supabaseClient";
import { useAuthStore } from "@/store/useAuthStore";

export function useSupabaseRealtime() {
  const accessToken = useAuthStore((state) => state.accessToken);
  const userID = useAuthStore((state) => state.user?.id);
  const queryClient = useQueryClient();

  useEffect(() => {
    if (isTauriRuntime()) {
      return;
    }

    if (!accessToken || !userID) {
      console.warn("Supabase Realtime disabled: missing authenticated user session");
      return;
    }

    const supabase = authenticateSupabaseRealtime(accessToken);
    if (!supabase) {
      console.warn("Supabase Realtime disabled: missing VITE_SUPABASE_URL or VITE_SUPABASE_ANON_KEY");
      return;
    }

    const channel = supabase
      .channel(`taskify-realtime-${userID}`)
      .on(
        "postgres_changes",
        { event: "*", schema: "public", table: "boards" },
        (payload) => {
          console.info("Supabase Realtime event", {
            table: payload.table,
            event: payload.eventType,
            id: "id" in payload.new ? payload.new.id : "id" in payload.old ? payload.old.id : undefined,
          });
          void queryClient.invalidateQueries();
        },
      )
      .on(
        "postgres_changes",
        { event: "*", schema: "public", table: "tasks" },
        (payload) => {
          console.info("Supabase Realtime event", {
            table: payload.table,
            event: payload.eventType,
            id: "id" in payload.new ? payload.new.id : "id" in payload.old ? payload.old.id : undefined,
          });
          void queryClient.invalidateQueries();
        },
      )
      .subscribe((status, error) => {
        if (status === "SUBSCRIBED") {
          console.info("Supabase Realtime subscribed", { channel: channel.topic });
          return;
        }

        if (status === "CHANNEL_ERROR" || status === "TIMED_OUT" || status === "CLOSED") {
          console.warn("Supabase Realtime status", { status, error });
        }
      });

    return () => {
      void supabase.removeChannel(channel);
    };
  }, [accessToken, queryClient, userID]);
}
