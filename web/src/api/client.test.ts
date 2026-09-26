import { describe, expect, it } from 'vitest'
import { ApiError, unwrap } from './client'

describe('unwrap', () => {
  it('returns data from a successful response', () => {
    expect(unwrap({ data: { ok: true }, response: new Response(null, { status: 200 }) })).toEqual({ ok: true })
  })

  it('throws ApiError carrying the Problem Details', () => {
    const problem = { type: 'about:blank', title: 'Not Found', status: 404, code: 'not_found', detail: 'No such API endpoint.' }

    const thrown = (() => {
      try {
        unwrap({ error: problem, response: new Response(null, { status: 404 }) })
      } catch (e) {
        return e
      }
    })()

    expect(thrown).toBeInstanceOf(ApiError)
    expect(thrown).toMatchObject({ status: 404, problem, message: 'No such API endpoint.' })
  })

  it('throws ApiError without a problem for non-Problem error bodies', () => {
    expect(() => unwrap({ error: 'oops', response: new Response(null, { status: 502 }) })).toThrow(
      'Request failed with status 502',
    )
  })
})
