import React from 'react';
import { CardItem } from './CardItem';
import { UnlinkedMention } from '../../models/Card';

interface UnlinkedMentionsProps {
  mentions: UnlinkedMention[];
  onCardClick: (cardId: number) => void;
  onAddLink: (mention: UnlinkedMention) => void;
}

export function UnlinkedMentions({
  mentions,
  onCardClick,
  onAddLink,
}: UnlinkedMentionsProps) {
  if (mentions.length === 0) {
    return (
      <div className="text-gray-400 text-sm">
        No unlinked mentions. Cards that reference this card without linking to
        it will appear here.
      </div>
    );
  }

  return (
    <ul className="space-y-1">
      {mentions.map((mention) => (
        <li key={mention.card.id} className="cursor-pointer group">
          <div className="flex items-center justify-between">
            <div
              className="flex-1"
              onClick={() => onCardClick(mention.card.id)}
            >
              <CardItem card={mention.card} />
            </div>
            <div className="flex items-center gap-2 ml-2 shrink-0">
              <span
                className="text-xs text-gray-400"
                title={`${mention.mention_count} mention(s)`}
              >
                {mention.mention_count}x
              </span>
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  onAddLink(mention);
                }}
                className="opacity-0 group-hover:opacity-100 text-xs text-blue-500 hover:text-blue-700 border border-blue-300 hover:border-blue-500 rounded px-1.5 py-0.5 transition-opacity"
                title="Insert a link to this card"
              >
                +Link
              </button>
            </div>
          </div>
          {mention.context_snippet && (
            <div className="mt-0.5 px-2 text-xs text-gray-500 italic line-clamp-2">
              {mention.context_snippet}
            </div>
          )}
        </li>
      ))}
    </ul>
  );
}
