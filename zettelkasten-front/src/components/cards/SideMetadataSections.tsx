import React from "react";
import { useNavigate } from "react-router-dom";
import { Card } from "../../models/Card";
import { linkifyWithDefaultOptions } from "../../utils/strings";
import { RSSArticle } from "../../api/rss";

/**
 * Shared side-metadata sub-sections used by both the desktop rail
 * (ViewPageSidePanels) and the mobile accordions (ViewMobileLayout).
 *
 * These intentionally render only the inner content. Each layout wraps them
 * in its own chrome (HeaderSubSection / Collapsible on desktop,
 * ViewMobileAccordion on mobile) so the structure stays layout-specific
 * while the duplicated markup for tags, details, and the source-article link
 * lives in one place.
 *
 * The two layouts previously differed in small ways (ISO vs locale dates, an
 * ID row only on mobile, `×` vs `x` tag-remove buttons, separator glyphs).
 * These components unify both onto the more readable variant: locale dates,
 * the card ID row shown everywhere, the `×` remove button, and the `•`
 * separator.
 */

interface TagsListProps {
  card: Card;
  onRemoveTag: (tagName: string) => void;
  /** Extra classes for the pill row (e.g. "mt-2" to offset a header). */
  className?: string;
}

/** Renders the tag pills row, unified between desktop and mobile. */
export function TagsList({ card, onRemoveTag, className = "" }: TagsListProps) {
  const navigate = useNavigate();
  return (
    <div className={`flex flex-wrap gap-1.5 ${className}`}>
      {card.tags.map((tag) => (
        <span
          key={tag.name}
          className="inline-flex items-center px-1.5 py-0.5 bg-purple-50 text-purple-600 text-xs rounded-full"
        >
          <span
            className="cursor-pointer hover:bg-purple-100"
            onClick={() =>
              navigate(`/app/search?term=${encodeURIComponent("#" + tag.name)}`)
            }
          >
            #{tag.name}
          </span>
          {card.body.includes(`#${tag.name}`) && (
            <button
              onClick={() => onRemoveTag(tag.name)}
              className="ml-1.5 text-purple-400 hover:text-purple-600"
            >
              &times;
            </button>
          )}
        </span>
      ))}
    </div>
  );
}

interface DetailsListProps {
  card: Card;
  /** Extra classes for the container (e.g. "pt-4 border-t" for the rail divider). */
  className?: string;
}

/** Renders the card details rows (ID, link, created/updated), unified. */
export function DetailsList({ card, className = "" }: DetailsListProps) {
  return (
    <div className={`text-xs text-gray-600 space-y-1 ${className}`}>
      <div className="flex items-start">
        <span className="font-medium w-20">ID:</span>
        <span className="flex-1 text-blue-600 font-mono">[{card.card_id}]</span>
      </div>
      {card.link && (
        <div className="flex items-start">
          <span className="font-medium w-20">Link:</span>
          <div
            className="flex-1 break-all"
            dangerouslySetInnerHTML={{
              __html: linkifyWithDefaultOptions(card.link),
            }}
          />
        </div>
      )}
      <div className="flex items-start">
        <span className="font-medium w-20">Created:</span>
        <span className="flex-1">{card.created_at.toLocaleDateString()}</span>
      </div>
      <div className="flex items-start">
        <span className="font-medium w-20">Updated:</span>
        <span className="flex-1">{card.updated_at.toLocaleDateString()}</span>
      </div>
    </div>
  );
}

interface SourceArticleLinkProps {
  sourceArticle: RSSArticle;
}

/** Renders the "jump to RSS source article" button, unified. */
export function SourceArticleLink({ sourceArticle }: SourceArticleLinkProps) {
  const navigate = useNavigate();
  return (
    <button
      onClick={() =>
        navigate("/app/rss", {
          state: { selectedArticleId: sourceArticle.id },
        })
      }
      className="w-full text-left p-2 rounded-md hover:bg-gray-50 transition-colors"
    >
      <div className="flex items-start gap-2">
        <svg
          className="w-4 h-4 text-green-600 flex-shrink-0 mt-0.5"
          fill="currentColor"
          viewBox="0 0 20 20"
        >
          <path d="M7 3a1 1 0 000 2h6a1 1 0 100-2H7zM4 7a1 1 0 011-1h10a1 1 0 110 2H5a1 1 0 01-1-1zM2 11a2 2 0 012-2h12a2 2 0 012 2v4a2 2 0 01-2 2H4a2 2 0 01-2-2v-4z" />
        </svg>
        <div className="flex-1 min-w-0">
          <p className="text-sm font-medium text-blue-600 hover:text-blue-800 line-clamp-2">
            {sourceArticle.title}
          </p>
          <p className="text-xs text-gray-500 mt-1">
            RSS Feed • {new Date(sourceArticle.fetched_at).toLocaleDateString()}
          </p>
        </div>
      </div>
    </button>
  );
}
