import { useQuery } from '@tanstack/react-query'
import { api, unwrap } from '../../api/client'

/** Query key for server information; used for caching and invalidation. */
export const serverInfoKey = ['server'] as const

/** Information about the server this UI is talking to. */
export function useServerInfo() {
  return useQuery({
    queryKey: serverInfoKey,
    queryFn: async ({ signal }) => unwrap(await api.GET('/api/server', { signal })),
    // Server info changes only on deploy; don't refetch on every focus.
    staleTime: 5 * 60 * 1000,
  })
}
