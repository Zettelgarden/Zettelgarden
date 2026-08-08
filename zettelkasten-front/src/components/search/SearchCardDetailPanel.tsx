import React, { useState, useEffect } from 'react';
import { Card, SearchResult, Entity } from '../../models/Card';
import { getCard, saveExistingCard } from '../../api/cards';
import { CardBody } from '../cards/CardBody';
import { HeaderSubSection, HeaderSection } from '../Header';
import { CardTag } from '../cards/CardTag';
import { formatDate } from '../../utils/dates';

// Type guard to check if a selected card is a SearchResult
function isSearchResult(
  card: SearchResult | Card | null,
): card is SearchResult {
  return card !== null && 'type' in card && 'preview' in card;
}

// Type guard to check if a selected card is a full Card
function isFullCard(card: SearchResult | Card | null): card is Card {
  return card !== null && 'body' in card;
}

interface SearchCardDetailPanelProps {
  // The selected card (can be SearchResult or full Card)
  selectedCard: SearchResult | Card | null;

  // Event handlers
  onEdit?: (cardId: number) => void;
  onTagClick?: (tagName: string) => void;
  onEntityClick?: (entityName: string) => void;

  // Mobile back navigation
  onBack?: () => void;

  // Mobile visibility control
  isMobileDetailVisible?: boolean;
}

