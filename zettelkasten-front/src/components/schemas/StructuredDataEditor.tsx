import React from "react";
import { FieldDefinition } from "../../models/Schema";

interface StructuredDataEditorProps {
  fields: FieldDefinition[];
  data: Record<string, any>;
  onChange: (data: Record<string, any>) => void;
  disabled?: boolean;
}

export function StructuredDataEditor({ fields, data, onChange, disabled = false }: StructuredDataEditorProps) {
  const updateField = (fieldName: string, value: any) => {
    onChange({
      ...data,
      [fieldName]: value,
    });
  };

  const renderField = (field: FieldDefinition) => {
    const value = data[field.name];

    switch (field.type) {
      case "text":
        return (
          <input
            type="text"
            value={value || ""}
            onChange={(e) => updateField(field.name, e.target.value)}
            disabled={disabled}
            className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-2 focus:ring-blue-500 focus:ring-opacity-30 transition-colors disabled:bg-gray-100 disabled:cursor-not-allowed"
            placeholder={field.required ? `${field.name} (required)` : field.name}
          />
        );

      case "number":
        return (
          <input
            type="number"
            value={value ?? ""}
            onChange={(e) => updateField(field.name, e.target.value ? parseFloat(e.target.value) : null)}
            disabled={disabled}
            className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-2 focus:ring-blue-500 focus:ring-opacity-30 transition-colors disabled:bg-gray-100 disabled:cursor-not-allowed"
            placeholder={field.required ? `${field.name} (required)` : field.name}
          />
        );

      case "date":
        return (
          <input
            type="date"
            value={value || ""}
            onChange={(e) => updateField(field.name, e.target.value || null)}
            disabled={disabled}
            className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-2 focus:ring-blue-500 focus:ring-opacity-30 transition-colors disabled:bg-gray-100 disabled:cursor-not-allowed"
          />
        );

      case "boolean":
        return (
          <div className="flex items-center">
            <input
              type="checkbox"
              id={`field-${field.name}`}
              checked={value || false}
              onChange={(e) => updateField(field.name, e.target.checked)}
              disabled={disabled}
              className="rounded border-gray-300 text-blue-600 focus:ring-blue-500 disabled:opacity-50"
            />
            <label htmlFor={`field-${field.name}`} className="ml-2 text-sm text-gray-700">
              {value ? "Yes" : "No"}
            </label>
          </div>
        );

      case "select":
        return (
          <select
            value={value || ""}
            onChange={(e) => updateField(field.name, e.target.value || null)}
            disabled={disabled}
            className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-2 focus:ring-blue-500 focus:ring-opacity-30 transition-colors disabled:bg-gray-100 disabled:cursor-not-allowed"
          >
            <option value="">Select an option...</option>
            {field.options?.map((option) => (
              <option key={option} value={option}>
                {option}
              </option>
            ))}
          </select>
        );

      case "multi-select":
        return (
          <div className="space-y-2">
            {field.options?.map((option) => (
              <label key={option} className="flex items-center">
                <input
                  type="checkbox"
                  checked={(value || []).includes(option)}
                  onChange={(e) => {
                    const currentValues = value || [];
                    if (e.target.checked) {
                      updateField(field.name, [...currentValues, option]);
                    } else {
                      updateField(field.name, currentValues.filter((v: string) => v !== option));
                    }
                  }}
                  disabled={disabled}
                  className="rounded border-gray-300 text-blue-600 focus:ring-blue-500 disabled:opacity-50"
                />
                <span className="ml-2 text-sm text-gray-700">{option}</span>
              </label>
            ))}
          </div>
        );

      case "link_to_card":
        return (
          <input
            type="text"
            value={value || ""}
            onChange={(e) => updateField(field.name, e.target.value)}
            disabled={disabled}
            className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-2 focus:ring-blue-500 focus:ring-opacity-30 transition-colors disabled:bg-gray-100 disabled:cursor-not-allowed"
            placeholder="Enter card ID or title..."
          />
        );

      default:
        return (
          <input
            type="text"
            value={value || ""}
            onChange={(e) => updateField(field.name, e.target.value)}
            disabled={disabled}
            className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
          />
        );
    }
  };

  if (fields.length === 0) {
    return null;
  }

  return (
    <div className="space-y-4">
      <h3 className="text-sm font-semibold text-gray-700 border-b pb-2">Structured Data</h3>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {fields.map((field) => (
          <div key={field.name} className="space-y-1">
            <label className="block text-sm font-medium text-gray-700">
              {field.name}
              {field.required && <span className="text-red-500 ml-1">*</span>}
            </label>
            {renderField(field)}
          </div>
        ))}
      </div>
    </div>
  );
}
