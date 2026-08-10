// zettelkasten-front/src/components/cards/EditorLinkSuggestions.test.tsx
import { render, screen, fireEvent, act } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { EditorLinkSuggestions } from './EditorLinkSuggestions';
import { Card, PartialCard } from '../../models/Card';
import { getRelatedCards } from '../../api/cards';

vi.mock('../../api/cards', () => ({
  getRelatedCards: vi.fn(),
}));

function makeCard(overrides: Partial<Card> = {}): Card {
  return {
    id: 1,
    card_id: 'edit-1',
    user_id: 1,
    title: 'Editing Card',
    body: 'Body being edited',
    link: '',
    is_deleted: false,
    created_at: new Date('2024-01-01'),
    updated_at: new Date('2024-01-01'),
    parent_id: 0,
    parent: null as any,
    files: [],
    children: [],
    references: [],
    tags: [],
    tasks: [],
    entities: [],
    ...overrides,
  };
}

function makePartial(id: number, cardId: string, title: string): PartialCard {
  return {
    id,
    card_id: cardId,
    user_id: 1,
    title,
    parent_id: 0,
    created_at: new Date('2024-01-01'),
    updated_at: new Date('2024-01-01'),
    tags: [],
  };
}

describe('EditorLinkSuggestions', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  function renderWithRouter(ui: React.ReactElement) {
    return render(<MemoryRouter>{ui}</MemoryRouter>);
  }

  it('renders nothing for new unsaved cards', () => {
    const card = makeCard({ id: 0 });
    renderWithRouter(
      <EditorLinkSuggestions card={card} newCard onInsertLink={() => {}} />,
    );
    expect(screen.queryByText('Link these')).not.toBeInTheDocument();
    expect(getRelatedCards).not.toHaveBeenCalled();
  });

  it('renders nothing when the card has no id', () => {
    const card = makeCard({ id: 0 });
    renderWithRouter(
      <EditorLinkSuggestions
        card={card}
        newCard={false}
        onInsertLink={() => {}}
      />,
    );
    expect(screen.queryByText('Link these')).not.toBeInTheDocument();
  });

  it('debounces and shows related-card suggestions for an existing card', async () => {
    vi.mocked(getRelatedCards).mockResolvedValue([
      {
        card: makePartial(10, 'REL-10', 'Related Card'),
        score: 5,
        reasons: ['tags'],
      },
    ]);
    const card = makeCard();

    renderWithRouter(
      <EditorLinkSuggestions
        card={card}
        newCard={false}
        onInsertLink={() => {}}
      />,
    );

    // Before the debounce fires, nothing shows.
    expect(screen.queryByText('Link these')).not.toBeInTheDocument();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(600);
    });

    expect(getRelatedCards).toHaveBeenCalledWith('1');
    expect(screen.getByText('Link these')).toBeInTheDocument();
    expect(screen.getByText('- Related Card')).toBeInTheDocument();
  });

  it('clicking a suggestion calls onInsertLink with card_id and title', async () => {
    vi.mocked(getRelatedCards).mockResolvedValue([
      {
        card: makePartial(10, 'REL-10', 'Related Card'),
        score: 5,
        reasons: ['tags'],
      },
    ]);
    const onInsertLink = vi.fn();
    const card = makeCard();

    renderWithRouter(
      <EditorLinkSuggestions
        card={card}
        newCard={false}
        onInsertLink={onInsertLink}
      />,
    );
    await act(async () => {
      await vi.advanceTimersByTimeAsync(600);
    });

    fireEvent.click(screen.getByText('- Related Card'));
    expect(onInsertLink).toHaveBeenCalledWith('REL-10', 'Related Card');
  });

  it('excludes cards already linked in the body', async () => {
    vi.mocked(getRelatedCards).mockResolvedValue([
      {
        card: makePartial(10, 'REL-10', 'Related Card'),
        score: 5,
        reasons: ['tags'],
      },
      {
        card: makePartial(11, 'REL-11', 'Already Linked'),
        score: 4,
        reasons: ['tags'],
      },
    ]);
    const card = makeCard({ body: 'Body mentioning [[REL-11]] already' });

    renderWithRouter(
      <EditorLinkSuggestions
        card={card}
        newCard={false}
        onInsertLink={() => {}}
      />,
    );
    await act(async () => {
      await vi.advanceTimersByTimeAsync(600);
    });

    expect(screen.getByText('- Related Card')).toBeInTheDocument();
    expect(screen.queryByText('- Already Linked')).not.toBeInTheDocument();
  });
});
