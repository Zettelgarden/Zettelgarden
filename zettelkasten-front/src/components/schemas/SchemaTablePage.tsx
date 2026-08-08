import React, { useState, useEffect } from 'react';
import { useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { SchemaDefinition, FieldDefinition } from '../../models/Schema';
import { fetchSchema } from '../../api/schemas';
import { Card } from '../../models/Card';
import { setDocumentTitle } from '../../utils/title';
import { CardLink } from '../cards/CardLink';
import { getCard } from '../../api/cards';
import {
  FilterInput,
  ActiveFilterDisplay,
  FilterValue,
  FiltersState,
} from './SchemaTableFilters';
import { EditableCell } from './EditableCell';
import { schemaCardsToCsv, downloadCsv } from '../../utils/schemaCsv';

interface LinkedCardDisplayProps {
  cardId: number;
}

function LinkedCardDisplay({ cardId }: LinkedCardDisplayProps) {
  const [card, setCard] = useState<Card | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    getCard(cardId.toString())
      .then((result) => {
        if (isError(result)) {
          setCard(null);
        } else {
          setCard(result);
        }
      })
      .catch(() => {
        setCard(null);
      })
      .finally(() => {
        setLoading(false);
      });
  }, [cardId]);

  function isError(result: any): result is { error: string } {
    return result && typeof result === 'object' && 'error' in result;
  }

  if (loading) {
    return <span className="text-sm text-gray-500">Loading...</span>;
  }

  if (!card) {
    return (
      <span className="text-blue-600 hover:underline text-sm font-mono">
        {cardId}
      </span>
    );
  }

  return (
    <CardLink card={card} showTitle={true} handleViewBacklink={() => {}} />
  );
}

interface SchemaTablePageProps {
  schemaId: number;
  onBack: () => void;
}

