import React from "react";
import { useNavigate } from "react-router-dom";
import { Card, PartialCard } from "../../models/Card";
import { Entity } from "../../models/Card";
import { HeaderSubSection } from "../Header";
import { Button } from "../Button";
import { CardItem } from "./CardItem";
import { SearchTagDropdown } from "../tags/SearchTagDropdown";
import { linkifyWithDefaultOptions } from "../../utils/strings";
import { PersonIcon } from "../../assets/icons/PersonIcon";
import { CardStructuredDataDisplay } from "../schemas/CardStructuredDataDisplay";
import { RSSArticle } from "../../api/rss";

interface ViewPageSidePanelsProps {
  parentCard: Card | null;
  prevSibling: PartialCard | null;
  nextSibling: PartialCard | null;
  linkedEntities: Entity[];
  onOpenEntity: (entity: Entity) => void;
  viewingCard: Card;
  tags: any[];
  onTagClick: (tagName: string) => void;
  onRemoveTag: (tagName: string) => void;
  sourceArticle?: RSSArticle;
}

export function ViewPageSidePanels({
  parentCard,
  prevSibling,
  nextSibling,
  linkedEntities,
  onOpenEntity,
  viewingCard,
  tags,
  onTagClick,
  onRemoveTag,
  sourceArticle
}: ViewPageSidePanelsProps) {
  const navigate = useNavigate();

  return (
    <div className="md:w-1/3 bg-white rounded-lg p-4 shadow-sm space-y-4">
      {/* Parent Card Section */}
      {parentCard && (
        <div>
          <HeaderSubSection text="Parent" />
          <div className="mt-2">
            <CardItem card={parentCard} />
          </div>

          {prevSibling && (
            <Button
              onClick={() => navigate(`/app/card/${prevSibling.id}`)}
              variant="secondary"
            >
              ← Prev
            </Button>
          )}
          {parentCard && (
            <Button
              onClick={() => navigate(`/app/card/${parentCard.id}`)}
              variant="secondary"
            >
              ↑ Up
            </Button>
          )}
          {nextSibling && (
            <Button
              onClick={() => navigate(`/app/card/${nextSibling.id}`)}
              variant="secondary"
            >
              Next →
            </Button>
          )}
          <hr className="my-4" />
        </div>
      )}

      {/* Source Article Section */}
      {sourceArticle && (
        <div>
          <HeaderSubSection text="Source Article" />
          <div className="mt-2">
            <button
              onClick={() => navigate('/app/rss', { state: { selectedArticleId: sourceArticle.id } })}
              className="w-full text-left p-2 rounded-md hover:bg-gray-50 transition-colors"
            >
              <div className="flex items-start gap-2">
                <svg className="w-4 h-4 text-green-600 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
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
          </div>
          <hr className="my-4" />
        </div>
      )}

      {/* Linked Entities Section */}
      {linkedEntities && linkedEntities.length > 0 && (
        <div>
          <HeaderSubSection text="Linked Entities" />
          <ul className="mt-2 space-y-1">
            {linkedEntities.map(entity => (
              <li
                key={entity.id}
                className="py-1 px-2 hover:bg-gray-50 rounded cursor-pointer"
                onClick={() => onOpenEntity(entity)}
              >
                <div className="flex items-center gap-2 text-xs">
                  <div className="text-gray-400 shrink-0">
                    <PersonIcon />
                  </div>
                  <span className="text-blue-600 hover:text-blue-800 shrink-0">
                    {entity.name}
                  </span>
                  <span className="text-gray-300">-</span>
                  <span className="text-gray-500 shrink-0">
                    {entity.type}
                  </span>
                  <span className="text-gray-300">-</span>
                  <span className="text-gray-600 truncate">
                    {entity.description || '(no description)'}
                  </span>
                </div>
              </li>
            ))}
          </ul>
          <hr className="my-4" />
        </div>
      )}

      <div>

        <CardStructuredDataDisplay
          schemaId={viewingCard.schema_id}
          structuredData={viewingCard.structured_data}
        />
      </div>

      {/* Tags Section */}
      <div>
        <div className="flex items-center justify-between">
          <HeaderSubSection text="Tags" />
          <SearchTagDropdown
            tags={tags}
            handleTagClick={onTagClick}
          />
        </div>
        <div className="flex flex-wrap gap-1.5 mt-2">
          {viewingCard.tags.map((tag) => (
            <span
              key={tag.name}
              className="inline-flex items-center px-1.5 py-0.5 bg-purple-50 text-purple-600 text-xs rounded-full"
            >
              <span
                className="cursor-pointer hover:bg-purple-100"
                onClick={() => navigate(`/app/search?term=${encodeURIComponent('#' + tag.name)}`)}
              >
                #{tag.name}
              </span>
              {viewingCard.body.includes(`#${tag.name}`) && (
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
        <hr className="my-4" />
      </div>

      {/* Details Section */}
      <div className="text-xs text-gray-600 space-y-1 pt-4 border-t">
        {viewingCard.link && (
          <div className="flex items-start">
            <span className="font-medium w-20">Link:</span>
            <div
              className="flex-1 break-all"
              dangerouslySetInnerHTML={{ __html: linkifyWithDefaultOptions(viewingCard.link) }}
            />
          </div>
        )}
        <div className="flex items-start">
          <span className="font-medium w-20">Created:</span>
          <span className="flex-1">{viewingCard.created_at.toISOString()}</span>
        </div>
        <div className="flex items-start">
          <span className="font-medium w-20">Updated:</span>
          <span className="flex-1">{viewingCard.updated_at.toISOString()}</span>
        </div>
      </div>
    </div>
  );
}