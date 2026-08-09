// zettelkasten-front/src/components/cards/RelatedCards.test.tsx
import { render, screen, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, it, expect, vi } from 'vitest';
import { RelatedCards } from './RelatedCards';
import { RelatedCard } from '../../models/Card';

function makeRelatedCard(
  id: number,
  cardId: string,
  title: string,
  score: number,
  reasons: string[],
): RelatedCard {
  return {
    card: {
      id,
      card_id: cardId,
      user_id: 1,
      title,
      parent_id: 0,
      created_at: new Date(),
      updated_at: new Date(),
      tags: [],
    },
    score,
    reasons,
  };
}

describe('RelatedCards', () => {
  function renderWithRouter(ui: React.ReactElement) {
    return render(<MemoryRouter>{ui}</MemoryRouter>);
  }

  it('renders nothing when there are no related cards', () => {
    const { container } = renderWithRouter(
      <RelatedCards relatedCards={[]} onCardClick={() => {}} />,
    );
    expect(container).toBeEmptyDOMElement();
  });

  it('renders card titles and scores', () => {
    const cards = [
      makeRelatedCard(1, 'CARD-1', 'First Card', 4.2, []),
      makeRelatedCard(2, 'CARD-2', 'Second Card', 2.5, []),
    ];
    renderWithRouter(
      <RelatedCards relatedCards={cards} onCardClick={() => {}} />,
    );
    expect(screen.getByText('- First Card')).toBeInTheDocument();
    expect(screen.getByText('- Second Card')).toBeInTheDocument();
    expect(screen.getByText('4.2')).toBeInTheDocument();
    expect(screen.getByText('2.5')).toBeInTheDocument();
  });

  it('renders reason chips explaining why each card is related', () => {
    const cards = [
      makeRelatedCard(1, 'CARD-1', 'First Card', 6, [
        '2 shared entities: Python, LLM',
        'semantically similar',
      ]),
      makeRelatedCard(2, 'CARD-2', 'Second Card', 1, [
        '1 shared tag: research',
      ]),
    ];
    renderWithRouter(
      <RelatedCards relatedCards={cards} onCardClick={() => {}} />,
    );

    expect(
      screen.getByText('2 shared entities: Python, LLM'),
    ).toBeInTheDocument();
    expect(screen.getByText('semantically similar')).toBeInTheDocument();
    expect(screen.getByText('1 shared tag: research')).toBeInTheDocument();
  });

  it('does not render a reason row when reasons are absent', () => {
    const cards = [makeRelatedCard(1, 'CARD-1', 'First Card', 6, [])];
    renderWithRouter(
      <RelatedCards relatedCards={cards} onCardClick={() => {}} />,
    );
    expect(
      screen.queryByText('shared entities', { exact: false }),
    ).not.toBeInTheDocument();
  });

  it('calls onCardClick when a card is clicked', () => {
    const onCardClick = vi.fn();
    const cards = [makeRelatedCard(1, 'CARD-1', 'First Card', 6, [])];
    renderWithRouter(
      <RelatedCards relatedCards={cards} onCardClick={onCardClick} />,
    );

    fireEvent.click(screen.getByText('- First Card'));
    expect(onCardClick).toHaveBeenCalledWith(1);
  });

  it('calls onAddReference when +Ref is clicked', () => {
    const onAddReference = vi.fn();
    const cards = [makeRelatedCard(1, 'CARD-1', 'First Card', 6, [])];
    renderWithRouter(
      <RelatedCards
        relatedCards={cards}
        onCardClick={() => {}}
        onAddReference={onAddReference}
      />,
    );

    fireEvent.click(screen.getByTitle('Add as reference'));
    expect(onAddReference).toHaveBeenCalledWith(cards[0]);
  });
});
