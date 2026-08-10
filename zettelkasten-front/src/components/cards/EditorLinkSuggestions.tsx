import React, { useEffect, useState } from 'react';
import { Card, RelatedCard } from '../../models/Card';
import { getRelatedCards } from '../../api/cards';
import { CardItem } from './CardItem';

interface EditorLinkSuggestionsProps {
  card: Card;
  newCard: boolean;
  onInsertLink: (cardId: string, title: string) => void;
}

const DEBOUNCE_MS = 600;
const MAX_SUGGESTIONS = 5;

/**
 * Slim "Link these" strip under the card editor. While typing, it debounces
 * related-card lookups and offers one-click wiki-link insertion. Only shown
 * for existing cards (new unsaved cards have no id to look up against).
 */
export function EditorLinkSuggestions({
  card,
  newCard,
  onInsertLink,
}: EditorLinkSuggestionsProps) {
  const [suggestions, setSuggestions] = useState<RelatedCard[] | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    // No id yet (new cards) -> never show the strip.
    if (newCard || !card.id || card.id <= 0) {
      setSuggestions(null);
      setLoading(false);
      return;
    }

    setLoading(true);
    const timer = setTimeout(() => {
      getRelatedCards(card.id.toString())
        .then((cards) => {
          // Exclude cards already linked in the unsaved body.
          const linkedIds = new Set(
            cards
              .filter(
                (rc) =>
                  card.body.includes(`[[${rc.card.card_id}]]`) ||
                  card.body.includes(`[[${rc.card.card_id}|`),
              )
              .map((rc) => rc.card.id),
          );
          setSuggestions(
            cards
              .filter((rc) => !linkedIds.has(rc.card.id))
              .slice(0, MAX_SUGGESTIONS),
          );
        })
        .catch((err) => {
          console.error('Failed to fetch link suggestions:', err);
          setSuggestions(null);
        })
        .finally(() => setLoading(false));
    }, DEBOUNCE_MS);

    return () => clearTimeout(timer);
  }, [card.body, card.title, card.id, newCard]);

  if (newCard || !card.id || card.id <= 0) {
    return null;
  }

  if (suggestions === null || suggestions.length === 0) {
    return null;
  }

  return (
    <div className="mt-2">
      <div className="text-xs font-medium text-gray-500 mb-1">
        Link these
        {loading && <span className="text-gray-400 ml-1">(refreshing…)</span>}
      </div>
      <div className="flex flex-wrap gap-1">
        {suggestions.map((rc) => (
          <div
            key={rc.card.id}
            role="button"
            tabIndex={0}
            onClick={(e) => {
              // CardItem renders an <a>; swallow it so the click inserts
              // instead of navigating.
              e.preventDefault();
              onInsertLink(rc.card.card_id, rc.card.title);
            }}
            onKeyDown={(e) => {
              if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                onInsertLink(rc.card.card_id, rc.card.title);
              }
            }}
            className="cursor-pointer border border-gray-200 rounded hover:border-blue-300 hover:bg-blue-50 transition-colors"
            title={`Insert [[${rc.card.card_id}|${rc.card.title}]]`}
          >
            <CardItem card={rc.card} />
          </div>
        ))}
      </div>
    </div>
  );
}
