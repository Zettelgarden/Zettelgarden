import React, { useState, useEffect } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { SchemaDefinition, FieldDefinition } from "../../models/Schema";
import { fetchSchema } from "../../api/schemas";
import { Card } from "../../models/Card";
import { setDocumentTitle } from "../../utils/title";

interface SchemaTablePageProps {
  schemaId: number;
  onBack: () => void;
}

export function SchemaTablePage({ schemaId, onBack }: SchemaTablePageProps) {
  const navigate = useNavigate();
  const [schema, setSchema] = useState<SchemaDefinition | null>(null);
  const [cards, setCards] = useState<Card[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [sortField, setSortField] = useState<string | null>(null);
  const [sortDirection, setSortDirection] = useState<"asc" | "desc">("asc");

  useEffect(() => {
    setDocumentTitle("Table View");
    loadData();
  }, [schemaId]);

  const loadData = async () => {
    setLoading(true);
    setError(null);

    try {
      // Fetch schema
      const schemaData = await fetchSchema(schemaId);
      setSchema(schemaData);

      // Fetch all cards and filter by schema_id
      const token = localStorage.getItem("token");
      const response = await fetch(`${import.meta.env.VITE_URL}/cards/unsorted`, {
        headers: { Authorization: `Bearer ${token}` },
      });

      if (!response.ok) {
        throw new Error("Failed to fetch cards");
      }

      const allCards = await response.json();
      const filteredCards = allCards
        .filter((card: Card) => card.schema_id === schemaId)
        .map((card: Card) => ({
          ...card,
          created_at: card.created_at instanceof Date ? card.created_at : new Date(card.created_at),
          updated_at: card.updated_at instanceof Date ? card.updated_at : new Date(card.updated_at),
        }));

      setCards(filteredCards);
    } catch (err) {
      console.error("Error loading data:", err);
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
        return new Date(value).toLocaleDateString();
      default:
        return String(value);
    }
  };

  const handleCardClick = (card: Card) => {
    navigate(`/app/card/${card.id}`);
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

  return (
    <div className="p-4">
      <div className="flex items-center justify-between mb-4">
        <div>
          <button onClick={onBack} className="text-blue-600 hover:underline text-sm mb-2">
            ← Back to Schemas
          </button>
          <h1 className="text-2xl font-bold text-gray-900">{schema.name} - Table View</h1>
          <p className="text-sm text-gray-500">{cards.length} cards</p>
        </div>
      </div>

      {cards.length === 0 ? (
        <div className="text-center text-gray-500 py-8 bg-gray-50 rounded-lg">
          <p>No cards with this schema yet.</p>
          <button
            onClick={() => navigate("/app/card/new")}
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
                  onClick={() => handleSort("title")}
                >
                  <div className="flex items-center gap-1">
                    Title
                    {sortField === "title" && (
                      <span>{sortDirection === "asc" ? "↑" : "↓"}</span>
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
                        <span>{sortDirection === "asc" ? "↑" : "↓"}</span>
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
                <tr
                  key={card.id}
                  className="hover:bg-gray-50 cursor-pointer"
                  onClick={() => handleCardClick(card)}
                >
                  <td className="px-4 py-3 text-sm font-medium text-blue-600">
                    {card.title}
                  </td>
                  {schema.fields.map((field) => (
                    <td key={field.name} className="px-4 py-3 text-sm text-gray-900">
                      {getFieldValue(card, field)}
                    </td>
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
