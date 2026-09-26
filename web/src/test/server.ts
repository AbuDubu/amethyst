import { setupServer } from 'msw/node'

/**
 * Intercepts fetch in tests. Each test declares the responses it needs with
 * server.use(...); unhandled requests fail the test (see test-setup.ts).
 */
export const server = setupServer()
