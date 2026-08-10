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

  it('renders a custom title when provided', () => {
    const cards = [makeRelatedCard(1, 'CARD-1', 'First Card', 6, [])];
    renderWithRouter(
      <RelatedCards
        title="Suggestions"
        relatedCards={cards}
        onCardClick={() => {}}
      />,
    );
    expect(screen.getByText('Suggestions')).toBeInTheDocument();
    expect(screen.queryByText('Related Cards')).not.toBeInTheDocument();
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

  it('stays flat when all cards share a single reason category', () => {
    const cards = [
      makeRelatedCard(1, 'CARD-1', 'First Card', 6, [
        '2 shared entities: Python, LLM',
      ]),
      makeRelatedCard(2, 'CARD-2', 'Second Card', 3, [
        '1 shared entity: Python',
      ]),
    ];
    renderWithRouter(
      <RelatedCards relatedCards={cards} onCardClick={() => {}} />,
    );

    expect(screen.queryByText('Shared entities')).not.toBeInTheDocument();
    expect(screen.queryByText('Shared tags')).not.toBeInTheDocument();
    expect(screen.queryByText('Semantically similar')).not.toBeInTheDocument();
    expect(screen.getByText('- First Card')).toBeInTheDocument();
    expect(screen.getByText('- Second Card')).toBeInTheDocument();
  });

  it('groups cards by reason category with headers when multiple categories exist', () => {
    const cards = [
      makeRelatedCard(1, 'CARD-1', 'Entity Card', 6, [
        '2 shared entities: Python, LLM',
      ]),
      makeRelatedCard(2, 'CARD-2', 'Tag Card', 2, ['2 shared tags: notes']),
      makeRelatedCard(3, 'CARD-3', 'Similar Card', 4, ['semantically similar']),
    ];
    renderWithRouter(
      <RelatedCards relatedCards={cards} onCardClick={() => {}} />,
    );

    expect(screen.getByText('Shared entities (1)')).toBeInTheDocument();
    expect(screen.getByText('Shared tags (1)')).toBeInTheDocument();
    expect(screen.getByText('Semantically similar (1)')).toBeInTheDocument();
    expect(screen.getByText('- Entity Card')).toBeInTheDocument();
    expect(screen.getByText('- Tag Card')).toBeInTheDocument();
    expect(screen.getByText('- Similar Card')).toBeInTheDocument();
  });

  it('assigns a card with multiple reasons to its first (primary) category only', () => {
    const cards = [
      makeRelatedCard(1, 'CARD-1', 'Entity Card', 8, [
        '1 shared entity: Python',
        '2 shared tags: notes, research',
        'semantically similar',
      ]),
      makeRelatedCard(2, 'CARD-2', 'Tag Card', 2, ['2 shared tags: notes']),
    ];
    renderWithRouter(
      <RelatedCards relatedCards={cards} onCardClick={() => {}} />,
    );

    expect(screen.getByText('Shared entities (1)')).toBeInTheDocument();
    expect(screen.getByText('Shared tags (1)')).toBeInTheDocument();
    // No card has similarity as its PRIMARY reason, so no group header renders.
    expect(screen.queryByText('Semantically similar')).not.toBeInTheDocument();

    // Entity Card should appear exactly once, under Shared entities.
    const entityCardTitles = screen.getAllByText('- Entity Card');
    expect(entityCardTitles).toHaveLength(1);
  });
});
