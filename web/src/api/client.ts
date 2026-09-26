import createClient from 'openapi-fetch'
import type { components, paths } from './schema'

export type Problem = components['schemas']['Problem']

/**
 * Typed client for the browser API. Paths, parameters, and response bodies are
 * checked against the generated contract types (src/api/schema.d.ts).
 */
export const api = createClient<paths>({
  // Absolute so the same client works in the browser and in tests (Node's
  // fetch rejects relative URLs).
  baseUrl: globalThis.location?.origin ?? 'http://localhost',
  // Look fetch up per request rather than capturing it at import time, so
  // anything that wraps fetch later (such as MSW in tests) is honored.
  fetch: (request) => globalThis.fetch(request),
})

/** An API error response, carrying its RFC 9457 Problem Details when present. */
export class ApiError extends Error {
  readonly status: number
  readonly problem?: Problem

  constructor(status: number, problem?: Problem) {
    super(problem?.detail ?? problem?.title ?? `Request failed with status ${status}`)
    this.name = 'ApiError'
    this.status = status
    this.problem = problem
  }
}

/**
 * Unwraps an openapi-fetch result: returns the data, or throws ApiError so
 * TanStack Query sees a failed query.
 */
export function unwrap<T>(result: { data?: T; error?: unknown; response: Response }): T {
  if (result.error !== undefined || result.data === undefined) {
    throw new ApiError(result.response.status, asProblem(result.error))
  }
  return result.data
}

function asProblem(error: unknown): Problem | undefined {
  if (typeof error === 'object' && error !== null && 'code' in error && 'status' in error) {
    return error as Problem
  }
  return undefined
}
