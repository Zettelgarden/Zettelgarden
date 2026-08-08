import React, { useState, useEffect } from 'react';
import { FieldDefinition } from '../../models/Schema';
import { Link } from 'react-router-dom';
import { getCard } from '../../api/cards';
import { Card } from '../../models/Card';
import { CardTag } from '../cards/CardTag';

interface LinkedCardDisplayProps {
  cardId: number;
}

function LinkedCardDisplay({ cardId }: LinkedCardDisplayProps) {
  const [card, setCard] = useState<Card | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    getCard(cardId.toString())
      .then((result) => {
        if (isError(result)) {
          setCard(null);
        } else {
          setCard(result);
        }
      })
      .catch(() => {
        setCard(null);
      })
      .finally(() => {
        setLoading(false);
      });
  }, [cardId]);

  function isError(result: any): result is { error: string } {
    return result && typeof result === 'object' && 'error' in result;
  }

  if (loading) {
    return <span className="text-sm text-gray-500">Loading...</span>;
  }

  if (!card) {
    return (
      <span className="text-blue-600 hover:underline text-sm font-mono">
        {cardId}
      </span>
    );
  }

  return (
    <Link to={`/app/card/${card.id}`} className="inline-block">
      <CardTag card={card} showTitle={true} />
    </Link>
  );
}

interface StructuredDataDisplayProps {
  fields: FieldDefinition[];
  data: Record<string, any>;
}

export function StructuredDataDisplay({
  fields,
  data,
}: StructuredDataDisplayProps) {
  const renderValue = (field: FieldDefinition) => {
    const value = data[field.name];

    if (value === null || value === undefined || value === '') {
      return <span className="text-gray-400 italic">Not set</span>;
    }

    switch (field.type) {
      case 'boolean':
        return value ? (
          <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-green-100 text-green-800">
            Yes
          </span>
        ) : (
          <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-gray-100 text-gray-800">
            No
          </span>
        );

      case 'select':
        return (
          <span className="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium bg-blue-100 text-blue-800 border border-blue-200">
            {value}
          </span>
        );

      case 'multi-select':
        return (
          <div className="flex flex-wrap gap-1">
            {(value as string[]).map((v: string) => (
              <span
                key={v}
                className="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium bg-blue-100 text-blue-800 border border-blue-200"
              >
                {v}
              </span>
            ))}
          </div>
        );

      case 'number':
        return <span className="font-mono text-sm">{value}</span>;

      case 'date':
        // Parse date and display in UTC to avoid timezone shifting
        const dateObj = new Date(value);
        return (
          <span className="text-sm">
            {dateObj.toLocaleDateString(undefined, { timeZone: 'UTC' })}
          </span>
        );

      case 'link_to_card':
        return <LinkedCardDisplay cardId={value} />;

      default:
        return <span className="text-sm">{value}</span>;
    }
  };

  if (fields.length === 0) {
    return null;
  }

  // Only show fields that have values
  const fieldsWithValues = fields.filter((field) => {
    const value = data[field.name];
    return value !== null && value !== undefined && value !== '';
  });

  if (fieldsWithValues.length === 0) {
    return null;
  }

  return (
    <div className="space-y-3">
      <h3 className="text-sm font-semibold text-gray-700 border-b pb-2">
        Structured Data
      </h3>
      <div className="grid grid-cols-1 gap-3">
        {fieldsWithValues.map((field) => (
          <div
            key={field.name}
            className="flex items-baseline justify-between py-1"
          >
            <span className="text-sm font-medium text-gray-600 mr-4">
              {field.name}
            </span>
            <div className="flex-grow text-right">{renderValue(field)}</div>
          </div>
        ))}
      </div>
    </div>
  );
}
