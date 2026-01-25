import React, { useState, useRef, useEffect } from "react";
import { FieldDefinition } from "../../models/Schema";
import { Card } from "../../models/Card";
import { saveExistingCard, getCard } from "../../api/cards";
import { CardLink } from "../cards/CardLink";

interface EditableCellProps {
  card: Card;
  field: FieldDefinition;
  onSave: () => void;
}

// LinkedCardDisplay component for link_to_card fields
interface LinkedCardDisplayProps {
  cardId: number;
}

function LinkedCardDisplay({ cardId }: LinkedCardDisplayProps) {
  const [linkedCard, setLinkedCard] = useState<Card | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    getCard(cardId.toString())
      .then((result) => {
        if (isError(result)) {
          setLinkedCard(null);
        } else {
          setLinkedCard(result);
        }
      })
      .catch(() => {
        setLinkedCard(null);
      })
      .finally(() => {
        setLoading(false);
      });
  }, [cardId]);

  function isError(result: any): result is { error: string } {
    return result && typeof result === "object" && "error" in result;
  }

  if (loading) {
    return <span className="text-sm text-gray-500">Loading...</span>;
  }

  if (!linkedCard) {
    return <span className="text-blue-600 hover:underline text-sm font-mono">{cardId}</span>;
  }

  return <CardLink card={linkedCard} showTitle={true} handleViewBacklink={() => {}} />;
}

