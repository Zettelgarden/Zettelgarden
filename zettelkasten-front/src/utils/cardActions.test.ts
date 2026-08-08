/**
 * Tests for card action helpers (src/utils/cardActions.ts)
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import {
  addTagToBody,
  removeTagFromBody,
  addBacklinkToBody,
  toggleCardStar,
  togglePartialCardStar,
  saveCard,
  resummarizeCard,
  addTagToCard,
  removeTagFromCard,
  addBacklinkToCard,
  calculateNextChildId,
} from './cardActions';
import { saveExistingCard, starCard, unstarCard } from '../api/cards';
import { findNextChildId } from './cards';
import { Card, PartialCard } from '../models/Card';

vi.mock('../api/cards', () => ({
  saveExistingCard: vi.fn(),
  starCard: vi.fn(),
  unstarCard: vi.fn(),
}));

vi.mock('./cards', () => ({
  findNextChildId: vi.fn(() => 'next-child-id'),
}));

function makeCard(overrides: Partial<Card> = {}): Card {
  return {
    id: 1,
    card_id: 'card-1',
    user_id: 1,
    title: 'Test Card',
    body: 'Some body text',
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

function makePartialCard(overrides: Partial<PartialCard> = {}): PartialCard {
  return {
    id: 2,
    card_id: 'card-2',
    user_id: 1,
    title: 'Child Card',
    parent_id: 1,
    created_at: new Date('2024-01-01'),
    updated_at: new Date('2024-01-01'),
    tags: [],
    ...overrides,
  };
}

describe('addTagToBody', () => {
  it('appends the tag with newline prefix', () => {
    const card = makeCard({ body: 'Existing text' });
    const result = addTagToBody(card, 'important');
    expect(result.body).toBe('Existing text\n\n#important');
  });

  it('does not mutate the original card', () => {
    const card = makeCard();
    const result = addTagToBody(card, 'tag');
    expect(card.body).toBe('Some body text');
    expect(result).not.toBe(card);
  });
});

describe('removeTagFromBody', () => {
  it('removes a tag preceded by newlines', () => {
    const card = makeCard({ body: 'text\n\n#oldtag and more' });
    expect(removeTagFromBody(card, 'oldtag').body).toBe('text and more');
  });

  it('removes the tag at the very start of the body', () => {
    const card = makeCard({ body: '#oldtag rest' });
    expect(removeTagFromBody(card, 'oldtag').body).toBe(' rest');
  });

  it('does not remove a tag that is a substring of another word', () => {
    const card = makeCard({ body: 'text #oldtag2 #oldtag' });
    const result = removeTagFromBody(card, 'oldtag').body;
    expect(result).toContain('#oldtag2');
    expect(result).not.toContain('#oldtag ');
  });

  it('leaves the body unchanged when the tag is absent', () => {
    const card = makeCard({ body: 'no tags here' });
    expect(removeTagFromBody(card, 'missing').body).toBe('no tags here');
  });
});

describe('addBacklinkToBody', () => {
  it('appends a backlink with card_id and placeholder text', () => {
    const card = makeCard({ body: 'text' });
    const target = makePartialCard({ card_id: 'target-card' });
    const result = addBacklinkToBody(card, target);
    expect(result.body).toBe('text\n\n[[target-card|*|]]');
  });
});

describe('toggleCardStar', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('unstars a card that is currently starred', async () => {
    const card = makeCard({ is_starred: true } as any);
    vi.mocked(unstarCard).mockResolvedValue(null as any);
    const refresh = vi.fn();

    await toggleCardStar(card, refresh);

    expect(unstarCard).toHaveBeenCalledWith(1);
    expect(starCard).not.toHaveBeenCalled();
    expect(refresh).toHaveBeenCalledTimes(1);
  });

  it('stars a card that is not starred', async () => {
    const card = makeCard({ is_starred: false } as any);
    vi.mocked(starCard).mockResolvedValue(null as any);

    await toggleCardStar(card);

    expect(starCard).toHaveBeenCalledWith(1);
    expect(unstarCard).not.toHaveBeenCalled();
  });

  it('skips the refresh callback when not provided', async () => {
    const card = makeCard({ is_starred: true } as any);
    vi.mocked(unstarCard).mockResolvedValue(null as any);

    await expect(toggleCardStar(card)).resolves.toBeUndefined();
  });

  it('logs and rethrows when the API call fails', async () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    const card = makeCard({ is_starred: false } as any);
    const error = new Error('star failed');
    vi.mocked(starCard).mockRejectedValue(error);

    await expect(toggleCardStar(card)).rejects.toThrow('star failed');
    expect(consoleSpy).toHaveBeenCalled();

    consoleSpy.mockRestore();
  });
});

describe('togglePartialCardStar', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('stars an unstarred partial card and calls refresh', async () => {
    const card = makePartialCard({ is_starred: false } as any);
    vi.mocked(starCard).mockResolvedValue(null as any);
    const refresh = vi.fn();

    await togglePartialCardStar(card, refresh);

    expect(starCard).toHaveBeenCalledWith(2);
    expect(refresh).toHaveBeenCalledTimes(1);
  });

  it('rethrows errors', async () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    const card = makePartialCard({ is_starred: true } as any);
    vi.mocked(unstarCard).mockRejectedValue(new Error('boom'));

    await expect(togglePartialCardStar(card)).rejects.toThrow('boom');

    consoleSpy.mockRestore();
  });
});

describe('saveCard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('saves the card via saveExistingCard', async () => {
    const card = makeCard();
    vi.mocked(saveExistingCard).mockResolvedValue(card);

    await saveCard(card);

    expect(saveExistingCard).toHaveBeenCalledWith(card);
  });

  it('calls refresh after saving when provided', async () => {
    const card = makeCard();
    vi.mocked(saveExistingCard).mockResolvedValue(card);
    const refresh = vi.fn();

    await saveCard(card, refresh);

    expect(refresh).toHaveBeenCalledTimes(1);
  });
});

describe('resummarizeCard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('sets process_entities_and_facts and saves', async () => {
    const card = makeCard();
    vi.mocked(saveExistingCard).mockResolvedValue(card);

    await resummarizeCard(card);

    expect(saveExistingCard).toHaveBeenCalledWith(
      expect.objectContaining({ process_entities_and_facts: true }),
    );
  });

  it('does not mutate the input card', async () => {
    const card = makeCard();
    vi.mocked(saveExistingCard).mockResolvedValue(card);

    await resummarizeCard(card);

    expect(card).not.toHaveProperty('process_entities_and_facts');
  });
});

describe('tag/backlink save helpers', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('addTagToCard appends the tag and saves', async () => {
    const card = makeCard();
    vi.mocked(saveExistingCard).mockResolvedValue(card);

    await addTagToCard(card, 'newtag');

    expect(saveExistingCard).toHaveBeenCalledWith(
      expect.objectContaining({ body: 'Some body text\n\n#newtag' }),
    );
  });

  it('removeTagFromCard removes the tag and saves', async () => {
    const card = makeCard({ body: 'text\n\n#oldtag' });
    vi.mocked(saveExistingCard).mockResolvedValue(card);

    await removeTagFromCard(card, 'oldtag');

    expect(saveExistingCard).toHaveBeenCalledWith(
      expect.objectContaining({ body: 'text' }),
    );
  });

  it('addBacklinkToCard appends the backlink and saves', async () => {
    const card = makeCard();
    const target = makePartialCard({ card_id: 'ref-card' });
    vi.mocked(saveExistingCard).mockResolvedValue(card);

    await addBacklinkToCard(card, target);

    expect(saveExistingCard).toHaveBeenCalledWith(
      expect.objectContaining({ body: 'Some body text\n\n[[ref-card|*|]]' }),
    );
  });
});

describe('calculateNextChildId', () => {
  it('delegates to findNextChildId', () => {
    const children = [makePartialCard()];
    expect(calculateNextChildId('parent', children)).toBe('next-child-id');
    expect(findNextChildId).toHaveBeenCalledWith('parent', children);
  });
});
