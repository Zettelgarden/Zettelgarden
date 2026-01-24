import React, { useState, useEffect } from "react";
import { SchemaDefinition, FieldDefinition } from "../../models/Schema";
import { fetchSchemas } from "../../api/schemas";
import { StructuredDataEditor } from "./StructuredDataEditor";

interface CardSchemaSectionProps {
  schemaId: number | null | undefined;
  structuredData: Record<string, any> | null | undefined;
  onSchemaChange: (schemaId: number | null) => void;
  onDataChange: (data: Record<string, any>) => void;
  disabled?: boolean;
}

export function CardSchemaSection({
  schemaId,
  structuredData,
  onSchemaChange,
  onDataChange,
  disabled = false,
}: CardSchemaSectionProps) {
  const [schemas, setSchemas] = useState<SchemaDefinition[]>([]);
  const [selectedSchema, setSelectedSchema] = useState<SchemaDefinition | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Load schemas when component mounts
  useEffect(() => {
    setLoading(true);
    fetchSchemas()
      .then((data) => {
        setSchemas(data);
        setLoading(false);
      })
      .catch((err) => {
        console.error("Error fetching schemas:", err);
        setError("Failed to load schemas");
        setLoading(false);
      });
  }, []);

  // Update selected schema when schemaId prop changes (only after schemas are loaded)
  useEffect(() => {
    if (loading || schemas.length === 0) return;

    if (schemaId) {
      const schema = schemas.find((s) => s.id === schemaId);
      setSelectedSchema(schema || null);
    } else {
      setSelectedSchema(null);
    }
  }, [schemaId, schemas, loading]);

  const handleSchemaChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const value = e.target.value;
    const newSchemaId = value ? parseInt(value) : null;

    if (newSchemaId) {
      const schema = schemas.find((s) => s.id === newSchemaId);
      setSelectedSchema(schema || null);
    } else {
      setSelectedSchema(null);
    }

    onSchemaChange(newSchemaId);

    // Clear structured data when schema changes
    if (newSchemaId !== schemaId) {
      onDataChange({});
    }
  };

  return (
    <div className="space-y-4">
      <div>
        <label htmlFor="schema-select" className="block text-sm font-medium text-gray-700 mb-1">
          Schema
        </label>
        <select
          id="schema-select"
          value={schemaId || ""}
          onChange={handleSchemaChange}
          disabled={disabled || loading}
          className="w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm disabled:bg-gray-100 disabled:cursor-not-allowed"
        >
          <option value="">No schema</option>
          {schemas.map((schema) => (
            <option key={schema.id} value={schema.id}>
              {schema.name}
            </option>
          ))}
        </select>
        {loading && <p className="text-xs text-gray-500 mt-1">Loading schemas...</p>}
        {error && <p className="text-xs text-red-600 mt-1">{error}</p>}
      </div>

      {selectedSchema && (
        <StructuredDataEditor
          fields={selectedSchema.fields}
          data={structuredData || {}}
          onChange={onDataChange}
          disabled={disabled}
        />
      )}
    </div>
  );
}
