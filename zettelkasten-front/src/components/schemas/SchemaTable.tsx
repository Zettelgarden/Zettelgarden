import React, { useState, useEffect, useMemo } from "react";
import { SchemaDefinition, FieldDefinition } from "../../models/Schema";
import { Card } from "../../models/Card";
import { fetchSchema } from "../../api/schemas";
import { FilterValue, FiltersState } from "./SchemaTableFilters";

interface SchemaTableProps {
  schemaId: number;
  onCardClick?: (card: Card) => void;
  compact?: boolean;
  columns?: string[]; // List of column names to display
  filters?: string; // Filter string like "status=In Progress,priority>High"
}

export function SchemaTable({ schemaId, onCardClick, compact = false, columns, filters }: SchemaTableProps) {
  const [schema, setSchema] = useState<SchemaDefinition | null>(null);
  const [cards, setCards] = useState<Card[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [sortField, setSortField] = useState<string | null>(null);
  const [sortDirection, setSortDirection] = useState<"asc" | "desc">("asc");

  useEffect(() => {
    loadData();
  }, [schemaId]);

  // Filter fields based on columns prop
  const getFilteredFields = (fields: FieldDefinition[]): FieldDefinition[] => {
    if (!columns || columns.length === 0) {
      return fields;
    }
    return fields.filter(field => columns.includes(field.name));
  };

  // Parse filter string into FiltersState
  const parsedFilters = useMemo((): FiltersState => {
    if (!filters || !schema) return {};

    const result: FiltersState = {};
    // Split by comma (but be careful with values that might contain commas)
    // For simplicity, we'll split by comma and assume values don't contain commas
    const filterParts = filters.split(',').map(f => f.trim());

    for (const part of filterParts) {
      // Match operator patterns: field=value, field~value, field>value, field<value, etc.
      const match = part.match(/^([^=~^><]+)([=~^><]=?|>=|<=)(.+)$/);
      if (!match) continue;

      const [, fieldName, operator, rawValue] = match;
      const field = schema.fields.find(f => f.name === fieldName);
      if (!field) continue;

      let filterOperator: FilterValue["operator"];
      let value: any = rawValue;

      switch (operator) {
        case "~":
          filterOperator = "contains";
          break;
        case "^":
          filterOperator = "startsWith";
          break;
        case "=":
        case "==":
          filterOperator = "equals";
          break;
        case ">":
          filterOperator = "gt";
          value = field.type === "number" ? parseFloat(rawValue) : rawValue;
          break;
        case ">=":
          filterOperator = "gte";
          value = field.type === "number" ? parseFloat(rawValue) : rawValue;
          break;
        case "<":
          filterOperator = "lt";
          value = field.type === "number" ? parseFloat(rawValue) : rawValue;
          break;
        case "<=":
          filterOperator = "lte";
          value = field.type === "number" ? parseFloat(rawValue) : rawValue;
          break;
        default:
          filterOperator = "contains";
      }

      // Handle type-specific conversions
      if (field.type === "number") {
        value = parseFloat(rawValue);
      } else if (field.type === "boolean") {
        value = rawValue.toLowerCase() === "true" || rawValue === "1";
      } else if (field.type === "multi-select") {
        // Multi-select values can be separated by |
        value = rawValue.split('|').map(v => v.trim());
      } else if (field.type === "link_to_card") {
        value = parseInt(rawValue, 10);
      } else if (field.type === "date") {
        // Keep as string, Date parsing happens in filter logic
        value = rawValue;
      } else {
        // For text/select, use the raw value
        value = rawValue;
      }

      result[fieldName] = {
        type: field.type,
        operator: filterOperator,
        value
      };
    }

    return result;
  }, [filters, schema]);

  const loadData = async () => {
    setLoading(true);
    setError(null);

    try {
      // Fetch schema
      const schemaData = await fetchSchema(schemaId);
      setSchema(schemaData);

      // Fetch cards with this schema_id
      const token = localStorage.getItem("token");
      const response = await fetch(`${import.meta.env.VITE_URL}/schemas/${schemaId}/cards`, {
        headers: { Authorization: `Bearer ${token}` },
      });

      if (!response.ok) {
        throw new Error("Failed to fetch cards");
      }

      const fetchedCards = await response.json();
      const cardsWithDates = fetchedCards.map((card: Card) => ({
        ...card,
        created_at: card.created_at instanceof Date ? card.created_at : new Date(card.created_at),
        updated_at: card.updated_at instanceof Date ? card.updated_at : new Date(card.updated_at),
      }));

      setCards(cardsWithDates);
    } catch (err) {
      console.error("Error loading schema table:", err);
      setError("Failed to load data");
    } finally {
      setLoading(false);
    }
  };

  const handleSort = (fieldName: string) => {
    if (sortField === fieldName) {
      setSortDirection(sortDirection === "asc" ? "desc" : "asc");
    } else {
      setSortField(fieldName);
      setSortDirection("asc");
    }
  };

  const getSortedCards = () => {
    let filteredCards = [...cards];

    // Apply filters
    Object.entries(parsedFilters).forEach(([fieldName, filterValue]) => {
      filteredCards = filteredCards.filter((card) => {
        const cardValue = card.structured_data?.[fieldName];

        if (cardValue === null || cardValue === undefined || cardValue === "") {
          return false;
        }

        switch (filterValue.type) {
          case "text":
            switch (filterValue.operator) {
              case "contains":
                return String(cardValue).toLowerCase().includes(String(filterValue.value).toLowerCase());
              case "equals":
                return String(cardValue).toLowerCase() === String(filterValue.value).toLowerCase();
              case "startsWith":
                return String(cardValue).toLowerCase().startsWith(String(filterValue.value).toLowerCase());
              default:
                return true;
            }

          case "number":
            const cardNum = parseFloat(cardValue);
            const filterNum = parseFloat(filterValue.value);
            if (isNaN(cardNum) || isNaN(filterNum)) return false;
            switch (filterValue.operator) {
              case "equals":
                return cardNum === filterNum;
              case "gt":
                return cardNum > filterNum;
              case "gte":
                return cardNum >= filterNum;
              case "lt":
                return cardNum < filterNum;
              case "lte":
                return cardNum <= filterNum;
              default:
                return true;
            }

          case "date":
            const cardDate = new Date(cardValue);
            const filterDate = new Date(filterValue.value);
            if (isNaN(cardDate.getTime()) || isNaN(filterDate.getTime())) return false;
            switch (filterValue.operator) {
              case "equals":
                return cardDate.toDateString() === filterDate.toDateString();
              case "gt":
              case "after":
                return cardDate > filterDate;
              case "lt":
              case "before":
                return cardDate < filterDate;
              default:
                return true;
            }

          case "boolean":
            return Boolean(cardValue) === Boolean(filterValue.value);

          case "select":
            return String(cardValue) === String(filterValue.value);

          case "multi-select":
            if (filterValue.operator === "any" && Array.isArray(filterValue.value)) {
              const cardValues = Array.isArray(cardValue) ? cardValue : [cardValue];
              return filterValue.value.some((v: string) => cardValues.includes(v));
            }
            // For simple equals with multi-select, check if any value matches
            const cardValues = Array.isArray(cardValue) ? cardValue : [cardValue];
            const filterValues = Array.isArray(filterValue.value) ? filterValue.value : [filterValue.value];
            return filterValues.some((v: string) => cardValues.includes(v));

          case "link_to_card":
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
      return sortDirection === "asc" ? comparison : -comparison;
    });
  };

  const getFieldValue = (card: Card, field: FieldDefinition) => {
    const value = card.structured_data?.[field.name];

    if (value === null || value === undefined || value === "") {
      return <span className="text-gray-400 italic">—</span>;
    }

    switch (field.type) {
      case "boolean":
        return value ? "Yes" : "No";
      case "multi-select":
        return (value as string[]).join(", ");
      case "date":
        // Parse date and display in UTC to avoid timezone shifting
        const dateObj = new Date(value);
        return dateObj.toLocaleDateString(undefined, { timeZone: 'UTC' });
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
  const hasActiveFilters = Object.keys(parsedFilters).length > 0;

  return (
    <div className={compact ? "my-2" : "my-4"}>
      <div className="flex items-center justify-between mb-2">
        <div>
          <h3 className={compact ? "text-base font-semibold text-gray-900" : "text-xl font-bold text-gray-900"}>
            {schema.name}
          </h3>
          <p className="text-xs text-gray-500">
            {hasActiveFilters ? `${sortedCards.length} of ${cards.length} cards` : `${cards.length} cards`}
          </p>
        </div>
      </div>

      {sortedCards.length === 0 ? (
        <div className="text-center text-gray-500 py-4 bg-gray-50 rounded-lg">
          <p className="text-sm">No cards with this schema yet.</p>
        </div>
      ) : (
        <div className="overflow-x-auto border rounded-lg">
          <table className="min-w-full bg-white">
            <thead className="bg-gray-50 border-b">
              <tr>
                <th
                  className="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase cursor-pointer hover:bg-gray-100"
                  onClick={() => handleSort("title")}
                >
                  <div className="flex items-center gap-1">
                    Title
                    {sortField === "title" && (
                      <span>{sortDirection === "asc" ? "↑" : "↓"}</span>
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
                        <span>{sortDirection === "asc" ? "↑" : "↓"}</span>
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
              {sortedCards.map((card) => (
                <tr
                  key={card.id}
                  className={onCardClick ? "hover:bg-gray-50 cursor-pointer" : ""}
                  onClick={() => onCardClick?.(card)}
                >
                  <td className="px-3 py-2 text-sm font-medium text-blue-600">
                    {card.title}
                  </td>
                  {filteredFields.map((field) => (
                    <td key={field.name} className="px-3 py-2 text-sm text-gray-900">
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
    </div>
  );
}
