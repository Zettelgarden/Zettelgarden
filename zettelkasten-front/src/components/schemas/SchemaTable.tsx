import React, { useState, useEffect, useMemo, useCallback } from 'react';
import { useSearchParams } from 'react-router-dom';
import { SchemaDefinition, FieldDefinition } from '../../models/Schema';
import { Card } from '../../models/Card';
import { fetchSchemaByRef } from '../../api/schemas';
import { NotFoundError, isAPIError } from '../../api/errors';
import { applyFilterGroupsToCard } from '../../utils/schemaFilters';
import { CardLink } from '../cards/CardLink';
import { getCard } from '../../api/cards';

// LinkedCardDisplay component for link_to_card fields
interface LinkedCardDisplayProps {
  cardId: number;
}

function LinkedCardDisplay({ cardId }: LinkedCardDisplayProps) {
  const [linkedCard, setLinkedCard] = useState<Card | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    getCard(cardId.toString())
      .then((result) => {
        if (isError(result)) {
          setLinkedCard(null);
        } else {
          setLinkedCard(result);
        }
      })
      .catch(() => {
        setLinkedCard(null);
      })
      .finally(() => {
        setLoading(false);
      });
  }, [cardId]);

  function isError(result: any): result is { error: string } {
    return result && typeof result === 'object' && 'error' in result;
  }

  if (loading) {
    return <span className="text-sm text-gray-400">Loading...</span>;
  }

  if (!linkedCard) {
    return <span className="text-blue-600 text-sm font-mono">{cardId}</span>;
  }

  return (
    <CardLink
      card={linkedCard}
      showTitle={true}
      handleViewBacklink={() => {}}
    />
  );
}

interface SchemaTableProps {
  schemaRef: string; // Can be an ID (numeric string) or slug
  onCardClick?: (card: Card) => void;
  compact?: boolean;
  columns?: string[]; // List of column names to display
  filters?: Record<string, string> | Array<Record<string, string>>; // AND map, or OR-of-AND groups from "a=1,b=2||c=3"
}

