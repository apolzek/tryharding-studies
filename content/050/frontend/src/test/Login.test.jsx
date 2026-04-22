import React from 'react';
import { describe, test, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';

const loginMock = vi.fn();
vi.mock('../auth.jsx', () => ({
  useAuth: () => ({ login: loginMock })
}));

import Login from '../pages/Login.jsx';

describe('Login page', () => {
  beforeEach(() => { loginMock.mockReset(); });

  test('submits username and password', async () => {
    loginMock.mockResolvedValue({});
    render(<MemoryRouter><Login /></MemoryRouter>);
    fireEvent.change(screen.getByLabelText('usuario'), { target: { value: 'alice' } });
    fireEvent.change(screen.getByLabelText('senha'), { target: { value: 'secret1' } });
    fireEvent.click(screen.getByRole('button', { name: 'entrar' }));
    await waitFor(() => expect(loginMock).toHaveBeenCalledWith('alice', 'secret1'));
  });

  test('shows error when login fails', async () => {
    loginMock.mockRejectedValue(new Error('invalid credentials'));
    render(<MemoryRouter><Login /></MemoryRouter>);
    fireEvent.change(screen.getByLabelText('usuario'), { target: { value: 'x' } });
    fireEvent.change(screen.getByLabelText('senha'), { target: { value: 'y' } });
    fireEvent.click(screen.getByRole('button', { name: 'entrar' }));
    expect(await screen.findByRole('alert')).toHaveTextContent('invalid credentials');
  });
});
