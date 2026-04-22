import React from 'react';
import { describe, test, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';

vi.mock('../api.js', () => ({
  api: {
    rate: vi.fn(async (_id, payload) => payload)
  },
  setToken: vi.fn(),
  getToken: vi.fn()
}));

import RatingEditor from '../components/RatingEditor.jsx';
import { api } from '../api.js';

describe('RatingEditor', () => {
  beforeEach(() => { api.rate.mockClear(); });

  test('clicking a trust star calls api.rate with that value', async () => {
    render(<RatingEditor userId={42} initial={{ trust: 0, cool: 0, sexy: 0, is_fan: false }} />);
    fireEvent.click(screen.getByLabelText('confiavel 2'));
    await waitFor(() => expect(api.rate).toHaveBeenCalledWith(42, expect.objectContaining({ trust: 2 })));
  });

  test('toggles fan state', async () => {
    render(<RatingEditor userId={7} initial={{ trust: 0, cool: 0, sexy: 0, is_fan: false }} />);
    fireEvent.click(screen.getByText('♡ virar fa'));
    await waitFor(() => expect(api.rate).toHaveBeenCalledWith(7, expect.objectContaining({ is_fan: true })));
  });

  test('renders initial state with filled icons matching value', () => {
    render(<RatingEditor userId={1} initial={{ trust: 2, cool: 0, sexy: 0, is_fan: true }} />);
    expect(screen.getByText('♥ sou fa')).toBeInTheDocument();
  });
});
