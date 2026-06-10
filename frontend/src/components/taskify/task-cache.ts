import type { QueryClient } from "@tanstack/react-query"

export function invalidateTaskCaches(
  queryClient: QueryClient,
  boardId?: string | null,
) {
  const invalidations = [
    queryClient.invalidateQueries({ queryKey: ["tasks", "global"] }),
  ]

  if (boardId) {
    invalidations.push(
      queryClient.invalidateQueries({ queryKey: ["tasks", boardId] }),
    )
  }

  return Promise.all(invalidations)
}
