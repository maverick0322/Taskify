import type { QueryClient } from "@tanstack/react-query";

const realtimeQueryKeys = [
  ["boards"],
  ["tasks"],
  ["financial"],
  ["notifications"],
  ["users", "me"],
] as const;

export async function invalidateRealtimeQueries(queryClient: QueryClient) {
  await Promise.all(
    realtimeQueryKeys.map((queryKey) =>
      queryClient.invalidateQueries({
        queryKey: [...queryKey],
        refetchType: "active",
      }),
    ),
  );

  await queryClient.refetchQueries({ type: "active" });
}
