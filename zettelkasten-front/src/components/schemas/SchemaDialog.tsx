import React, { useState, useEffect, useMemo } from "react";
import { Dialog } from "@headlessui/react";
import { SchemaDefinition, FieldDefinition } from "../../models/Schema";
import { createSchema, updateSchema, CreateSchemaParams, UpdateSchemaParams } from "../../api/schemas";

interface SchemaDialogProps {
  schema: SchemaDefinition | null;
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

const FIELD_TYPES = [
  { value: "text", label: "Text" },
  { value: "number", label: "Number" },
  { value: "date", label: "Date" },
  { value: "boolean", label: "Boolean" },
  { value: "select", label: "Select" },
  { value: "multi-select", label: "Multi-Select" },
  { value: "link_to_card", label: "Link to Card" },
] as const;

type FieldType = (typeof FIELD_TYPES)[number]["value"];

interface FieldEditor extends Omit<FieldDefinition, "type"> {
  type: FieldType;
  id: string;
}

// Generate a URL-safe slug from a schema name
function generateSlug(name: string): string {
  const slug = name
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9\s-]/g, '') // Remove special characters
    .replace(/\s+/g, '-') // Replace spaces with hyphens
    .replace(/-+/g, '-'); // Replace multiple hyphens with single hyphen
  return slug || "schema";
}