export function SearchCardDetailPanel({
  selectedCard,
  onEdit,
  onTagClick,
  onEntityClick,
  onBack,
  isMobileDetailVisible = false,
}: SearchCardDetailPanelProps) {
  const [fullCard, setFullCard] = useState<Card | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Fetch full card data when a SearchResult is selected
  useEffect(() => {
    if (!selectedCard) {
      setFullCard(null);
      setError(null);
      return;
    }

    // If already a full Card, use it directly
    if (isFullCard(selectedCard)) {
      setFullCard(selectedCard);
      return;
    }

    // If it's a SearchResult of type card, fetch the full card
    if (isSearchResult(selectedCard) && selectedCard.type === 'card') {
      const cardId = selectedCard.metadata?.id;
      if (cardId) {
        fetchCardData(cardId);
      }
    }
  }, [selectedCard]);

  const fetchCardData = async (cardId: number) => {
    setIsLoading(true);
    setError(null);
    try {
      const card = await getCard(cardId.toString());
      setFullCard(card);
    } catch (err) {
      console.error('Failed to fetch card details:', err);
      setError('Failed to load card details');
    } finally {
      setIsLoading(false);
    }
  };

  const handleEditClick = () => {
    if (fullCard && onEdit) {
      onEdit(fullCard.id);
    }
  };

  const handleTagClick = (tagName: string) => {
    if (onTagClick) {
      onTagClick(tagName);
    }
  };

  const handleEntityClick = async (entityName: string) => {
    if (onEntityClick) {
      onEntityClick(entityName);
    }
  };

  const handleSaveCard = async (updatedCard: Card) => {
    try {
      const saved = await saveExistingCard(updatedCard);
      setFullCard(saved);
    } catch (err) {
      console.error('Failed to save card:', err);
      setError('Failed to save card');
    }
  };

  // Empty state when no card is selected
  if (!selectedCard) {
    return (
      <div className="flex flex-col items-center justify-center h-full bg-white p-8 text-center">
        <svg
          className="w-16 h-16 text-gray-300 mb-4"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
          />
        </svg>
        <p className="text-gray-600 text-lg font-medium">No card selected</p>
        <p className="text-gray-500 text-sm mt-2">
          Select a card from the search results to view its details
        </p>
      </div>
    );
  }

  // Loading state
  if (isLoading) {
    return (
      <div className="flex flex-col items-center justify-center h-full bg-white p-8">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mb-4"></div>
        <p className="text-gray-600">Loading card details...</p>
      </div>
    );
  }

  // Error state
  if (error && !fullCard) {
    return (
      <div className="flex flex-col items-center justify-center h-full bg-white p-8 text-center">
        <svg
          className="w-16 h-16 text-red-300 mb-4"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
          />
        </svg>
        <p className="text-red-600 text-lg font-medium">{error}</p>
      </div>
    );
  }

  // Display selected search result (non-card types)
  if (
    !fullCard &&
    isSearchResult(selectedCard) &&
    selectedCard.type !== 'card'
  ) {
    return (
      <div className="h-full bg-white overflow-y-auto">
        {/* Mobile header with back button */}
        {onBack && (
          <div className="md:hidden sticky top-0 z-10 bg-white border-b border-gray-200 px-4 py-3">
            <button
              onClick={onBack}
              className="flex items-center gap-2 text-blue-600 hover:text-blue-800"
            >
              <svg
                className="w-5 h-5"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M15 19l-7-7 7-7"
                />
              </svg>
              Back to results
            </button>
          </div>
        )}

        <div className="p-6">
          <div className="mb-4">
            <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-purple-100 text-purple-800">
              {selectedCard.type === 'entity'
                ? 'Entity'
                : selectedCard.type === 'fact'
                ? 'Fact'
                : selectedCard.type}
            </span>
          </div>

          <h2 className="text-xl font-bold text-gray-900 mb-4">
            {selectedCard.title}
          </h2>

          {selectedCard.preview && (
            <div className="prose prose-sm max-w-none text-gray-700 mb-6">
              {selectedCard.preview}
            </div>
          )}

          {/* Metadata */}
          <div className="bg-gray-50 rounded-lg p-4 text-sm text-gray-600 space-y-2">
            <div>
              <span className="font-medium">Created:</span>{' '}
              {formatDate(selectedCard.created_at.toISOString())}
            </div>
            <div>
              <span className="font-medium">Updated:</span>{' '}
              {formatDate(selectedCard.updated_at.toISOString())}
            </div>
            {selectedCard.score !== undefined && (
              <div>
                <span className="font-medium">Relevance Score:</span>{' '}
                {selectedCard.score.toFixed(2)}
              </div>
            )}
          </div>

          {/* Tags */}
          {selectedCard.tags && selectedCard.tags.length > 0 && (
            <div className="mt-6">
              <HeaderSubSection text="Tags" />
              <div className="flex flex-wrap gap-2 mt-2">
                {selectedCard.tags.map((tag) => (
                  <button
                    key={tag.id}
                    onClick={() => handleTagClick(tag.name)}
                    className="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium bg-purple-50 text-purple-700 hover:bg-purple-100 transition-colors"
                  >
                    #{tag.name}
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    );
  }

  // Display full card details
  if (!fullCard) {
    return null;
  }

  return (
    <div className="h-full bg-white overflow-y-auto">
      {/* Mobile header with back button */}
      {onBack && (
        <div className="md:hidden sticky top-0 z-10 bg-white border-b border-gray-200 px-4 py-3">
          <button
            onClick={onBack}
            className="flex items-center gap-2 text-blue-600 hover:text-blue-800"
          >
            <svg
              className="w-5 h-5"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M15 19l-7-7 7-7"
              />
            </svg>
            Back to results
          </button>
        </div>
      )}

      <div className="p-6 space-y-6">
        {/* Card Header */}
        <div className="space-y-4">
          {/* Card ID and actions */}
          <div className="flex items-start justify-between">
            <div className="flex items-center gap-2">
              <span className="text-blue-600 font-mono text-sm">
                [{fullCard.card_id}]
              </span>
              {fullCard.is_starred && (
                <svg
                  className="w-5 h-5 text-yellow-500 fill-current"
                  viewBox="0 0 20 20"
                >
                  <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z" />
                </svg>
              )}
            </div>

            {/* Action buttons */}
            <div className="flex items-center gap-2">
              {onEdit && (
                <button
                  onClick={handleEditClick}
                  className="inline-flex items-center gap-1 px-3 py-1.5 text-sm font-medium text-blue-600 hover:text-blue-800 hover:bg-blue-50 rounded-md transition-colors"
                >
                  <svg
                    className="w-4 h-4"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
                    />
                  </svg>
                  Edit
                </button>
              )}
            </div>
          </div>

          {/* Title */}
          <h1 className="text-2xl font-bold text-gray-900">
            {fullCard.title || 'Untitled Card'}
          </h1>

          {/* Metadata */}
          <div className="flex flex-wrap items-center gap-4 text-sm text-gray-500">
            <span>Created {formatDate(fullCard.created_at.toISOString())}</span>
            <span>Updated {formatDate(fullCard.updated_at.toISOString())}</span>
          </div>
        </div>

        {/* Tags */}
        {fullCard.tags && fullCard.tags.length > 0 && (
          <div>
            <HeaderSubSection text="Tags" />
            <div className="flex flex-wrap gap-2 mt-2">
              {fullCard.tags.map((tag) => (
                <button
                  key={tag.id}
                  onClick={() => handleTagClick(tag.name)}
                  className="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium bg-purple-50 text-purple-700 hover:bg-purple-100 transition-colors"
                >
                  #{tag.name}
                </button>
              ))}
            </div>
          </div>
        )}

        {/* Card Body (Content) */}
        <div className="bg-white rounded-lg border border-gray-200 shadow-sm">
          <div className="p-4">
            <CardBody
              viewingCard={fullCard}
              entities={fullCard.entities}
              onSave={handleSaveCard}
            />
          </div>
        </div>

        {/* Entities Section */}
        {fullCard.entities && fullCard.entities.length > 0 && (
          <div>
            <HeaderSubSection text="Entities" />
            <div className="flex flex-wrap gap-2 mt-2">
              {fullCard.entities.map((entity) => (
                <button
                  key={entity.id}
                  onClick={() => handleEntityClick(`@[${entity.name}]`)}
                  className="inline-flex items-center gap-1 px-2 py-1 rounded-md text-xs font-medium bg-yellow-50 text-yellow-700 hover:bg-yellow-100 transition-colors"
                >
                  <svg
                    className="w-3 h-3"
                    fill="currentColor"
                    viewBox="0 0 20 20"
                  >
                    <path
                      fillRule="evenodd"
                      d="M10 9a3 3 0 100-6 3 3 0 000 6zm-7 9a7 7 0 1114 0H3z"
                      clipRule="evenodd"
                    />
                  </svg>
                  {entity.name}
                </button>
              ))}
            </div>
          </div>
        )}

        {/* Parent Card */}
        {fullCard.parent && fullCard.parent.id > 0 && (
          <div>
            <HeaderSubSection text="Parent Card" />
            <div className="mt-2">
              <CardTag card={fullCard.parent} showTitle={true} />
            </div>
          </div>
        )}

        {/* Children Cards */}
        {fullCard.children && fullCard.children.length > 0 && (
          <div>
            <HeaderSubSection text={`Children (${fullCard.children.length})`} />
            <div className="mt-2 space-y-2">
              {fullCard.children.map((child) => (
                <div key={child.id} className="pl-4 border-l-2 border-gray-200">
                  <CardTag card={child} showTitle={true} />
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Backlinks/References */}
        {fullCard.references && fullCard.references.length > 0 && (
          <div>
            <HeaderSubSection
              text={`References (${fullCard.references.length})`}
            />
            <div className="mt-2 space-y-2">
              {fullCard.references.map((ref) => (
                <div key={ref.id} className="pl-4 border-l-2 border-gray-200">
                  <CardTag card={ref} showTitle={true} />
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Linked Files */}
        {fullCard.files && fullCard.files.length > 0 && (
          <div>
            <HeaderSubSection text={`Files (${fullCard.files.length})`} />
            <div className="mt-2 space-y-2">
              {fullCard.files.map((file) => (
                <div
                  key={file.id}
                  className="flex items-center gap-2 text-sm text-gray-700"
                >
                  <svg
                    className="w-4 h-4 text-gray-400"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z"
                    />
                  </svg>
                  <span>{file.name}</span>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Tasks */}
        {fullCard.tasks && fullCard.tasks.length > 0 && (
          <div>
            <HeaderSubSection text={`Tasks (${fullCard.tasks.length})`} />
            <div className="mt-2 space-y-2">
              {fullCard.tasks.map((task) => (
                <div
                  key={task.id}
                  className={`flex items-start gap-2 text-sm p-2 rounded ${
                    task.is_complete
                      ? 'bg-gray-50 text-gray-500'
                      : 'bg-white border border-gray-200'
                  }`}
                >
                  <svg
                    className={`w-4 h-4 mt-0.5 flex-shrink-0 ${
                      task.is_complete ? 'text-green-500' : 'text-gray-300'
                    }`}
                    fill="currentColor"
                    viewBox="0 0 20 20"
                  >
                    <path
                      fillRule="evenodd"
                      d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
                      clipRule="evenodd"
                    />
                  </svg>
                  <span className={task.is_complete ? 'line-through' : ''}>
                    {task.title}
                  </span>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
