import React from 'react';

export interface CardReference {
  cardId: string;
  title: string;
  fullMatch: string;
  startIndex: number;
  endIndex: number;
}

export interface CardData {
  id: number;
  card_id: string;
  title: string;
  body_preview?: string;
  created_at: string;
  updated_at: string;
  tags?: string[];
}

export interface ParsedMessageContent {
  text: string;
  cards: CardData[];
}

/**
 * Parses card references from text in the format: [Card: card_id | title]
 * Returns an array of CardReference objects with position information
 */
export function parseCardReferences(text: string): CardReference[] {
  const cardReferenceRegex = /\[Card:\s*([^|]+?)\s*\|\s*([^\]]+?)\s*\]/g;
  const references: CardReference[] = [];
  let match;

  while ((match = cardReferenceRegex.exec(text)) !== null) {
    references.push({
      cardId: match[1].trim(),
      title: match[2].trim(),
      fullMatch: match[0],
      startIndex: match.index,
      endIndex: match.index + match[0].length,
    });
  }

  return references;
}

/**
 * Parses a message that may contain both text and a JSON cards section
 * Returns the separated text content and parsed card data
 */
export function parseMessageContent(content: string): ParsedMessageContent {
  const cardsSectionRegex = /---CARDS---\s*```json\s*([\s\S]*?)\s*```/;
  const match = content.match(cardsSectionRegex);

  if (!match) {
    return {
      text: content,
      cards: [],
    };
  }

  // Extract text without the cards section
  const text = content.substring(0, match.index).trim();

  // Parse the JSON cards data
  let cards: CardData[] = [];
  try {
    const jsonData = JSON.parse(match[1]);
    if (jsonData.cards && Array.isArray(jsonData.cards)) {
      cards = jsonData.cards;
    }
  } catch (error) {
    console.error('Failed to parse cards JSON:', error);
  }

  return {
    text,
    cards,
  };
}

/**
 * Converts text with card references into JSX with clickable card links
 */
export function renderTextWithCardLinks(text: string, onCardClick?: (cardId: string) => void): React.ReactNode {
  const references = parseCardReferences(text);

  if (references.length === 0) {
    return text;
  }

  const elements: React.ReactNode[] = [];
  let lastIndex = 0;

  references.forEach((ref, index) => {
    // Add text before this reference
    if (ref.startIndex > lastIndex) {
      elements.push(
        <span key={`text-${index}`}>
          {text.substring(lastIndex, ref.startIndex)}
        </span>
      );
    }

    // Add the card link
    elements.push(
      <button
        key={`card-${index}`}
        onClick={() => onCardClick?.(ref.cardId)}
        className="inline-flex items-center gap-1 px-2 py-1 mx-1 text-sm font-medium text-blue-700 bg-blue-50 border border-blue-200 rounded-lg hover:bg-blue-100 hover:border-blue-300 transition-colors duration-200 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-opacity-50"
        title={`Open card: ${ref.title}`}
      >
        <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
        </svg>
        <span>{ref.title}</span>
      </button>
    );

    lastIndex = ref.endIndex;
  });

  // Add any remaining text after the last reference
  if (lastIndex < text.length) {
    elements.push(
      <span key="text-final">
        {text.substring(lastIndex)}
      </span>
    );
  }

  return <>{elements}</>;
}

/**
 * Rich card component for displaying detailed card information
 */
export function CardPreview({ card, onCardClick }: { card: CardData; onCardClick?: (cardId: string) => void }) {

  return (
    <div className="bg-white border border-gray-200 rounded-lg p-3 shadow-sm hover:shadow-md transition-shadow duration-200">
      <div className="flex items-center justify-between">
        <div className="flex-1">
          <h3 className="font-medium text-gray-900 text-sm">
            <span className="text-blue-600 font-mono">[{card.card_id}]</span> - {card.title}
          </h3>
        </div>
        <button
          onClick={() => onCardClick?.(String(card.id))}
          className="ml-2 px-2 py-1 text-xs font-medium text-blue-700 bg-blue-50 border border-blue-200 rounded hover:bg-blue-100 hover:border-blue-300 transition-colors duration-200 flex items-center gap-1"
          title={`Open card: ${card.title}`}
        >
          <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
          </svg>
          Open
        </button>
      </div>
    </div>
  );
}

/**
 * Container for displaying multiple card previews
 */
export function CardsSection({ cards, onCardClick }: { cards: CardData[]; onCardClick?: (cardId: string) => void }) {
  if (cards.length === 0) return null;

  return (
    <div className="mt-4 p-4 bg-gradient-to-br from-blue-50 to-indigo-50 border border-blue-200 rounded-lg">
      <div className="flex items-center gap-2 mb-3">
        <svg className="w-4 h-4 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
        </svg>
        <h4 className="text-sm font-semibold text-blue-900">
          {cards.length === 1 ? 'Referenced Card' : `Referenced Cards (${cards.length})`}
        </h4>
      </div>
      <div className="grid gap-3">
        {cards.map((card) => (
          <CardPreview
            key={card.id}
            card={card}
            onCardClick={onCardClick}
          />
        ))}
      </div>
    </div>
  );
}