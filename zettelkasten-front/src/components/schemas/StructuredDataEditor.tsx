import React, { useState, useEffect } from "react";
import { FieldDefinition } from "../../models/Schema";
import { BacklinkInputDropdownList } from "../cards/BacklinkInputDropdownList";
import { PartialCard, Card } from "../../models/Card";
import { Link } from "react-router-dom";
import { getCard } from "../../api/cards";
import { CardTag } from "../cards/CardTag";

interface LinkedCardFieldProps {
  value: number | null;
  onUpdate: (cardId: number | null) => void;
}

function LinkedCardField({ value, onUpdate }: LinkedCardFieldProps) {
  const [card, setCard] = useState<Card | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!value) {
      setCard(null);
      setError(null);
      return;
    }

    setLoading(true);
    setError(null);

    getCard(value.toString())
      .then((result) => {
        if (isError(result)) {
          setError("Failed to load card");
          setCard(null);
        } else {
          setCard(result);
        }
      })
      .catch(() => {
        setError("Failed to load card");
        setCard(null);
      })
      .finally(() => {
        setLoading(false);
      });
  }, [value]);

  // Type guard for error response
  function isError(result: any): result is { error: string } {
    return result && typeof result === "object" && "error" in result;
  }

  return (
    <div className="space-y-2">
      <BacklinkInputDropdownList
        onSelect={(selectedCard: PartialCard) => onUpdate(selectedCard.id)}
        onSearch={() => {}}
        placeholder="Search for a card..."
        className="w-full"
      />
      {loading && (
        <div className="text-sm text-gray-500">Loading card...</div>
      )}
      {error && (
        <div className="text-sm text-red-600">{error}</div>
      )}
      {card && (
        <div className="flex items-center justify-between p-2 bg-blue-50 border border-blue-200 rounded-lg">
          <Link
            to={`/app/card/${card.id}`}
            className="flex-1 min-w-0"
          >
            <CardTag card={card} showTitle={true} />
          </Link>
          <button
            type="button"
            onClick={() => onUpdate(null)}
            className="text-blue-600 hover:text-blue-800 hover:bg-blue-100 p-1 rounded flex-shrink-0 ml-2"
            title="Remove link"
          >
            <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
              <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
            </svg>
          </button>
        </div>
      )}
    </div>
  );
}

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
          <LinkedCardField
            value={value}
            onUpdate={(cardId) => updateField(field.name, cardId)}
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
