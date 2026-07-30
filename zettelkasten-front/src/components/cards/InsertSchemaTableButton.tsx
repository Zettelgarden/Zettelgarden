import React, { useState, useEffect } from "react";
import { Popover } from "@headlessui/react";
import { SchemaDefinition, FieldDefinition } from "../../models/Schema";
import { fetchSchemas } from "../../api/schemas";

interface InsertSchemaTableButtonProps {
  onInsert: (syntax: string) => void;
}

type FilterOperator = "eq" | "ne" | "gt" | "gte" | "lt" | "lte";

const NUMBER_OPERATORS: { value: FilterOperator; label: string }[] = [
  { value: "eq", label: "=" },
  { value: "ne", label: "≠" },
  { value: "gt", label: ">" },
  { value: "gte", label: "≥" },
  { value: "lt", label: "<" },
  { value: "lte", label: "≤" },
];

const DATE_OPERATORS: { value: FilterOperator; label: string }[] = [
  { value: "eq", label: "On" },
  { value: "ne", label: "Not on" },
  { value: "gt", label: "After" },
  { value: "gte", label: "On or after" },
  { value: "lt", label: "Before" },
  { value: "lte", label: "On or before" },
];

export function InsertSchemaTableButton({ onInsert }: InsertSchemaTableButtonProps) {
  const [schemas, setSchemas] = useState<SchemaDefinition[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedSchema, setSelectedSchema] = useState<SchemaDefinition | null>(null);
  const [selectedColumns, setSelectedColumns] = useState<Set<string>>(new Set());
  const [filters, setFilters] = useState<Array<{ field: string; operator: FilterOperator; value: string }>>([]);

  // Fetch schemas on mount
  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    fetchSchemas()
      .then(result => {
        if (!cancelled) {
          setSchemas(result.filter(s => !s.is_deleted));
        }
      })
      .catch(err => {
        console.error("Failed to load schemas:", err);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => { cancelled = true; };
  }, []);

  const handleSelectSchema = (schema: SchemaDefinition) => {
    setSelectedSchema(schema);
    setSelectedColumns(new Set());
    setFilters([]);
  };

  const toggleColumn = (fieldName: string) => {
    setSelectedColumns(prev => {
      const next = new Set(prev);
      if (next.has(fieldName)) {
        next.delete(fieldName);
      } else {
        next.add(fieldName);
      }
      return next;
    });
  };

  const addFilter = () => {
    if (!selectedSchema) return;
    const firstField = selectedSchema.fields[0]?.name || "";
    setFilters(prev => [...prev, { field: firstField, operator: "eq", value: "" }]);
  };

  // Changing the field resets operator/value because the available operators
  // and input type depend on the field type.
  const changeFilterField = (index: number, fieldName: string) => {
    setFilters(prev =>
      prev.map((f, i) =>
        i === index ? { field: fieldName, operator: "eq" as FilterOperator, value: "" } : f
      )
    );
  };

  const removeFilter = (index: number) => {
    setFilters(prev => prev.filter((_, i) => i !== index));
  };

  const updateFilter = (index: number, key: "field" | "value" | "operator", val: string) => {
    setFilters(prev =>
      prev.map((f, i) => (i === index ? { ...f, [key]: val } : f))
    );
  };

  const buildSyntax = (): string => {
    if (!selectedSchema) return "";

    // Use slug if available, otherwise fall back to ID
    const ref = selectedSchema.slug || selectedSchema.id.toString();
    let syntax = `{{schema:${ref}}}`;

    const parts: string[] = [];

    if (selectedColumns.size > 0) {
      parts.push(`columns:${Array.from(selectedColumns).join(",")}`);
    }

    // Build filter string, skipping empty values. Operators other than "eq"
    // are emitted as "<op>:<value>" so the schema filter engine can apply
    // gt/gte/lt/lte comparisons (e.g. due=gt:2026-01-01).
    const validFilters = filters.filter(f => f.field && f.value.trim());
    if (validFilters.length > 0) {
      const filterStr = validFilters
        .map(f => {
          const v = f.value.trim();
          return f.operator && f.operator !== "eq"
            ? `${f.field}=${f.operator}:${v}`
            : `${f.field}=${v}`;
        })
        .join(",");
      parts.push(`filter:${filterStr}`);
    }

    if (parts.length > 0) {
      syntax = `{{schema:${ref}|${parts.join("|")}}}`;
    }

    return syntax;
  };

  const handleInsert = () => {
    const syntax = buildSyntax();
    if (syntax) {
      onInsert("\n" + syntax + "\n");
    }
  };

  const getFilterField = (fieldName: string): FieldDefinition | undefined => {
    if (!selectedSchema) return undefined;
    return selectedSchema.fields.find(f => f.name === fieldName);
  };

  const reset = () => {
    setSelectedSchema(null);
    setSelectedColumns(new Set());
    setFilters([]);
  };

  return (
    <Popover className="relative inline-block">
      <Popover.Button
        className="inline-flex items-center gap-1 px-3 py-1.5 text-xs font-medium text-gray-600 bg-white border border-gray-300 rounded hover:bg-gray-50 focus:outline-none focus:ring-1 focus:ring-offset-1 focus:ring-blue-500 min-h-[32px]"
        title="Insert schema table"
      >
        <svg xmlns="http://www.w3.org/2000/svg" className="h-3.5 w-3.5" viewBox="0 0 20 20" fill="currentColor">
          <path fillRule="evenodd" d="M5 4a1 1 0 00-2 0v7a1 1 0 001 1h7a1 1 0 001-1V4a1 1 0 00-1-1H5zM3 14a1 1 0 011-1h7a1 1 0 011 1v2a1 1 0 01-1 1H4a1 1 0 01-1-1v-2zm12-5a1 1 0 00-1 1v6a1 1 0 001 1h2a1 1 0 001-1v-6a1 1 0 00-1-1h-2z" clipRule="evenodd" />
        </svg>
        Schema Table
      </Popover.Button>

      <Popover.Panel className="absolute z-[60] left-0 mt-2 w-96 bg-white rounded-lg shadow-lg border border-gray-200">
        {({ close }) => (
          <div className="p-4">
            <div className="flex items-center justify-between mb-3">
              <h3 className="font-semibold text-gray-900 text-sm">Insert Schema Table</h3>
              {selectedSchema && (
                <button
                  onClick={reset}
                  className="text-xs text-blue-600 hover:text-blue-800"
                >
                  ← Change schema
                </button>
              )}
            </div>

            {/* Schema Picker */}
            {!selectedSchema && (
              <div>
                {loading ? (
                  <div className="text-sm text-gray-500 py-4 text-center">Loading schemas...</div>
                ) : schemas.length === 0 ? (
                  <div className="text-sm text-gray-500 py-4 text-center">No schemas found</div>
                ) : (
                  <ul className="divide-y divide-gray-100 border border-gray-200 rounded-md max-h-60 overflow-auto">
                    {schemas.map(schema => (
                      <li key={schema.id}>
                        <button
                          onClick={() => handleSelectSchema(schema)}
                          className="w-full text-left px-3 py-2 hover:bg-gray-50 text-sm"
                        >
                          <span className="font-medium text-gray-800">{schema.name}</span>
                          <span className="text-gray-400 ml-2 text-xs">({schema.slug})</span>
                        </button>
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            )}

            {/* Configuration */}
            {selectedSchema && (
              <div className="space-y-4">
                {/* Schema name */}
                <div className="text-sm text-gray-600">
                  Schema: <span className="font-medium text-gray-900">{selectedSchema.name}</span>
                </div>

                {/* Column Selection */}
                {selectedSchema.fields.length > 0 && (
                  <div>
                    <div className="text-xs font-medium text-gray-700 mb-1.5">Columns (leave empty for all)</div>
                    <div className="flex flex-wrap gap-2">
                      {selectedSchema.fields.map(field => (
                        <label key={field.name} className="inline-flex items-center gap-1 text-xs">
                          <input
                            type="checkbox"
                            checked={selectedColumns.has(field.name)}
                            onChange={() => toggleColumn(field.name)}
                            className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                          />
                          <span className="text-gray-700">{field.name}</span>
                        </label>
                      ))}
                    </div>
                  </div>
                )}

                {/* Filters */}
                <div>
                  <div className="flex items-center justify-between mb-1.5">
                    <span className="text-xs font-medium text-gray-700">Filters</span>
                    <button
                      onClick={addFilter}
                      className="text-xs text-blue-600 hover:text-blue-800"
                    >
                      + Add filter
                    </button>
                  </div>
                  {filters.length === 0 && (
                    <div className="text-xs text-gray-400">No filters</div>
                  )}
                  {filters.map((filter, index) => {
                    const field = getFilterField(filter.field);
                    const fieldType = field?.type;
                    const operators =
                      fieldType === "date"
                        ? DATE_OPERATORS
                        : fieldType === "number"
                          ? NUMBER_OPERATORS
                          : null;
                    const options =
                      field && (field.type === "select" || field.type === "multi-select")
                        ? field.options
                        : undefined;
                    return (
                      <div key={index} className="flex items-center gap-1.5 mb-2">
                        <select
                          value={filter.field}
                          onChange={(e) => changeFilterField(index, e.target.value)}
                          className="flex-1 min-w-0 text-xs border border-gray-300 rounded px-2 py-1.5 bg-white"
                        >
                          {selectedSchema.fields.map(f => (
                            <option key={f.name} value={f.name}>{f.name}</option>
                          ))}
                        </select>
                        {operators ? (
                          <select
                            value={filter.operator}
                            onChange={(e) => updateFilter(index, "operator", e.target.value)}
                            className="text-xs border border-gray-300 rounded px-1.5 py-1.5 bg-white"
                          >
                            {operators.map(op => (
                              <option key={op.value} value={op.value}>{op.label}</option>
                            ))}
                          </select>
                        ) : (
                          <span className="text-gray-400 text-xs">=</span>
                        )}
                        {fieldType === "date" ? (
                          <input
                            type="date"
                            value={filter.value}
                            onChange={(e) => updateFilter(index, "value", e.target.value)}
                            className="flex-1 min-w-0 text-xs border border-gray-300 rounded px-2 py-1.5"
                          />
                        ) : fieldType === "number" ? (
                          <input
                            type="number"
                            value={filter.value}
                            onChange={(e) => updateFilter(index, "value", e.target.value)}
                            placeholder="value"
                            className="flex-1 min-w-0 text-xs border border-gray-300 rounded px-2 py-1.5"
                          />
                        ) : options ? (
                          <select
                            value={filter.value}
                            onChange={(e) => updateFilter(index, "value", e.target.value)}
                            className="flex-1 min-w-0 text-xs border border-gray-300 rounded px-2 py-1.5 bg-white"
                          >
                            <option value="">Select...</option>
                            {options.map(opt => (
                              <option key={opt} value={opt}>{opt}</option>
                            ))}
                          </select>
                        ) : (
                          <input
                            type="text"
                            value={filter.value}
                            onChange={(e) => updateFilter(index, "value", e.target.value)}
                            placeholder="value"
                            className="flex-1 min-w-0 text-xs border border-gray-300 rounded px-2 py-1.5"
                          />
                        )}
                        <button
                          onClick={() => removeFilter(index)}
                          className="text-red-400 hover:text-red-600 text-xs p-1"
                          title="Remove filter"
                        >
                          ✕
                        </button>
                      </div>
                    );
                  })}
                </div>

                {/* Preview */}
                <div>
                  <div className="text-xs font-medium text-gray-700 mb-1">Preview</div>
                  <div className="bg-gray-50 rounded px-2 py-1.5 font-mono text-xs text-blue-700 break-all">
                    {buildSyntax()}
                  </div>
                </div>

                {/* Insert Button */}
                <div className="flex justify-end gap-2 pt-1">
                  <button
                    onClick={() => { reset(); close(); }}
                    className="px-3 py-1.5 text-xs font-medium text-gray-700 bg-white border border-gray-300 rounded hover:bg-gray-50"
                  >
                    Cancel
                  </button>
                  <button
                    onClick={() => {
                      handleInsert();
                      reset();
                      close();
                    }}
                    className="px-3 py-1.5 text-xs font-medium text-white bg-blue-600 rounded hover:bg-blue-700"
                  >
                    Insert
                  </button>
                </div>
              </div>
            )}
          </div>
        )}
      </Popover.Panel>
    </Popover>
  );
}
