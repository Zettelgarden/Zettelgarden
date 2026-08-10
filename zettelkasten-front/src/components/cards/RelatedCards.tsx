import React from 'react';
import { HeaderSubSection } from '../Header';
import { CardItem } from './CardItem';
import { RelatedCard } from '../../models/Card';

interface RelatedCardsProps {
  relatedCards: RelatedCard[];
  onCardClick: (cardId: number) => void;
  onAddReference?: (card: RelatedCard) => void;
  title?: string;
}

type ReasonCategory = 'entities' | 'tags' | 'similarity';

const CATEGORY_LABELS: { key: ReasonCategory; label: string }[] = [
  { key: 'entities', label: 'Shared entities' },
  { key: 'tags', label: 'Shared tags' },
  { key: 'similarity', label: 'Semantically similar' },
];

// Map a backend reason string (e.g. "3 shared entities: Python, LLM",
// "1 shared tag: research", "semantically similar") to its category.
function reasonCategory(reason: string): ReasonCategory | null {
  if (reason.includes('shared entit')) {
    return 'entities';
  }
  if (reason.includes('shared tag')) {
    return 'tags';
  }
  if (reason === 'semantically similar') {
    return 'similarity';
  }
  return null;
}

// A card's primary category is the first recognizable reason, which follows
// the backend merge order (entities, tags, similarity). This avoids
// duplicating scoring-weight logic on the client.
function primaryCategory(rc: RelatedCard): ReasonCategory | null {
  for (const reason of rc.reasons || []) {
    const cat = reasonCategory(reason);
    if (cat) {
      return cat;
    }
  }
  return null;
}

export function RelatedCards({
  relatedCards,
  onCardClick,
  onAddReference,
  title = 'Related Cards',
}: RelatedCardsProps) {
  if (relatedCards.length === 0) {
    return null;
  }

  const categories = Array.from(
    new Set(
      relatedCards
        .map((rc) => primaryCategory(rc))
        .filter((c): c is ReasonCategory => c !== null),
    ),
  );
  // Only group when there is more than one distinct reason category; a
  // single-category list stays flat to avoid clutter.
  const shouldGroup = categories.length > 1;

  const renderCard = (rc: RelatedCard) => (
    <li key={rc.card.id} className="cursor-pointer group">
      <div className="flex items-center justify-between">
        <div className="flex-1" onClick={() => onCardClick(rc.card.id)}>
          <CardItem card={rc.card} />
        </div>
        <div className="flex items-center gap-1 ml-2 shrink-0">
          {onAddReference && (
            <button
              onClick={(e) => {
                e.stopPropagation();
                onAddReference(rc);
              }}
              className="opacity-0 group-hover:opacity-100 text-xs text-blue-500 hover:text-blue-700 border border-blue-300 hover:border-blue-500 rounded px-1.5 py-0.5 transition-opacity"
              title="Add as reference"
            >
              +Ref
            </button>
          )}
          {rc.score > 0 && (
            <span className="text-xs text-gray-400">{rc.score.toFixed(1)}</span>
          )}
        </div>
      </div>
      {rc.reasons && rc.reasons.length > 0 && (
        <div
          className="mt-1 flex flex-wrap gap-1 px-2"
          onClick={() => onCardClick(rc.card.id)}
        >
          {rc.reasons.map((reason) => (
            <span
              key={reason}
              className="inline-flex items-center rounded-full bg-blue-50 text-blue-700 text-[10px] leading-4 px-2 py-0.5 border border-blue-100 max-w-full truncate"
              title={reason}
            >
              {reason}
            </span>
          ))}
        </div>
      )}
    </li>
  );

  return (
    <div>
      <HeaderSubSection text={title} />
      {shouldGroup ? (
        <div className="mt-2 space-y-3">
          {CATEGORY_LABELS.filter(({ key }) => categories.includes(key)).map(
            ({ key, label }) => {
              const groupCards = relatedCards.filter(
                (rc) => primaryCategory(rc) === key,
              );
              if (groupCards.length === 0) {
                return null;
              }
              return (
                <div key={key}>
                  <h3 className="text-xs font-medium text-gray-600 mb-1">
                    {label} ({groupCards.length})
                  </h3>
                  <ul className="space-y-1">{groupCards.map(renderCard)}</ul>
                </div>
              );
            },
          )}
          {(() => {
            const other = relatedCards.filter(
              (rc) => primaryCategory(rc) === null,
            );
            if (other.length === 0) {
              return null;
            }
            return <ul className="space-y-1">{other.map(renderCard)}</ul>;
          })()}
        </div>
      ) : (
        <ul className="mt-2 space-y-1">{relatedCards.map(renderCard)}</ul>
      )}
      <hr className="my-4" />
    </div>
  );
}