export function EditableCell({ card, field, onSave }: EditableCellProps) {
  const [isEditing, setIsEditing] = useState(false);
  const [value, setValue] = useState<any>(card.structured_data?.[field.name] ?? "");
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [linkedCardLoading, setLinkedCardLoading] = useState(false);
  const inputRef = useRef<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>(null);

  useEffect(() => {
    if (isEditing && inputRef.current) {
      inputRef.current.focus();
    }
  }, [isEditing]);

  const handleClick = () => {
    setIsEditing(true);
  };

  const handleBlur = async () => {
    if (isEditing) {
      await saveValue();
    }
  };

  const handleSaveClick = async () => {
    await saveValue();
  };

  const handleCancelClick = () => {
    setValue(card.structured_data?.[field.name] ?? "");
    setIsEditing(false);
    setError(null);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      (e.currentTarget as HTMLElement).blur();
    } else if (e.key === "Escape") {
      setValue(card.structured_data?.[field.name] ?? "");
      setIsEditing(false);
      setError(null);
    }
  };

  const saveValue = async () => {
    // Skip save if value hasn't changed
    const currentValue = card.structured_data?.[field.name];
    if (value === currentValue || (value === "" && (currentValue === null || currentValue === undefined))) {
      setIsEditing(false);
      setError(null);
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      const updatedCard = {
        ...card,
        structured_data: {
          ...card.structured_data,
          [field.name]: value,
        },
      };

      await saveExistingCard(updatedCard);
      setIsEditing(false);
      onSave();
    } catch (err) {
      console.error("Failed to save cell value:", err);
      setError("Failed to save");
      setIsLoading(false);
    }
  };

  const renderDisplayValue = () => {
    const displayValue = card.structured_data?.[field.name];

    if (displayValue === null || displayValue === undefined || displayValue === "") {
      return <span className="text-gray-400 italic cursor-pointer hover:bg-gray-100 px-1 rounded">—</span>;
    }

    switch (field.type) {
      case "boolean":
        return (
          <span className="cursor-pointer hover:bg-gray-100 px-1 rounded">
            {displayValue ? "Yes" : "No"}
          </span>
        );
      case "multi-select":
        return (
          <span className="cursor-pointer hover:bg-gray-100 px-1 rounded">
            {(displayValue as string[]).join(", ")}
          </span>
        );
      case "date":
        return (
          <span className="cursor-pointer hover:bg-gray-100 px-1 rounded">
            {new Date(displayValue).toLocaleDateString()}
          </span>
        );
      case "link_to_card":
        return (
          <span className="cursor-pointer hover:bg-gray-100 px-1 rounded">
            <LinkedCardDisplay cardId={displayValue} />
          </span>
        );
      default:
        return (
          <span className="cursor-pointer hover:bg-gray-100 px-1 rounded">
            {String(displayValue)}
          </span>
        );
    }
  };

  const renderEditInput = () => {
    const baseClassName = "w-full px-2 py-1 text-sm border border-blue-500 rounded focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-opacity-30";

    switch (field.type) {
      case "text":
        return (
          <input
            ref={inputRef as React.RefObject<HTMLInputElement>}
            type="text"
            value={value}
            onChange={(e) => setValue(e.target.value)}
            onBlur={handleBlur}
            onKeyDown={handleKeyDown}
            className={baseClassName}
            disabled={isLoading}
          />
        );

      case "number":
        return (
          <input
            ref={inputRef as React.RefObject<HTMLInputElement>}
            type="number"
            value={value}
            onChange={(e) => setValue(e.target.value ? parseFloat(e.target.value) : "")}
            onBlur={handleBlur}
            onKeyDown={handleKeyDown}
            className={baseClassName}
            disabled={isLoading}
          />
        );

      case "date":
        return (
          <input
            ref={inputRef as React.RefObject<HTMLInputElement>}
            type="date"
            value={value}
            onChange={(e) => setValue(e.target.value || "")}
            onBlur={handleBlur}
            onKeyDown={handleKeyDown}
            className={baseClassName}
            disabled={isLoading}
          />
        );

      case "boolean":
        return (
          <select
            ref={inputRef as React.RefObject<HTMLSelectElement>}
            value={value}
            onChange={(e) => setValue(e.target.value === "true")}
            onBlur={handleBlur}
            className={baseClassName}
            disabled={isLoading}
          >
            <option value="true">Yes</option>
            <option value="false">No</option>
          </select>
        );

      case "select":
        return (
          <select
            ref={inputRef as React.RefObject<HTMLSelectElement>}
            value={value}
            onChange={(e) => setValue(e.target.value)}
            onBlur={handleBlur}
            onKeyDown={handleKeyDown}
            className={baseClassName}
            disabled={isLoading}
          >
            <option value="">Select...</option>
            {field.options?.map((option) => (
              <option key={option} value={option}>
                {option}
              </option>
            ))}
          </select>
        );

      case "multi-select":
        return (
          <div className="flex flex-col gap-2">
            <div className="flex flex-wrap gap-1">
              {field.options?.map((option) => (
                <label key={option} className="inline-flex items-center text-sm">
                  <input
                    type="checkbox"
                    checked={(value || []).includes(option)}
                    onChange={(e) => {
                      const currentValues = value || [];
                      if (e.target.checked) {
                        setValue([...currentValues, option]);
                      } else {
                        setValue(currentValues.filter((v: string) => v !== option));
                      }
                    }}
                    className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                    disabled={isLoading}
                  />
                  <span className="ml-1">{option}</span>
                </label>
              ))}
            </div>
            <div className="flex gap-2">
              <button
                type="button"
                onClick={handleSaveClick}
                className="px-2 py-1 text-xs bg-blue-500 text-white rounded hover:bg-blue-600 disabled:opacity-50"
                disabled={isLoading}
              >
                Save
              </button>
              <button
                type="button"
                onClick={handleCancelClick}
                className="px-2 py-1 text-xs bg-gray-200 text-gray-700 rounded hover:bg-gray-300 disabled:opacity-50"
                disabled={isLoading}
              >
                Cancel
              </button>
            </div>
          </div>
        );

      case "link_to_card":
        return (
          <input
            ref={inputRef as React.RefObject<HTMLInputElement>}
            type="number"
            value={value}
            onChange={(e) => setValue(e.target.value ? parseInt(e.target.value, 10) : "")}
            onBlur={handleBlur}
            onKeyDown={handleKeyDown}
            className={baseClassName}
            placeholder="Card ID"
            disabled={isLoading}
          />
        );

      default:
        return null;
    }
  };

  return (
    <td
      className="px-4 py-3 text-sm text-gray-900 relative"
      onClick={handleClick}
    >
      {isEditing ? (
        <div className="relative">
          {renderEditInput()}
          {isLoading && (
            <div className="absolute inset-0 bg-white bg-opacity-50 flex items-center justify-center">
              <div className="w-4 h-4 border-2 border-blue-500 border-t-transparent rounded-full animate-spin"></div>
            </div>
          )}
        </div>
      ) : (
        <div>
          {renderDisplayValue()}
          {error && (
            <div className="text-xs text-red-500 mt-1">{error}</div>
          )}
        </div>
      )}
    </td>
  );
}
