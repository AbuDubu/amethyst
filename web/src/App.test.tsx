import { screen } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import App from './App'
import { server } from './test/server'
import { renderWithProviders } from './test/render'

describe('App', () => {
  it('shows the product name as the page heading', async () => {
    server.use(
      http.get('*/api/server', () => HttpResponse.json({ software: 'amethyst', version: 'dev' })),
    )

    renderWithProviders(<App />)

    expect(
      screen.getByRole('heading', { level: 1, name: 'Amethyst' }),
    ).toBeInTheDocument()
    // Let the server status query settle so it doesn't outlive the test.
    expect(await screen.findByText('Running amethyst dev')).toBeInTheDocument()
  })
})
