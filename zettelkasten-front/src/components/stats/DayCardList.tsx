import React from 'react';
import { PartialCard } from '../../models/Card';
import { Link } from 'react-router-dom';

interface DayCardListProps {
  cards: PartialCard[];
  date: Date;
  onClose: () => void;
}

function formatTime(date: Date): string {
  return date.toLocaleTimeString('en-US', {
    hour: 'numeric',
    minute: '2-digit',
    hour12: true,
  });
}

function formatDate(date: Date): string {
  return date.toLocaleDateString('en-US', {
    weekday: 'long',
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  });
}

export function DayCardList({ cards, date, onClose }: DayCardListProps) {
  return (
    <div className="bg-white rounded-lg shadow p-4 mt-4">
      <div className="flex justify-between items-center mb-3">
        <h2 className="text-lg font-semibold text-gray-800">
          {formatDate(date)} ({cards.length} card{cards.length !== 1 ? 's' : ''}{' '}
          created)
        </h2>
        <button
          onClick={onClose}
          className="text-gray-400 hover:text-gray-600 text-xl leading-none"
          aria-label="Close"
        >
          &times;
        </button>
      </div>

      {cards.length === 0 ? (
        <div className="text-center py-4 text-sm text-gray-500">
          No cards were created on this day.
        </div>
      ) : (
        <div className="divide-y divide-gray-100">
          {cards.map((card) => (
            <div
              key={card.id}
              className="py-2 hover:bg-gray-50 -mx-4 px-4 transition-colors"
            >
              <div className="flex items-baseline gap-2 flex-wrap">
                <Link
                  to={`/app/card/${card.id}`}
                  className="text-sm font-medium text-blue-600 hover:text-blue-800 hover:underline"
                >
                  {card.card_id} - {card.title}
                </Link>
                {card.created_at && (
                  <span className="text-xs text-gray-500">
                    {formatTime(card.created_at)}
                  </span>
                )}
              </div>
              {card.tags && card.tags.length > 0 && (
                <div className="flex flex-wrap gap-1 mt-1">
                  {card.tags.map((tag) => (
                    <span
                      key={tag.id}
                      className="px-1.5 py-0.5 bg-gray-100 text-gray-600 rounded text-xs"
                    >
                      #{tag.name}
                    </span>
                  ))}
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
