/**
 * Tests for useEntityData (src/hooks/useEntityData.ts)
 *
 * The hook fetches associated cards, facts, and similar entities when a
 * dialog is shown for an entity, and exposes per-section loading/error state.
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { useEntityData } from './useEntityData';
import { semanticSearchCards, escapeEntityNameForSearch } from '../api/cards';
import { getEntityFacts, getSimilarEntities } from '../api/entities';
import { Entity } from '../models/Card';

vi.mock('../api/cards', () => ({
  semanticSearchCards: vi.fn(),
  escapeEntityNameForSearch: vi.fn((name: string) => name),
}));

vi.mock('../api/entities', () => ({
  getEntityFacts: vi.fn(),
  getSimilarEntities: vi.fn(),
}));

const mockEntity: Entity = {
  id: 42,
  user_id: 1,
  name: 'Zettelkasten',
  description: 'A note-taking method',
  type: 'concept',
  created_at: new Date('2024-01-01'),
  updated_at: new Date('2024-01-01'),
  card_count: 0,
  card_pk: null,
};

beforeEach(() => {
  vi.clearAllMocks();
});

describe('useEntityData', () => {
  it('does not fetch anything when the dialog is closed', async () => {
    const { result } = renderHook(() => useEntityData(false, mockEntity));

    expect(result.current.associatedCards).toEqual([]);
    expect(result.current.facts).toEqual([]);
    expect(result.current.similarEntities).toEqual([]);

    // Give any stray effects a chance to run, then confirm no API calls
    await new Promise((r) => setTimeout(r, 10));
    expect(semanticSearchCards).not.toHaveBeenCalled();
    expect(getEntityFacts).not.toHaveBeenCalled();
    expect(getSimilarEntities).not.toHaveBeenCalled();
  });

  it('does not fetch when there is no entity', async () => {
    const { result } = renderHook(() => useEntityData(true, null));

    expect(result.current.isLoading).toBe(false);
    await new Promise((r) => setTimeout(r, 10));
    expect(semanticSearchCards).not.toHaveBeenCalled();
  });

  it('fetches associated cards, facts and similar entities for an entity', async () => {
    vi.mocked(semanticSearchCards).mockResolvedValue([
      {
        id: '9',
        type: 'card',
        title: 'My note',
        preview: 'preview text',
        score: 1,
        created_at: new Date('2024-01-01'),
        updated_at: new Date('2024-01-01'),
        tags: [{ id: 1, name: 'tag', color: 'black', user_id: 1 }],
        metadata: { id: 9, card_id: 'my-note', parent_id: 0 },
      },
    ]);
    vi.mocked(getEntityFacts).mockResolvedValue([
      { id: 1, fact: 'fact text', card_id: 9 } as any,
    ]);
    vi.mocked(getSimilarEntities).mockResolvedValue([
      { ...mockEntity, id: 7, name: 'Similar', score: 0.8 },
    ]);

    const { result } = renderHook(() => useEntityData(true, mockEntity));

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    await waitFor(() => expect(result.current.factsLoading).toBe(false));
    await waitFor(() => expect(result.current.loadingSimilar).toBe(false));

    expect(semanticSearchCards).toHaveBeenCalledWith(
      `@[${escapeEntityNameForSearch(mockEntity.name)}]`,
      false,
      false,
      false,
      true,
    );
    expect(getEntityFacts).toHaveBeenCalledWith(42);
    expect(getSimilarEntities).toHaveBeenCalledWith(42);

    expect(result.current.associatedCards).toHaveLength(1);
    expect(result.current.associatedCards[0].card_id).toBe('my-note');
    expect(result.current.associatedCards[0].title).toBe('My note');
    expect(result.current.facts).toHaveLength(1);
    expect(result.current.similarEntities).toHaveLength(1);
    expect(result.current.similarEntities[0].score).toBe(0.8);
  });

  it('maps a null search response to no associated cards', async () => {
    vi.mocked(semanticSearchCards).mockResolvedValue(null as any);
    vi.mocked(getEntityFacts).mockResolvedValue([]);
    vi.mocked(getSimilarEntities).mockResolvedValue([]);

    const { result } = renderHook(() => useEntityData(true, mockEntity));

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.associatedCards).toEqual([]);
  });

  it('sets per-section errors when fetches fail', async () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    vi.mocked(semanticSearchCards).mockRejectedValue(new Error('search down'));
    vi.mocked(getEntityFacts).mockRejectedValue(new Error('facts down'));
    vi.mocked(getSimilarEntities).mockRejectedValue(new Error('similar down'));

    const { result } = renderHook(() => useEntityData(true, mockEntity));

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    await waitFor(() => expect(result.current.factsLoading).toBe(false));
    await waitFor(() => expect(result.current.loadingSimilar).toBe(false));

    expect(result.current.error).toBe('Failed to load associated cards.');
    expect(result.current.factsError).toBe('Failed to load facts.');
    expect(result.current.similarError).toBe('Failed to load similar entities');
    expect(result.current.associatedCards).toEqual([]);
    expect(result.current.facts).toEqual([]);
    expect(result.current.similarEntities).toEqual([]);

    consoleSpy.mockRestore();
  });
});
