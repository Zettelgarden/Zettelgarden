import React from 'react';
import { CardReference, ParsedMessageContent } from '../models/Chat';

import { Card } from '../models/Card';

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
  let cards: Card[] = [];
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
        React.createElement('span', { key: `text-${index}` },
          text.substring(lastIndex, ref.startIndex)
        )
      );
    }

    // Add the card link
    elements.push(
      React.createElement('button', {
        key: `card-${index}`,
        onClick: () => onCardClick?.(ref.cardId),
        className: "inline-flex items-center gap-1 px-2 py-1 mx-1 text-sm font-medium text-blue-700 bg-blue-50 border border-blue-200 rounded-lg hover:bg-blue-100 hover:border-blue-300 transition-colors duration-200 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-opacity-50",
        title: `Open card: ${ref.title}`,
      }, [
        React.createElement('svg', {
          key: 'icon',
          className: "w-3 h-3",
          fill: "none",
          stroke: "currentColor",
          viewBox: "0 0 24 24"
        }, React.createElement('path', {
          strokeLinecap: "round",
          strokeLinejoin: "round",
          strokeWidth: 2,
          d: "M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z"
        })),
        React.createElement('span', { key: 'text' }, ref.title)
      ])
    );

    lastIndex = ref.endIndex;
  });

  // Add any remaining text after the last reference
  if (lastIndex < text.length) {
    elements.push(
      React.createElement('span', { key: 'text-final' },
        text.substring(lastIndex)
      )
    );
  }

  return React.createElement(React.Fragment, {}, ...elements);
}