import React from 'react';
import { describe, test, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import ProfileCard from '../components/ProfileCard.jsx';

function wrap(ui) {
  return render(<MemoryRouter>{ui}</MemoryRouter>);
}

describe('ProfileCard', () => {
  test('renders user display name and meta', () => {
    wrap(<ProfileCard user={{ id: 1, display_name: 'Joao', status: 'solteiro(a)', age: 22, city: 'Sampa', country: 'Brasil' }} />);
    expect(screen.getByText('Joao')).toBeInTheDocument();
    expect(screen.getByText('solteiro(a)')).toBeInTheDocument();
    expect(screen.getByText('22 anos')).toBeInTheDocument();
    expect(screen.getByText('Sampa, Brasil')).toBeInTheDocument();
  });

  test('renders placeholder when no photo', () => {
    wrap(<ProfileCard user={{ id: 1, display_name: 'Maria' }} />);
    expect(screen.getByText('sem foto')).toBeInTheDocument();
  });

  test('renders rating summary with filled icons', () => {
    wrap(
      <ProfileCard
        user={{ id: 1, display_name: 'Ana' }}
        summary={{ trust: 3, cool: 1, sexy: 0, fans: 5 }}
      />
    );
    expect(screen.getByText('5 fas')).toBeInTheDocument();
    expect(screen.getByText('confiavel')).toBeInTheDocument();
    expect(screen.getByText('legal')).toBeInTheDocument();
    expect(screen.getByText('sexy')).toBeInTheDocument();
  });

  test('returns nothing when no user', () => {
    const { container } = wrap(<ProfileCard user={null} />);
    expect(container.firstChild).toBeNull();
  });
});
