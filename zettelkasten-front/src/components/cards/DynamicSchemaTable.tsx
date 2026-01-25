import React from "react";
import { useNavigate } from "react-router-dom";
import { SchemaTable } from "../schemas/SchemaTable";

interface DynamicSchemaTableProps {
  schemaRef: string; // Can be an ID (numeric string) or slug
  columns?: string; // Comma-separated list of column names
  filters?: string; // Filter string like "status=active,priority=high"
}

export function DynamicSchemaTable({ schemaRef, columns, filters }: DynamicSchemaTableProps) {
  const navigate = useNavigate();

  const handleCardClick = (card: any) => {
    navigate(`/app/card/${card.id}`);
  };

  // Parse columns string into array
  const columnsList = columns ? columns.split(',').map(c => c.trim()) : undefined;

  // Parse filters string into object
  const filtersObj = React.useMemo(() => {
    if (!filters) return undefined;
    const result: Record<string, string> = {};
    filters.split(',').forEach(f => {
      const [key, value] = f.split('=').map(s => s.trim());
      if (key && value) {
        result[key] = value;
      }
    });
    return result;
  }, [filters]);

  return (
    <div className="my-4 border-l-4 border-blue-500 pl-4">
      <SchemaTable
        schemaRef={schemaRef}
        onCardClick={handleCardClick}
        compact={true}
        columns={columnsList}
        filters={filtersObj}
      />
    </div>
  );
}