export function SchemaTable({
  schemaRef,
  onCardClick,
  compact = false,
  columns,
  filters,
}: SchemaTableProps) {
  const [searchParams, setSearchParams] = useSearchParams();
  const [schema, setSchema] = useState<SchemaDefinition | null>(null);
  const [cards, setCards] = useState<Card[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [sortField, setSortField] = useState<string | null>(null);
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc');

  // Pagination state
  const [currentPage, setCurrentPage] = useState(() => {
    const pageParam = searchParams.get(`schema_${schemaRef}_page`);
    return pageParam ? parseInt(pageParam, 10) : 1;
  });

  // Responsive items per page
  const [itemsPerPage, setItemsPerPage] = useState(() => {
    const isMobile = window.innerWidth < 768;
    return isMobile ? 5 : 10;
  });

  useEffect(() => {
    const handleResize = () => {
      const isMobile = window.innerWidth < 768;
      setItemsPerPage(isMobile ? 5 : 10);
    };

    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

  useEffect(() => {
    loadData();
  }, [schemaRef]);

  // Filter fields based on columns prop
  const getFilteredFields = (fields: FieldDefinition[]): FieldDefinition[] => {
    if (!columns || columns.length === 0) {
      return fields;
    }
    return fields.filter((field) => columns.includes(field.name));
  };

  // Apply filters to cards (AND within a group, OR across groups)
  const filteredCards = useMemo(() => {
    if (!filters) {
      return cards;
    }

    const groups = Array.isArray(filters) ? filters : [filters];
    if (groups.length === 0) {
      return cards;
    }

    return cards.filter((card) => applyFilterGroupsToCard(card, groups));
  }, [cards, filters]);

  // Calculate total pages for pagination
  const totalPages = useMemo(() => {
    return Math.ceil(filteredCards.length / itemsPerPage);
  }, [filteredCards.length, itemsPerPage]);

  const loadData = async () => {
    setLoading(true);
    setError(null);

    try {
      // Fetch schema by ref (ID or slug)
      const schemaData = await fetchSchemaByRef(schemaRef);
      setSchema(schemaData);

      // Fetch cards with this schema_id using the actual ID from the schema
      const token = localStorage.getItem('token');
      const response = await fetch(
        `${import.meta.env.VITE_URL}/schemas/${schemaData.id}/cards`,
        {
          headers: { Authorization: `Bearer ${token}` },
        },
      );

      if (!response.ok) {
        throw new Error('Failed to fetch cards');
      }

      const fetchedCards = await response.json();
      const cardsWithDates = fetchedCards.map((card: Card) => ({
        ...card,
        created_at:
          card.created_at instanceof Date
            ? card.created_at
            : new Date(card.created_at),
        updated_at:
          card.updated_at instanceof Date
            ? card.updated_at
            : new Date(card.updated_at),
      }));

      setCards(cardsWithDates);
    } catch (err) {
      console.error('Error loading schema table:', err);
      if (err instanceof NotFoundError) {
        setError(`Schema '${schemaRef}' not found`);
      } else if (isAPIError(err)) {
        setError(err.message || 'Failed to load data');
      } else {
        setError('Failed to load data');
      }
    } finally {
      setLoading(false);
    }
  };

  const handleSort = (fieldName: string) => {
    if (sortField === fieldName) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
    } else {
      setSortField(fieldName);
      setSortDirection('asc');
    }
  };

  const getSortedCards = () => {
    if (!sortField) return filteredCards;

    return [...filteredCards].sort((a, b) => {
      // Handle title field separately
      const aValue =
        sortField === 'title' ? a.title : a.structured_data?.[sortField];
      const bValue =
        sortField === 'title' ? b.title : b.structured_data?.[sortField];

      if (aValue === undefined) return 1;
      if (bValue === undefined) return -1;
      if (aValue === bValue) return 0;

      const comparison = aValue < bValue ? -1 : 1;
      return sortDirection === 'asc' ? comparison : -comparison;
    });
  };

  const handlePageChange = useCallback(
    (newPage: number) => {
      const clampedPage = Math.max(1, Math.min(newPage, totalPages));
      setCurrentPage(clampedPage);

      // Update URL state
      const newParams = new URLSearchParams(searchParams);
      if (clampedPage === 1) {
        newParams.delete(`schema_${schemaRef}_page`);
      } else {
        newParams.set(`schema_${schemaRef}_page`, clampedPage.toString());
      }
      setSearchParams(newParams);
    },
    [totalPages, searchParams, schemaRef, setSearchParams],
  );

  // Reset to page 1 if filters change and current page is out of bounds
  useEffect(() => {
    if (currentPage > totalPages && totalPages > 0) {
      setCurrentPage(1);
    }
  }, [currentPage, totalPages]);

  // Handle keyboard navigation
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'ArrowLeft' && currentPage > 1) {
        handlePageChange(currentPage - 1);
      } else if (e.key === 'ArrowRight' && currentPage < totalPages) {
        handlePageChange(currentPage + 1);
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [currentPage, totalPages, handlePageChange]);

  const getFieldValue = (card: Card, field: FieldDefinition) => {
    const value = card.structured_data?.[field.name];

    if (value === null || value === undefined || value === '') {
      return <span className="text-gray-400 italic">—</span>;
    }

    switch (field.type) {
      case 'boolean':
        return value ? 'Yes' : 'No';
      case 'multi-select':
        return (value as string[]).join(', ');
      case 'date':
        // Parse date and display in UTC to avoid timezone shifting
        const dateObj = new Date(value);
        return dateObj.toLocaleDateString(undefined, { timeZone: 'UTC' });
      case 'link_to_card':
        return <LinkedCardDisplay cardId={value} />;
      default:
        return String(value);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center p-4 bg-gray-50 rounded-lg">
        <div className="text-sm text-gray-500">Loading schema table...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-4 bg-red-50 rounded-lg">
        <div className="text-sm text-red-600">{error}</div>
      </div>
    );
  }

  if (!schema) {
    return (
      <div className="p-4 bg-yellow-50 rounded-lg">
        <div className="text-sm text-yellow-600">Schema not found</div>
      </div>
    );
  }

  const sortedCards = getSortedCards();
  const filteredFields = schema ? getFilteredFields(schema.fields) : [];
  const hasFilters = filters
    ? Array.isArray(filters)
      ? filters.length > 0
      : Object.keys(filters).length > 0
    : false;
  const totalCards = cards.length;
  const displayCards = sortedCards.length;

  // Paginate cards
  const paginatedCards = sortedCards.slice(
    (currentPage - 1) * itemsPerPage,
    currentPage * itemsPerPage,
  );

  return (
    <div className={compact ? 'my-2' : 'my-4'}>
      <div className="flex items-center justify-between mb-2">
        <div>
          <h3
            className={
              compact
                ? 'text-base font-semibold text-gray-900'
                : 'text-xl font-bold text-gray-900'
            }
          >
            {schema.name}
          </h3>
          <p className="text-xs text-gray-500">
            {hasFilters
              ? `${displayCards} of ${totalCards} cards`
              : `${totalCards} cards`}
          </p>
        </div>
      </div>

      {totalCards === 0 ? (
        <div className="text-center text-gray-500 py-4 bg-gray-50 rounded-lg">
          <p className="text-sm">No cards with this schema yet.</p>
        </div>
      ) : displayCards === 0 ? (
        <div className="text-center text-gray-500 py-4 bg-gray-50 rounded-lg">
          <p className="text-sm">No cards match the current filters.</p>
        </div>
      ) : (
        <div className="overflow-x-auto border rounded-lg">
          <table className="min-w-full bg-white">
            <thead className="bg-gray-50 border-b">
              <tr>
                <th
                  className="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase cursor-pointer hover:bg-gray-100"
                  onClick={() => handleSort('title')}
                >
                  <div className="flex items-center gap-1">
                    Title
                    {sortField === 'title' && (
                      <span>{sortDirection === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </th>
                {filteredFields.map((field) => (
                  <th
                    key={field.name}
                    className="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase cursor-pointer hover:bg-gray-100"
                    onClick={() => handleSort(field.name)}
                  >
                    <div className="flex items-center gap-1">
                      {field.name}
                      {sortField === field.name && (
                        <span>{sortDirection === 'asc' ? '↑' : '↓'}</span>
                      )}
                    </div>
                  </th>
                ))}
                <th className="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase">
                  Updated
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {paginatedCards.map((card) => (
                <tr
                  key={card.id}
                  className={
                    onCardClick ? 'hover:bg-gray-50 cursor-pointer' : ''
                  }
                  onClick={() => onCardClick?.(card)}
                >
                  <td className="px-3 py-2 text-sm font-medium text-blue-600">
                    {card.title}
                  </td>
                  {filteredFields.map((field) => (
                    <td
                      key={field.name}
                      className="px-3 py-2 text-sm text-gray-900"
                    >
                      {getFieldValue(card, field)}
                    </td>
                  ))}
                  <td className="px-3 py-2 text-sm text-gray-500">
                    {card.updated_at instanceof Date
                      ? card.updated_at.toLocaleDateString()
                      : new Date(card.updated_at).toLocaleDateString()}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Pagination Controls */}
      {totalPages > 1 && (
        <div className="mt-4 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <button
              onClick={() => handlePageChange(currentPage - 1)}
              disabled={currentPage === 1}
              className="min-w-[44px] min-h-[44px] px-3 py-2 text-sm font-medium rounded-md border disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
              aria-label="Previous page"
            >
              Previous
            </button>

            {/* Page indicator - desktop shows range, mobile shows current */}
            <span className="hidden md:inline text-sm text-gray-700 px-2">
              Page {currentPage} of {totalPages}
            </span>
            <span className="md:hidden text-sm text-gray-700 px-2">
              {currentPage} / {totalPages}
            </span>

            <button
              onClick={() => handlePageChange(currentPage + 1)}
              disabled={currentPage === totalPages}
              className="min-w-[44px] min-h-[44px] px-3 py-2 text-sm font-medium rounded-md border disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
              aria-label="Next page"
            >
              Next
            </button>
          </div>

          {/* Mobile-optimized page jump buttons */}
          <div className="hidden sm:flex items-center gap-1">
            {totalPages <= 7 ? (
              // Show all page numbers if 7 or fewer pages
              Array.from({ length: totalPages }, (_, i) => i + 1).map(
                (pageNum) => (
                  <button
                    key={pageNum}
                    onClick={() => handlePageChange(pageNum)}
                    className={`min-w-[44px] min-h-[44px] px-3 py-2 text-sm font-medium rounded-md border focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 ${
                      pageNum === currentPage
                        ? 'bg-blue-600 text-white border-blue-600'
                        : 'bg-white text-gray-700 border-gray-300 hover:bg-gray-50'
                    }`}
                    aria-label={`Go to page ${pageNum}`}
                    aria-current={pageNum === currentPage ? 'page' : undefined}
                  >
                    {pageNum}
                  </button>
                ),
              )
            ) : (
              // Show abbreviated page numbers for more than 7 pages
              <>
                {currentPage > 3 && (
                  <>
                    <button
                      onClick={() => handlePageChange(1)}
                      className="min-w-[44px] min-h-[44px] px-3 py-2 text-sm font-medium rounded-md border bg-white text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
                    >
                      1
                    </button>
                    {currentPage > 4 && (
                      <span className="px-2 text-gray-500">...</span>
                    )}
                  </>
                )}

                {Array.from(
                  { length: Math.min(3, totalPages) },
                  (_, i) => Math.max(1, currentPage - 1) + i,
                )
                  .filter((pageNum) => pageNum >= 1 && pageNum <= totalPages)
                  .map((pageNum) => (
                    <button
                      key={pageNum}
                      onClick={() => handlePageChange(pageNum)}
                      className={`min-w-[44px] min-h-[44px] px-3 py-2 text-sm font-medium rounded-md border focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 ${
                        pageNum === currentPage
                          ? 'bg-blue-600 text-white border-blue-600'
                          : 'bg-white text-gray-700 border-gray-300 hover:bg-gray-50'
                      }`}
                      aria-label={`Go to page ${pageNum}`}
                      aria-current={
                        pageNum === currentPage ? 'page' : undefined
                      }
                    >
                      {pageNum}
                    </button>
                  ))}

                {currentPage < totalPages - 2 && (
                  <>
                    {currentPage < totalPages - 3 && (
                      <span className="px-2 text-gray-500">...</span>
                    )}
                    <button
                      onClick={() => handlePageChange(totalPages)}
                      className="min-w-[44px] min-h-[44px] px-3 py-2 text-sm font-medium rounded-md border bg-white text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
                    >
                      {totalPages}
                    </button>
                  </>
                )}
              </>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
