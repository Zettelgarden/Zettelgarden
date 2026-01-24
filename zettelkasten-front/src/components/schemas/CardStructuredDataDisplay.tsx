import React, { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { SchemaDefinition } from "../../models/Schema";
import { fetchSchema } from "../../api/schemas";
import { StructuredDataDisplay } from "./StructuredDataDisplay";

interface CardStructuredDataDisplayProps {
  schemaId: number | null | undefined;
  structuredData: Record<string, any> | null | undefined;
}

export function CardStructuredDataDisplay({ schemaId, structuredData }: CardStructuredDataDisplayProps) {
  const navigate = useNavigate();
  const [schema, setSchema] = useState<SchemaDefinition | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!schemaId) {
      setSchema(null);
      return;
    }

    setLoading(true);
    setError(null);

    fetchSchema(schemaId)
      .then((data) => {
        setSchema(data);
        setLoading(false);
      })
      .catch((err) => {
        console.error("Error fetching schema:", err);
        setError("Failed to load schema");
        setLoading(false);
      });
  }, [schemaId]);

  if (!schemaId) {
    return null;
  }

  if (loading) {
    return (
      <div className="bg-white rounded-lg p-4 shadow-sm">
        <p className="text-sm text-gray-500">Loading schema...</p>
      </div>
    );
  }

  if (error || !schema) {
    return (
      <div className="bg-white rounded-lg p-4 shadow-sm">
        <p className="text-sm text-red-600">{error || "Schema not found"}</p>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-lg p-4 shadow-sm">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-semibold text-gray-700 border-b pb-2">
          Structured Data
        </h3>
        <button
          onClick={() => navigate(`/app/schemas/${schemaId}/table`)}
          className="text-xs text-blue-600 hover:text-blue-800 hover:underline"
        >
          View all {schema.name} cards →
        </button>
      </div>
      <StructuredDataDisplay fields={schema.fields} data={structuredData || {}} />
    </div>
  );
}
