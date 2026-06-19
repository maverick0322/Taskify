import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";

import { isTauriRuntime } from "@/lib/runtime";
import { getSupabaseClient } from "@/services/supabaseClient";

export function useSupabaseRealtime() {
  const queryClient = useQueryClient();

  useEffect(() => {
    if (isTauriRuntime()) {
      return;
    }

    const supabase = getSupabaseClient();
    if (!supabase) {
      console.warn("Supabase Realtime disabled: missing VITE_SUPABASE_URL or VITE_SUPABASE_ANON_KEY");
      return;
    }

    const channel = supabase
      .channel("taskify-realtime")
      .on(
        "postgres_changes",
        { event: "*", schema: "public", table: "boards" },
        () => {
          void queryClient.invalidateQueries();
        },
      )
      .on(
        "postgres_changes",
        { event: "*", schema: "public", table: "tasks" },
        () => {
          void queryClient.invalidateQueries();
        },
      )
      .subscribe();

    return () => {
      void supabase.removeChannel(channel);
    };
  }, [queryClient]);
}
