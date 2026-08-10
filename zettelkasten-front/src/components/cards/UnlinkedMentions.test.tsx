// zettelkasten-front/src/components/cards/UnlinkedMentions.test.tsx
import { render, screen, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, it, expect, vi } from 'vitest';
import { UnlinkedMentions } from './UnlinkedMentions';
import { UnlinkedMention } from '../../models/Card';

function makeMention(
  id: number,
  cardId: string,
  title: string,
  count: number,
  snippet: string,
): UnlinkedMention {
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
    mention_count: count,
    context_snippet: snippet,
  };
}

describe('UnlinkedMentions', () => {
  function renderWithRouter(ui: React.ReactElement) {
    return render(<MemoryRouter>{ui}</MemoryRouter>);
  }

  it('shows an empty-state message when there are no mentions', () => {
    renderWithRouter(
      <UnlinkedMentions
        mentions={[]}
        onCardClick={() => {}}
        onAddLink={() => {}}
      />,
    );
    expect(screen.getByText(/no unlinked mentions/i)).toBeInTheDocument();
  });

  it('renders mention cards with count and snippet', () => {
    const mentions = [
      makeMention(
        1,
        'NOTE-1',
        'Old Notes',
        2,
        '...see NOTE-1 for details and NOTE-1 again...',
      ),
      makeMention(2, 'NOTE-2', 'Other Notes', 1, '...mentions NOTE-2 here...'),
    ];
    renderWithRouter(
      <UnlinkedMentions
        mentions={mentions}
        onCardClick={() => {}}
        onAddLink={() => {}}
      />,
    );

    expect(screen.getByText('- Old Notes')).toBeInTheDocument();
    expect(screen.getByText('- Other Notes')).toBeInTheDocument();
    expect(screen.getByText('2x')).toBeInTheDocument();
    expect(screen.getByText('1x')).toBeInTheDocument();
    expect(
      screen.getByText('...see NOTE-1 for details and NOTE-1 again...'),
    ).toBeInTheDocument();
  });

  it('calls onCardClick when a card row is clicked', () => {
    const onCardClick = vi.fn();
    const mentions = [makeMention(1, 'NOTE-1', 'Old Notes', 1, 'snippet')];
    renderWithRouter(
      <UnlinkedMentions
        mentions={mentions}
        onCardClick={onCardClick}
        onAddLink={() => {}}
      />,
    );

    fireEvent.click(screen.getByText('- Old Notes'));
    expect(onCardClick).toHaveBeenCalledWith(1);
  });

  it('calls onAddLink when +Link is clicked', () => {
    const onAddLink = vi.fn();
    const mentions = [makeMention(1, 'NOTE-1', 'Old Notes', 1, 'snippet')];
    renderWithRouter(
      <UnlinkedMentions
        mentions={mentions}
        onCardClick={() => {}}
        onAddLink={onAddLink}
      />,
    );

    fireEvent.click(screen.getByTitle('Insert a link to this card'));
    expect(onAddLink).toHaveBeenCalledWith(mentions[0]);
  });
});
