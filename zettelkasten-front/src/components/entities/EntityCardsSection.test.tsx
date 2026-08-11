// zettelkasten-front/src/components/entities/EntityCardsSection.test.tsx
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, it, expect } from 'vitest';
import { EntityCardsSection } from './EntityCardsSection';
import { EntityCard } from '../../models/Card';

function makeEntityCard(
  id: number,
  cardId: string,
  title: string,
  count: number,
): EntityCard {
  return {
    card: {
      id,
      card_id: cardId,
      user_id: 1,
      title,
      parent_id: 0,
      created_at: new Date('2024-01-01'),
      updated_at: new Date('2024-01-01'),
      tags: [],
    },
    entity_count: count,
  };
}

describe('EntityCardsSection', () => {
  function renderWithRouter(ui: React.ReactElement) {
    return render(<MemoryRouter>{ui}</MemoryRouter>);
  }

  it('shows loading state', () => {
    renderWithRouter(<EntityCardsSection cards={[]} isLoading error={null} />);
    expect(screen.getByText('Loading cards...')).toBeInTheDocument();
  });

  it('shows an empty state when there are no cards', () => {
    renderWithRouter(
      <EntityCardsSection cards={[]} isLoading={false} error={null} />,
    );
    expect(
      screen.getByText('No cards found for this entity.'),
    ).toBeInTheDocument();
  });

  it('renders associated cards with a count header and per-card entity counts', () => {
    const cards = [
      makeEntityCard(1, 'CARD-1', 'First Card', 3),
      makeEntityCard(2, 'CARD-2', 'Second Card', 1),
    ];
    renderWithRouter(
      <EntityCardsSection cards={cards} isLoading={false} error={null} />,
    );

    expect(screen.getByText('Associated Cards (2)')).toBeInTheDocument();
    expect(screen.getByText('- First Card')).toBeInTheDocument();
    expect(screen.getByText('- Second Card')).toBeInTheDocument();
    expect(screen.getByText('3 ent')).toBeInTheDocument();
    expect(screen.getByText('1 ent')).toBeInTheDocument();
  });

  it('shows the error message when loading fails', () => {
    renderWithRouter(
      <EntityCardsSection cards={[]} isLoading={false} error="boom" />,
    );
    expect(screen.getByText('boom')).toBeInTheDocument();
  });
});