export function SchemaTablePage({ schemaId, onBack }: SchemaTablePageProps) {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const [schema, setSchema] = useState<SchemaDefinition | null>(null);
  const [cards, setCards] = useState<Card[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [sortField, setSortField] = useState<string | null>(null);
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc');
  const [filters, setFilters] = useState<FiltersState>({});
  const [showFilters, setShowFilters] = useState(false);

  // Sync filters with URL query params
  useEffect(() => {
    const parsedFilters: FiltersState = {};

    for (const [key, value] of searchParams.entries()) {
      if (key === 'sort') {
        setSortField(value);
        continue;
      }
      if (key === 'order') {
        setSortDirection(value === 'desc' ? 'desc' : 'asc');
        continue;
      }
      if (key.startsWith('filter_')) {
        const fieldName = key.replace('filter_', '');
        try {
          const filterData = JSON.parse(decodeURIComponent(value));
          parsedFilters[fieldName] = filterData;
        } catch {
          // Skip invalid filter values
        }
      }
    }

    setFilters(parsedFilters);
  }, [searchParams]);

  // Update URL when filters change
  const updateFilters = (newFilters: FiltersState) => {
    setFilters(newFilters);

    const params = new URLSearchParams();

    // Add sort params
    if (sortField) params.set('sort', sortField);
    if (sortDirection !== 'asc') params.set('order', sortDirection);

    // Add filter params
    Object.entries(newFilters).forEach(([fieldName, filterValue]) => {
      params.set(
        `filter_${fieldName}`,
        encodeURIComponent(JSON.stringify(filterValue)),
      );
    });

    setSearchParams(params, { replace: true });
  };

  const clearFilter = (fieldName: string) => {
    const newFilters = { ...filters };
    delete newFilters[fieldName];
    updateFilters(newFilters);
  };

  const clearAllFilters = () => {
    updateFilters({});
  };

  useEffect(() => {
    setDocumentTitle('Table View');
    loadData();
  }, [schemaId]);

  const loadData = async () => {
    setLoading(true);
    setError(null);

    try {
      // Fetch schema
      const schemaData = await fetchSchema(schemaId);
      setSchema(schemaData);

      // Fetch cards with this schema_id
      const token = localStorage.getItem('token');
      const response = await fetch(
        `${import.meta.env.VITE_URL}/schemas/${schemaId}/cards`,
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
      console.error('Error loading data:', err);
      setError('Failed to load data');
    } finally {
      setLoading(false);
    }
  };

  const handleSort = (fieldName: string) => {
    let newDirection: 'asc' | 'desc' = 'asc';
    if (sortField === fieldName) {
      newDirection = sortDirection === 'asc' ? 'desc' : 'asc';
    }
    setSortField(fieldName);
    setSortDirection(newDirection);

    // Update URL
    const params = new URLSearchParams(searchParams);
    params.set('sort', fieldName);
    params.set('order', newDirection);
    setSearchParams(params, { replace: true });
  };

  const getSortedCards = () => {
    let filteredCards = [...cards];

    // Apply filters
    Object.entries(filters).forEach(([fieldName, filterValue]) => {
      filteredCards = filteredCards.filter((card) => {
        const cardValue = card.structured_data?.[fieldName];

        if (cardValue === null || cardValue === undefined || cardValue === '') {
          return false;
        }

        switch (filterValue.type) {
          case 'text':
            switch (filterValue.operator) {
              case 'contains':
                return String(cardValue)
                  .toLowerCase()
                  .includes(String(filterValue.value).toLowerCase());
              case 'equals':
                return (
                  String(cardValue).toLowerCase() ===
                  String(filterValue.value).toLowerCase()
                );
              case 'startsWith':
                return String(cardValue)
                  .toLowerCase()
                  .startsWith(String(filterValue.value).toLowerCase());
              default:
                return true;
            }

          case 'number':
            const cardNum = parseFloat(cardValue);
            const filterNum = parseFloat(filterValue.value);
            if (isNaN(cardNum) || isNaN(filterNum)) return false;
            switch (filterValue.operator) {
              case 'equals':
                return cardNum === filterNum;
              case 'gt':
                return cardNum > filterNum;
              case 'gte':
                return cardNum >= filterNum;
              case 'lt':
                return cardNum < filterNum;
              case 'lte':
                return cardNum <= filterNum;
              default:
                return true;
            }

          case 'date':
            const cardDate = new Date(cardValue);
            const filterDate = new Date(filterValue.value);
            if (isNaN(cardDate.getTime()) || isNaN(filterDate.getTime()))
              return false;
            switch (filterValue.operator) {
              case 'equals':
                return cardDate.toDateString() === filterDate.toDateString();
              case 'before':
                return cardDate < filterDate;
              case 'after':
                return cardDate > filterDate;
              default:
                return true;
            }

          case 'boolean':
            return Boolean(cardValue) === Boolean(filterValue.value);

          case 'select':
            return String(cardValue) === String(filterValue.value);

          case 'multi-select':
            if (
              filterValue.operator === 'any' &&
              Array.isArray(filterValue.value)
            ) {
              const cardValues = Array.isArray(cardValue)
                ? cardValue
                : [cardValue];
              return filterValue.value.some((v: string) =>
                cardValues.includes(v),
              );
            }
            return false;

          case 'link_to_card':
            return parseInt(cardValue, 10) === parseInt(filterValue.value, 10);

          default:
            return true;
        }
      });
    });

    // Apply sorting
    if (!sortField) return filteredCards;

    return filteredCards.sort((a, b) => {
      const aValue = a.structured_data?.[sortField];
      const bValue = b.structured_data?.[sortField];

      if (aValue === undefined) return 1;
      if (bValue === undefined) return -1;
      if (aValue === bValue) return 0;

      const comparison = aValue < bValue ? -1 : 1;
      return sortDirection === 'asc' ? comparison : -comparison;
    });
  };

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
        return new Date(value).toLocaleDateString();
      case 'link_to_card':
        return <LinkedCardDisplay cardId={value} />;
      default:
        return String(value);
    }
  };

  const handleCardClick = (card: Card) => {
    navigate(`/app/card/${card.id}`);
  };

  const refreshCards = async () => {
    try {
      const token = localStorage.getItem('token');
      const response = await fetch(
        `${import.meta.env.VITE_URL}/schemas/${schemaId}/cards`,
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
      console.error('Error refreshing cards:', err);
    }
  };

  if (loading) {
    return (
      <div className="p-4 flex items-center justify-center">
        <div className="text-gray-500">Loading...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-4">
        <div className="text-red-600 mb-4">{error}</div>
        <button onClick={onBack} className="text-blue-600 hover:underline">
          ← Back to Schemas
        </button>
      </div>
    );
  }

  if (!schema) {
    return (
      <div className="p-4">
        <div className="text-red-600 mb-4">Schema not found</div>
        <button onClick={onBack} className="text-blue-600 hover:underline">
          ← Back to Schemas
        </button>
      </div>
    );
  }

  const sortedCards = getSortedCards();

  const handleAddCard = () => {
    navigate(`/app/card/new?schema=${schemaId}`);
  };

  const handleExportCsv = () => {
    const csv = schemaCardsToCsv(sortedCards, schema.fields);
    downloadCsv(`${schema.slug || 'schema'}-cards.csv`, csv);
  };

  return (
    <div className="p-4">
      <div className="flex items-center justify-between mb-4">
        <div>
          <button
            onClick={onBack}
            className="text-blue-600 hover:underline text-sm mb-2"
          >
            ← Back to Schemas
          </button>
          <h1 className="text-2xl font-bold text-gray-900">
            {schema.name} - Table View
          </h1>
          <p className="text-sm text-gray-500">{cards.length} cards</p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={handleAddCard}
            className="px-4 py-3 min-h-[44px] bg-blue-500 text-white text-sm rounded-lg hover:bg-blue-600 flex items-center gap-1"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              className="h-4 w-4"
              viewBox="0 0 20 20"
              fill="currentColor"
            >
              <path
                fillRule="evenodd"
                d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z"
                clipRule="evenodd"
              />
            </svg>
            Add Card
          </button>
          <button
            onClick={handleExportCsv}
            className="px-4 py-3 min-h-[44px] bg-white border border-gray-300 text-gray-700 text-sm rounded-lg hover:bg-gray-50 flex items-center gap-1"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              className="h-4 w-4"
              viewBox="0 0 20 20"
              fill="currentColor"
            >
              <path d="M10 2a1 1 0 011 1v7.586l2.293-2.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 111.414-1.414L9 10.586V3a1 1 0 011-1z" />
              <path
                fillRule="evenodd"
                d="M4 14a1 1 0 011 1v1a1 1 0 001 1h8a1 1 0 001-1v-1a1 1 0 112 0v1a3 3 0 01-3 3H6a3 3 0 01-3-3v-1a1 1 0 011-1z"
                clipRule="evenodd"
              />
            </svg>
            Export CSV
          </button>
          <button
            onClick={() => setShowFilters(!showFilters)}
            className="px-4 py-3 min-h-[44px] bg-blue-500 text-white text-sm rounded-lg hover:bg-blue-600 flex items-center gap-1"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              className="h-4 w-4"
              viewBox="0 0 20 20"
              fill="currentColor"
            >
              <path
                fillRule="evenodd"
                d="M3 3a1 1 0 011-1h12a1 1 0 011 1v3a1 1 0 01-.293.707L12 11.414V15a1 1 0 01-.293.707l-2 2A1 1 0 018 17v-5.586L3.293 6.707A1 1 0 013 6V3z"
                clipRule="evenodd"
              />
            </svg>
            Filters{' '}
            {Object.keys(filters).length > 0 &&
              `(${Object.keys(filters).length})`}
          </button>
        </div>
      </div>

      {/* Filter Section */}
      {showFilters && (
        <div className="mb-4 p-4 bg-gray-50 border rounded-lg">
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-semibold text-gray-700">
              Filter by field values
            </h3>
            {Object.keys(filters).length > 0 && (
              <button
                onClick={clearAllFilters}
                className="text-xs text-blue-600 hover:text-blue-800 hover:underline"
              >
                Clear all filters
              </button>
            )}
          </div>

          {/* Filter Inputs */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3 mb-3">
            {schema.fields.map((field) => (
              <div key={field.name} className="space-y-1">
                <label className="block text-xs font-medium text-gray-600">
                  {field.name}
                </label>
                <FilterInput
                  field={field}
                  value={filters[field.name] || null}
                  onChange={(newValue) => {
                    const newFilters = { ...filters };
                    if (newValue) {
                      newFilters[field.name] = newValue;
                    } else {
                      delete newFilters[field.name];
                    }
                    updateFilters(newFilters);
                  }}
                />
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Active Filters Display */}
      {Object.keys(filters).length > 0 && (
        <div className="mb-3 flex flex-wrap items-center gap-2">
          <span className="text-xs font-medium text-gray-500">
            Active filters:
          </span>
          {Object.entries(filters).map(([fieldName, filterValue]) => (
            <ActiveFilterDisplay
              key={fieldName}
              fieldName={fieldName}
              value={filterValue}
              onClear={() => clearFilter(fieldName)}
            />
          ))}
        </div>
      )}

      {cards.length === 0 ? (
        <div className="text-center text-gray-500 py-8 bg-gray-50 rounded-lg">
          <p>No cards with this schema yet.</p>
          <button
            onClick={() => navigate(`/app/card/new?schema=${schema.id}`)}
            className="mt-4 px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600"
          >
            Create First Card
          </button>
        </div>
      ) : (
        <div className="overflow-x-auto border rounded-lg">
          <table className="min-w-full bg-white">
            <thead className="bg-gray-50 border-b">
              <tr>
                <th
                  className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase cursor-pointer hover:bg-gray-100"
                  onClick={() => handleSort('title')}
                >
                  <div className="flex items-center gap-1">
                    Title
                    {sortField === 'title' && (
                      <span>{sortDirection === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </th>
                {schema.fields.map((field) => (
                  <th
                    key={field.name}
                    className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase cursor-pointer hover:bg-gray-100"
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
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  Updated
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {sortedCards.map((card) => (
                <tr key={card.id} className="hover:bg-gray-50">
                  <td className="px-4 py-3 text-sm">
                    <CardLink
                      card={card}
                      showTitle={true}
                      handleViewBacklink={() => {}}
                    />
                  </td>
                  {schema.fields.map((field) => (
                    <EditableCell
                      key={`${card.id}-${field.name}`}
                      card={card}
                      field={field}
                      onSave={refreshCards}
                    />
                  ))}
                  <td className="px-4 py-3 text-sm text-gray-500">
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
    </div>
  );
}
