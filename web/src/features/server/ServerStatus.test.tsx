import { screen } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '../../test/server'
import { renderWithProviders } from '../../test/render'
import { ServerStatus } from './ServerStatus'

describe('ServerStatus', () => {
  it('shows the running version once the server answers', async () => {
    server.use(
      http.get('*/api/server', () =>
        HttpResponse.json({ software: 'amethyst', version: 'v0.1.0' }),
      ),
    )

    renderWithProviders(<ServerStatus />)

    expect(screen.getByText('Checking server…')).toBeInTheDocument()
    expect(await screen.findByText('Running amethyst v0.1.0')).toBeInTheDocument()
  })

  it('explains when the server returns an error', async () => {
    server.use(
      http.get('*/api/server', () =>
        HttpResponse.json(
          { type: 'about:blank', title: 'Internal Server Error', status: 500, code: 'internal' },
          { status: 500, headers: { 'Content-Type': 'application/problem+json' } },
        ),
      ),
    )

    renderWithProviders(<ServerStatus />)

    expect(await screen.findByRole('alert')).toHaveTextContent('Couldn’t reach the server')
  })

  it('explains when the network fails', async () => {
    server.use(http.get('*/api/server', () => HttpResponse.error()))

    renderWithProviders(<ServerStatus />)

    expect(await screen.findByRole('alert')).toBeInTheDocument()
  })
})
