import React, { useState, useEffect } from "react";
import { SchemaDefinition, FieldDefinition } from "../../models/Schema";
import { Card } from "../../models/Card";
import { fetchSchema } from "../../api/schemas";

interface SchemaTableProps {
  schemaId: number;
  onCardClick?: (card: Card) => void;
  compact?: boolean;
  columns?: string[]; // List of column names to display
}

export function SchemaTable({ schemaId, onCardClick, compact = false, columns }: SchemaTableProps) {
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
    if (!sortField) return cards;

    return [...cards].sort((a, b) => {
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

  return (
    <div className={compact ? "my-2" : "my-4"}>
      <div className="flex items-center justify-between mb-2">
        <div>
          <h3 className={compact ? "text-base font-semibold text-gray-900" : "text-xl font-bold text-gray-900"}>
            {schema.name}
          </h3>
          <p className="text-xs text-gray-500">{cards.length} cards</p>
        </div>
      </div>

      {cards.length === 0 ? (
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
