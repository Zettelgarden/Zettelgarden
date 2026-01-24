// @vitest-environment happy-dom

import React from 'react';
import { screen, fireEvent, waitFor, cleanup } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { CardItem } from './CardItem';
import { samplePartialCards } from '../../tests/data';
import { renderWithProviders } from '../../tests/utils';

// Mock the CardPreviewWindow component to avoid complex dependencies
vi.mock('./CardPreviewWindow', () => ({
  CardPreviewWindow: ({ cardPK, mousePosition }: { cardPK: number; mousePosition: { x: number; y: number } }) => (
    <div data-testid="card-preview-window" data-card-pk={cardPK} data-mouse-x={mousePosition.x} data-mouse-y={mousePosition.y}>
      Preview for card {cardPK}
    </div>
  ),
}));

// Mock the CardLink component to focus on CardItem behavior
vi.mock('./CardLink', () => ({
  CardLink: ({ card, showTitle }: { card: any; showTitle: boolean }) => (
    <div data-testid="card-link" data-card-id={card.card_id} data-show-title={showTitle}>
      {card.title}
    </div>
  ),
}));

describe('CardItem', () => {
  const sampleCards = samplePartialCards();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it('should render without crashing', () => {
    const card = sampleCards[0];
    renderWithProviders(<CardItem card={card} />);

    expect(screen.getByTestId('card-link')).toBeTruthy();
    expect(screen.getByText(card.title)).toBeTruthy();
  });

  it('should render card with correct props', () => {
    const card = sampleCards[0];
    renderWithProviders(<CardItem card={card} />);

    const cardLink = screen.getByTestId('card-link');
    expect(cardLink.getAttribute('data-card-id')).toBe(card.card_id);
    expect(cardLink.getAttribute('data-show-title')).toBe('true');
    expect(cardLink.textContent).toBe(card.title);
  });

  it('should render different cards with different data', () => {
    const card1 = sampleCards[0];
    const card2 = sampleCards[1];

    const { rerender } = renderWithProviders(<CardItem card={card1} />);
    expect(screen.getByText(card1.title)).toBeTruthy();

    rerender(<CardItem card={card2} />);
    expect(screen.getByText(card2.title)).toBeTruthy();
    expect(screen.queryByText(card1.title)).toBeNull();
  });

  it('should render card with long title', () => {
    const longTitleCard = sampleCards[2]; // This has a long title in our sample data
    renderWithProviders(<CardItem card={longTitleCard} />);

    expect(screen.getByText(longTitleCard.title)).toBeTruthy();
  });

  it('should not show preview window initially', () => {
    const card = sampleCards[0];
    renderWithProviders(<CardItem card={card} />);

    expect(screen.queryByTestId('card-preview-window')).toBeNull();
  });

  it('should show preview window on mouse enter', async () => {
    const card = sampleCards[0];
    renderWithProviders(<CardItem card={card} />);

    const hoverTarget = screen.getByTestId('card-link').parentElement!;

    fireEvent.mouseEnter(hoverTarget, {
      clientX: 100,
      clientY: 200,
    });

    await waitFor(() => {
      expect(screen.getByTestId('card-preview-window')).toBeTruthy();
    });

    const previewWindow = screen.getByTestId('card-preview-window');
    expect(previewWindow.getAttribute('data-card-pk')).toBe(card.id.toString());
    expect(previewWindow.getAttribute('data-mouse-x')).toBe('100');
    expect(previewWindow.getAttribute('data-mouse-y')).toBe('200');
  });

  it('should hide preview window on mouse leave', async () => {
    const card = sampleCards[0];
    renderWithProviders(<CardItem card={card} />);

    const hoverTarget = screen.getByTestId('card-link').parentElement!;

    // Show preview window
    fireEvent.mouseEnter(hoverTarget, {
      clientX: 100,
      clientY: 200,
    });

    await waitFor(() => {
      expect(screen.getByTestId('card-preview-window')).toBeTruthy();
    });

    // Hide preview window
    fireEvent.mouseLeave(hoverTarget);

    await waitFor(() => {
      expect(screen.queryByTestId('card-preview-window')).toBeNull();
    });
  });

  it('should update mouse position when entering at different coordinates', async () => {
    const card = sampleCards[0];
    renderWithProviders(<CardItem card={card} />);

    const hoverTarget = screen.getByTestId('card-link').parentElement!;

    // First hover
    fireEvent.mouseEnter(hoverTarget, {
      clientX: 150,
      clientY: 250,
    });

    await waitFor(() => {
      const previewWindow = screen.getByTestId('card-preview-window');
      expect(previewWindow.getAttribute('data-mouse-x')).toBe('150');
      expect(previewWindow.getAttribute('data-mouse-y')).toBe('250');
    });

    // Hide and show again with different coordinates
    fireEvent.mouseLeave(hoverTarget);

    await waitFor(() => {
      expect(screen.queryByTestId('card-preview-window')).toBeNull();
    });

    fireEvent.mouseEnter(hoverTarget, {
      clientX: 300,
      clientY: 400,
    });

    await waitFor(() => {
      const previewWindow = screen.getByTestId('card-preview-window');
      expect(previewWindow.getAttribute('data-mouse-x')).toBe('300');
      expect(previewWindow.getAttribute('data-mouse-y')).toBe('400');
    });
  });

  it('should have correct CSS classes', () => {
    const card = sampleCards[0];
    renderWithProviders(<CardItem card={card} />);

    const cardLink = screen.getByTestId('card-link');
    // Navigate up to the outer container div (CardItem's div)
    const cardItemContainer = cardLink.parentElement?.parentElement;
    expect(cardItemContainer).toBeTruthy();
    expect(cardItemContainer?.className).toContain('py-2');
    expect(cardItemContainer?.className).toContain('px-2.5');
  });

  it('should handle cards with no tags', () => {
    const cardWithoutTags = sampleCards[1]; // This card has no tags
    renderWithProviders(<CardItem card={cardWithoutTags} />);

    expect(screen.getByText(cardWithoutTags.title)).toBeTruthy();
    expect(screen.getByTestId('card-link')).toBeTruthy();
  });
});