import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";

import { isTauriRuntime } from "@/lib/runtime";
import { authenticateSupabaseRealtime } from "@/services/supabaseClient";
import { useAuthStore } from "@/store/useAuthStore";

type RealtimeRecord = Record<string, unknown>;

function realtimeRecordID(record: unknown): unknown {
  if (!record || typeof record !== "object") {
    return undefined;
  }
  return (record as RealtimeRecord).id;
}

function realtimeDeletedAt(record: unknown): unknown {
  if (!record || typeof record !== "object") {
    return undefined;
  }
  return (record as RealtimeRecord).deleted_at;
}

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
          const id = realtimeRecordID(payload.new) ?? realtimeRecordID(payload.old);
          const deletedAt = realtimeDeletedAt(payload.new) ?? realtimeDeletedAt(payload.old);
          console.info("Supabase Realtime event", {
            table: payload.table,
            event: payload.eventType,
            id,
            deletedAt,
            softDeleted: Boolean(deletedAt),
          });
          if (!id) {
            console.warn("Supabase Realtime payload without row id; check RLS SELECT policy", {
              table: payload.table,
              event: payload.eventType,
              payload,
            });
          }
          void queryClient.invalidateQueries();
        },
      )
      .on(
        "postgres_changes",
        { event: "*", schema: "public", table: "tasks" },
        (payload) => {
          const id = realtimeRecordID(payload.new) ?? realtimeRecordID(payload.old);
          const deletedAt = realtimeDeletedAt(payload.new) ?? realtimeDeletedAt(payload.old);
          console.info("Supabase Realtime event", {
            table: payload.table,
            event: payload.eventType,
            id,
            deletedAt,
            softDeleted: Boolean(deletedAt),
          });
          if (!id) {
            console.warn("Supabase Realtime payload without row id; check RLS SELECT policy", {
              table: payload.table,
              event: payload.eventType,
              payload,
            });
          }
          void queryClient.invalidateQueries();
        },
      )
      .on(
        "postgres_changes",
        { event: "*", schema: "public", table: "columns" },
        (payload) => {
          const id = realtimeRecordID(payload.new) ?? realtimeRecordID(payload.old);
          const deletedAt = realtimeDeletedAt(payload.new) ?? realtimeDeletedAt(payload.old);
          console.info("Supabase Realtime event", {
            table: payload.table,
            event: payload.eventType,
            id,
            deletedAt,
            softDeleted: Boolean(deletedAt),
          });
          if (!id) {
            console.warn("Supabase Realtime payload without row id; check RLS SELECT policy", {
              table: payload.table,
              event: payload.eventType,
              payload,
            });
          }
          void queryClient.invalidateQueries();
        },
      )
      .subscribe((status, error) => {
        if (status === "SUBSCRIBED") {
          console.info("Supabase Realtime SUBSCRIBED", { channel: channel.topic, userID });
          return;
        }

        if (status === "CHANNEL_ERROR" || status === "TIMED_OUT" || status === "CLOSED") {
          console.warn(`Supabase Realtime ${status}`, { channel: channel.topic, error });
        }
      });

    return () => {
      void supabase.removeChannel(channel);
    };
  }, [accessToken, queryClient, userID]);
}
