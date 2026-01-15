import { renderHook } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { Card, PartialCard } from '../models/Card';
import { useCardNavigation } from './useCardNavigation';

// Mock the compareCardIds function to return predictable results
vi.mock('../utils/cards', () => ({
  compareCardIds: vi.fn((a: string, b: string) => a.localeCompare(b)),
}));

describe('useCardNavigation', () => {
  const mockPartialCard = (card_id: string, id: number, title: string): PartialCard => ({
    card_id,
    id,
    title,
    user_id: 1,
    parent_id: 1,
    created_at: new Date(),
    updated_at: new Date(),
    tags: [],
  });

  const mockCard = (id: number, children: PartialCard[] = []): Card => ({
    id,
    card_id: `card-${id}`,
    title: `Card ${id}`,
    body: '',
    link: '',
    is_deleted: false,
    created_at: new Date(),
    updated_at: new Date(),
    parent_id: 0,
    parent: mockPartialCard('parent', 0, 'Parent Card'),
    files: [],
    children,
    references: [],
    tags: [],
    tasks: [],
    entities: [],
    user_id: 1,
    is_starred: false,
  });

  it('returns null for both prev and next when no parent card', () => {
    const { result } = renderHook(() =>
      useCardNavigation(null, mockCard(1))
    );

    expect(result.current.prevSibling).toBeNull();
    expect(result.current.nextSibling).toBeNull();
  });

  it('returns null for both prev and next when no viewing card', () => {
    const { result } = renderHook(() =>
      useCardNavigation(mockCard(1), null)
    );

    expect(result.current.prevSibling).toBeNull();
    expect(result.current.nextSibling).toBeNull();
  });

  it('returns null for both prev and next when viewing card is not in parent children', () => {
    const parentCard = mockCard(1, [
      mockPartialCard('1/A', 11, 'Child A'),
      mockPartialCard('1/B', 12, 'Child B'),
    ]);
    const viewingCard = mockCard(99); // Not in parent's children

    const { result } = renderHook(() =>
      useCardNavigation(parentCard, viewingCard)
    );

    expect(result.current.prevSibling).toBeNull();
    expect(result.current.nextSibling).toBeNull();
  });

  it('returns correct prev and next siblings for middle child', () => {
    const children = [
      mockPartialCard('1/A', 11, 'Child A'),
      mockPartialCard('1/B', 12, 'Child B'),
      mockPartialCard('1/C', 13, 'Child C'),
    ];
    const parentCard = mockCard(1, children);
    const viewingCard = mockCard(12); // Child B

    const { result } = renderHook(() =>
      useCardNavigation(parentCard, viewingCard)
    );

    expect(result.current.prevSibling?.id).toBe(11); // Child A
    expect(result.current.nextSibling?.id).toBe(13); // Child C
  });

  it('returns null prev for first child', () => {
    const children = [
      mockPartialCard('1/A', 11, 'Child A'),
      mockPartialCard('1/B', 12, 'Child B'),
    ];
    const parentCard = mockCard(1, children);
    const viewingCard = mockCard(11); // Child A

    const { result } = renderHook(() =>
      useCardNavigation(parentCard, viewingCard)
    );

    expect(result.current.prevSibling).toBeNull();
    expect(result.current.nextSibling?.id).toBe(12); // Child B
  });

  it('returns null next for last child', () => {
    const children = [
      mockPartialCard('1/A', 11, 'Child A'),
      mockPartialCard('1/B', 12, 'Child B'),
    ];
    const parentCard = mockCard(1, children);
    const viewingCard = mockCard(12); // Child B

    const { result } = renderHook(() =>
      useCardNavigation(parentCard, viewingCard)
    );

    expect(result.current.prevSibling?.id).toBe(11); // Child A
    expect(result.current.nextSibling).toBeNull();
  });

  it('sorts children correctly using compareCardIds', () => {
    const children = [
      mockPartialCard('1/B', 12, 'Child B'),
      mockPartialCard('1/A', 11, 'Child A'), // This should come first after sorting
    ];
    const parentCard = mockCard(1, children);
    const viewingCard = mockCard(11); // Child A (sorted first)

    const { result } = renderHook(() =>
      useCardNavigation(parentCard, viewingCard)
    );

    expect(result.current.prevSibling).toBeNull(); // Child A is first after sorting
    expect(result.current.nextSibling?.id).toBe(12); // Child B is next
  });
});