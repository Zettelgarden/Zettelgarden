import React from "react";
import { FieldDefinition } from "../../models/Schema";

export interface FilterValue {
  type: "text" | "number" | "date" | "boolean" | "select" | "multi-select" | "link_to_card";
  operator?: "contains" | "equals" | "startsWith" | "gt" | "lt" | "gte" | "lte" | "before" | "after" | "any";
  value: any;
}

export interface FiltersState {
  [fieldName: string]: FilterValue;
}

interface FilterInputProps {
  field: FieldDefinition;
  value: FilterValue | null;
  onChange: (value: FilterValue | null) => void;
}

export function FilterInput({ field, value, onChange }: FilterInputProps) {
  const handleChange = (newValue: any, newOperator?: FilterValue["operator"]) => {
    if (newValue === null || newValue === undefined || newValue === "") {
      onChange(null);
      return;
    }
    onChange({
      type: field.type,
      operator: newOperator || getDefaultOperator(field.type),
      value: newValue,
    });
  };

  const getDefaultOperator = (type: FieldDefinition["type"]): FilterValue["operator"] => {
    switch (type) {
      case "text":
      case "link_to_card":
        return "contains";
      case "number":
        return "equals";
      case "date":
        return "equals";
      case "select":
      case "multi-select":
        return "equals";
      case "boolean":
        return "equals";
      default:
        return "contains";
    }
  };

  const renderOperatorSelect = () => {
    const operatorsByType: Record<FieldDefinition["type"], Array<{ value: FilterValue["operator"], label: string }>> = {
      text: [
        { value: "contains", label: "Contains" },
        { value: "equals", label: "Equals" },
        { value: "startsWith", label: "Starts with" },
      ],
      number: [
        { value: "equals", label: "=" },
        { value: "gt", label: ">" },
        { value: "gte", label: "≥" },
        { value: "lt", label: "<" },
        { value: "lte", label: "≤" },
      ],
      date: [
        { value: "equals", label: "On" },
        { value: "before", label: "Before" },
        { value: "after", label: "After" },
      ],
      boolean: [
        { value: "equals", label: "Is" },
      ],
      select: [
        { value: "equals", label: "Is" },
      ],
      "multi-select": [
        { value: "any", label: "Contains any of" },
      ],
      "link_to_card": [
        { value: "equals", label: "Is" },
      ],
    };

    const operators = operatorsByType[field.type] || [];
    if (operators.length <= 1) return null;

    return (
      <select
        value={value?.operator || getDefaultOperator(field.type)}
        onChange={(e) => handleChange(value?.value, e.target.value as FilterValue["operator"])}
        className="rounded-lg border border-gray-300 px-2 py-1.5 text-sm focus:border-blue-500 focus:ring-2 focus:ring-blue-500 focus:ring-opacity-30"
      >
        {operators.map((op) => (
          <option key={op.value} value={op.value}>
            {op.label}
          </option>
        ))}
      </select>
    );
  };

  const renderInput = () => {
    switch (field.type) {
      case "text":
        return (
          <input
            type="text"
            value={value?.value || ""}
            onChange={(e) => handleChange(e.target.value)}
            placeholder={`Filter by ${field.name.toLowerCase()}...`}
            className="flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-2 focus:ring-blue-500 focus:ring-opacity-30"
          />
        );

      case "number":
        return (
          <input
            type="number"
            value={value?.value ?? ""}
            onChange={(e) => handleChange(e.target.value ? parseFloat(e.target.value) : null)}
            placeholder="Enter number..."
            className="flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-2 focus:ring-blue-500 focus:ring-opacity-30"
          />
        );

      case "date":
        return (
          <input
            type="date"
            value={value?.value || ""}
            onChange={(e) => handleChange(e.target.value || null)}
            className="flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-2 focus:ring-blue-500 focus:ring-opacity-30"
          />
        );

      case "boolean":
        return (
          <select
            value={value?.value ?? ""}
            onChange={(e) => handleChange(e.target.value === "" ? null : e.target.value === "true")}
            className="flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-2 focus:ring-blue-500 focus:ring-opacity-30"
          >
            <option value="">All</option>
            <option value="true">Yes</option>
            <option value="false">No</option>
          </select>
        );

      case "select":
        return (
          <select
            value={value?.value ?? ""}
            onChange={(e) => handleChange(e.target.value || null)}
            className="flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-2 focus:ring-blue-500 focus:ring-opacity-30"
          >
            <option value="">All</option>
            {field.options?.map((option) => (
              <option key={option} value={option}>
                {option}
              </option>
            ))}
          </select>
        );

      case "multi-select":
        return (
          <div className="flex-1 flex flex-wrap gap-2">
            {field.options?.map((option) => (
              <label key={option} className="inline-flex items-center">
                <input
                  type="checkbox"
                  checked={(value?.value || []).includes(option)}
                  onChange={(e) => {
                    const currentValues = value?.value || [];
                    if (e.target.checked) {
                      handleChange([...currentValues, option]);
                    } else {
                      handleChange(currentValues.filter((v: string) => v !== option));
                    }
                  }}
                  className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                />
                <span className="ml-1 text-sm text-gray-700">{option}</span>
              </label>
            ))}
          </div>
        );

      case "link_to_card":
        return (
          <input
            type="number"
            value={value?.value ?? ""}
            onChange={(e) => handleChange(e.target.value ? parseInt(e.target.value, 10) : null)}
            placeholder="Card ID..."
            className="flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-2 focus:ring-blue-500 focus:ring-opacity-30"
          />
        );

      default:
        return null;
    }
  };

  return (
    <div className="flex items-center gap-2">
      {renderOperatorSelect()}
      {renderInput()}
    </div>
  );
}

interface ActiveFilterDisplayProps {
  fieldName: string;
  value: FilterValue;
  onClear: () => void;
}

export function ActiveFilterDisplay({ fieldName, value, onClear }: ActiveFilterDisplayProps) {
  const getDisplayValue = (): string => {
    if (value.type === "boolean") {
      return value.value ? "Yes" : "No";
    }
    if (value.type === "multi-select" && Array.isArray(value.value)) {
      return value.value.join(", ");
    }
    if (value.type === "date" && value.value) {
      return new Date(value.value).toLocaleDateString();
    }
    return String(value.value || "");
  };

  const getOperatorLabel = (): string => {
    switch (value.operator) {
      case "contains": return "~";
      case "equals": return "=";
      case "startsWith": return "^";
      case "gt": return ">";
      case "gte": return "≥";
      case "lt": return "<";
      case "lte": return "≤";
      case "before": return "<";
      case "after": return ">";
      case "any": return "∋";
      default: return "";
    }
  };

  return (
    <div className="inline-flex items-center gap-1 px-2 py-1 bg-blue-50 border border-blue-200 rounded-lg text-sm">
      <span className="font-medium text-gray-700">{fieldName}:</span>
      <span className="text-gray-500">{getOperatorLabel()}</span>
      <span className="text-gray-900">{getDisplayValue()}</span>
      <button
        onClick={onClear}
        className="ml-1 text-blue-600 hover:text-blue-800 hover:bg-blue-100 p-0.5 rounded"
        title="Remove filter"
      >
        <svg xmlns="http://www.w3.org/2000/svg" className="h-3.5 w-3.5" viewBox="0 0 20 20" fill="currentColor">
          <path fillRule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clipRule="evenodd" />
        </svg>
      </button>
    </div>
  );
}
