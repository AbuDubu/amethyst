import { QueryClient } from '@tanstack/react-query'
import { ApiError } from './client'

/** Creates the app's server-state cache. Tests create their own per test. */
export function createQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        // Retry network and server failures twice; a 4xx won't succeed on retry.
        retry: (failureCount, error) =>
          failureCount < 2 && !(error instanceof ApiError && error.status < 500),
      },
    },
  })
}
