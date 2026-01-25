import React from "react";
import { useNavigate } from "react-router-dom";
import { SchemaTable } from "../schemas/SchemaTable";

interface DynamicSchemaTableProps {
  schemaRef: string; // Can be an ID (numeric string) or slug
  columns?: string; // Comma-separated list of column names
}

export function DynamicSchemaTable({ schemaRef, columns }: DynamicSchemaTableProps) {
  const navigate = useNavigate();

  const handleCardClick = (card: any) => {
    navigate(`/app/card/${card.id}`);
  };

  // Parse columns string into array
  const columnsList = columns ? columns.split(',').map(c => c.trim()) : undefined;

  return (
    <div className="my-4 border-l-4 border-blue-500 pl-4">
      <SchemaTable
        schemaRef={schemaRef}
        onCardClick={handleCardClick}
        compact={true}
        columns={columnsList}
      />
    </div>
  );
}
