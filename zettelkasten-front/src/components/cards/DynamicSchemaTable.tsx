import React from "react";
import { useNavigate } from "react-router-dom";
import { SchemaTable } from "../schemas/SchemaTable";

interface DynamicSchemaTableProps {
  schemaId: string;
}

export function DynamicSchemaTable({ schemaId }: DynamicSchemaTableProps) {
  const navigate = useNavigate();

  const handleCardClick = (card: any) => {
    navigate(`/app/card/${card.id}`);
  };

  return (
    <div className="my-4 border-l-4 border-blue-500 pl-4">
      <SchemaTable
        schemaId={Number(schemaId)}
        onCardClick={handleCardClick}
        compact={true}
      />
    </div>
  );
}