export function SchemaDialog({ schema, isOpen, onClose, onSuccess }: SchemaDialogProps) {
  const [name, setName] = useState("");
  const [fields, setFields] = useState<FieldEditor[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Generate slug preview from name
  const slugPreview = useMemo(() => generateSlug(name), [name]);

  useEffect(() => {
    if (isOpen) {
      if (schema) {
        setName(schema.name);
        setFields(
          schema.fields.map((f, idx) => ({
            ...f,
            type: f.type as FieldType,
            id: `${idx}`,
          }))
        );
      } else {
        setName("");
        setFields([]);
      }
      setError(null);
    }
  }, [isOpen, schema]);

  const addField = () => {
    setFields([
      ...fields,
      {
        id: Date.now().toString(),
        name: "",
        type: "text",
        required: false,
        options: [],
      },
    ]);
  };

  const removeField = (id: string) => {
    setFields(fields.filter((f) => f.id !== id));
  };

  const updateField = (id: string, updates: Partial<FieldEditor>) => {
    setFields(
      fields.map((f) =>
        f.id === id ? { ...f, ...updates } : f
      )
    );
  };

  const validateFields = (): string | null => {
    if (!name.trim()) {
      return "Schema name is required";
    }

    if (fields.length === 0) {
      return "At least one field is required";
    }

    for (const field of fields) {
      if (!field.name.trim()) {
        return "All fields must have a name";
      }

      const duplicateName = fields.filter(
        (f, i) => f.id !== field.id && f.name.trim() === field.name.trim()
      );
      if (duplicateName.length > 0) {
        return `Field name "${field.name}" is duplicated`;
      }

      if ((field.type === "select" || field.type === "multi-select") && (!field.options || field.options.length === 0)) {
        return `Field "${field.name}" of type "${field.type}" must have at least one option`;
      }
    }

    return null;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    const validationError = validateFields();
    if (validationError) {
      setError(validationError);
      return;
    }

    setError(null);
    setIsSubmitting(true);

    try {
      const schemaFields: FieldDefinition[] = fields.map((f) => ({
        name: f.name.trim(),
        type: f.type,
        required: f.required,
        options: f.options || [],
      }));

      if (schema) {
        const params: UpdateSchemaParams = {
          name: name.trim(),
          fields: schemaFields,
        };
        await updateSchema(schema.id, params);
      } else {
        const params: CreateSchemaParams = {
          name: name.trim(),
          fields: schemaFields,
        };
        await createSchema(params);
      }

      onSuccess();
      onClose();
    } catch (err) {
      console.error("SchemaDialog: error", err);
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError("An unexpected error occurred");
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  const renderFieldEditor = (field: FieldEditor, index: number) => {
    const needsOptions = field.type === "select" || field.type === "multi-select";

    return (
      <div key={field.id} className="border border-gray-200 rounded-lg p-3 space-y-2">
        <div className="grid grid-cols-[1fr_auto_1fr] gap-2 items-start">
          <div>
            <label className="block text-xs font-medium text-gray-700 mb-0.5">
              Name *
            </label>
            <input
              type="text"
              value={field.name}
              onChange={(e) => updateField(field.id, { name: e.target.value })}
              className="w-full rounded border border-gray-300 px-2 py-1.5 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:ring-opacity-30 transition-colors"
              placeholder="field_name"
            />
          </div>

          <button
            type="button"
            onClick={() => removeField(field.id)}
            className="text-red-600 hover:text-red-800 p-1 hover:bg-red-50 rounded-full transition-colors mt-5"
            disabled={fields.length === 1}
          >
            <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
              <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
            </svg>
          </button>

          <div>
            <label className="block text-xs font-medium text-gray-700 mb-0.5">
              Type *
            </label>
            <select
              value={field.type}
              onChange={(e) => updateField(field.id, { type: e.target.value as FieldType })}
              className="w-full rounded border border-gray-300 px-2 py-1.5 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:ring-opacity-30 transition-colors"
            >
              {FIELD_TYPES.map((type) => (
                <option key={type.value} value={type.value}>
                  {type.label}
                </option>
              ))}
            </select>
          </div>
        </div>

        <div className="flex items-center gap-4">
          {needsOptions && (
            <div className="flex-1">
              <label className="block text-xs font-medium text-gray-700 mb-0.5">
                Options *
              </label>
              <input
                type="text"
                value={field.options?.join(", ") || ""}
                onChange={(e) =>
                  updateField(field.id, {
                    options: e.target.value
                      .split(",")
                      .map((s) => s.trim())
                      .filter(Boolean),
                  })
                }
                className="w-full rounded border border-gray-300 px-2 py-1.5 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:ring-opacity-30 transition-colors"
                placeholder="Option 1, Option 2"
              />
            </div>
          )}

          <label className="flex items-center gap-1.5 text-sm text-gray-700 cursor-pointer">
            <input
              type="checkbox"
              checked={field.required}
              onChange={(e) => updateField(field.id, { required: e.target.checked })}
              className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            Required
          </label>
        </div>
      </div>
    );
  };

  return (
    <Dialog open={isOpen} onClose={onClose} className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="fixed inset-0 bg-black bg-opacity-30" aria-hidden="true" />

      <Dialog.Panel className="bg-white rounded-xl shadow-xl w-full max-w-2xl mx-auto relative max-h-[90vh] overflow-hidden flex flex-col">
        <div className="px-6 py-4 border-b border-gray-200 flex-shrink-0">
          <Dialog.Title className="text-xl font-semibold text-gray-900">
            {schema ? "Edit Schema" : "Create Schema"}
          </Dialog.Title>
        </div>

        <form onSubmit={handleSubmit} className="flex-1 overflow-y-auto p-6">
          <div className="space-y-6">
            <div>
              <label htmlFor="schema-name" className="block text-sm font-medium text-gray-700 mb-1">
                Schema Name *
              </label>
              <input
                type="text"
                id="schema-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="w-full rounded-lg border border-gray-300 px-4 py-2 focus:border-blue-500 focus:ring-2 focus:ring-blue-500 focus:ring-opacity-30 transition-colors"
                placeholder="e.g., Book, Movie, Recipe"
                required
              />
              {slugPreview && (
                <div className="mt-2 flex items-center gap-2">
                  <span className="text-xs text-gray-500">Slug (auto-generated):</span>
                  <code className="text-xs bg-gray-100 px-2 py-1 rounded text-gray-700">
                    {schema?.slug || slugPreview}
                  </code>
                  <span className="text-xs text-gray-400">
                    Use as: <span className="font-mono bg-gray-50 px-1 rounded">{'{{schema:' + (schema?.slug || slugPreview) + '}}'}</span>
                  </span>
                </div>
              )}
            </div>

            <div>
              <div className="flex items-center justify-between mb-3">
                <label className="block text-sm font-medium text-gray-700">
                  Fields *
                </label>
                <button
                  type="button"
                  onClick={addField}
                  className="text-blue-600 hover:text-blue-800 flex items-center gap-1 text-sm font-medium"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
                    <path fillRule="evenodd" d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z" clipRule="evenodd" />
                  </svg>
                  Add Field
                </button>
              </div>

              {fields.length === 0 ? (
                <div className="text-center py-8 bg-gray-50 rounded-lg border-2 border-dashed border-gray-300">
                  <p className="text-gray-500 text-sm">No fields yet. Click "Add Field" to create your first field.</p>
                </div>
              ) : (
                <div className="space-y-3">
                  {fields.map((field, index) => renderFieldEditor(field, index))}
                </div>
              )}
            </div>

            {error && (
              <div className="text-red-600 text-sm bg-red-50 p-3 rounded-lg">
                {error}
              </div>
            )}
          </div>
        </form>

        <div className="px-6 py-4 border-t border-gray-200 flex justify-end gap-4 flex-shrink-0">
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-3 min-h-[44px] text-gray-700 hover:text-gray-900 transition-colors"
          >
            Cancel
          </button>
          <button
            type="button"
            disabled={isSubmitting || !name.trim() || fields.length === 0}
            className="px-6 py-3 min-h-[44px] bg-blue-500 text-white rounded-lg hover:bg-blue-600 disabled:opacity-50 disabled:cursor-not-allowed transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-opacity-50"
            onClick={handleSubmit}
          >
            {isSubmitting ? "Saving..." : schema ? "Save Changes" : "Create Schema"}
          </button>
        </div>
      </Dialog.Panel>
    </Dialog>
  );
}
