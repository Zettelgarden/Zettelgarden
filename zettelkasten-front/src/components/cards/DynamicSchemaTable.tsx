import React from 'react';
import { useNavigate } from 'react-router-dom';
import { SchemaTable } from '../schemas/SchemaTable';
import { PartialCard } from '../../models/Card';
import { parseFilterGroups } from '../../utils/schemaFilters';

interface DynamicSchemaTableProps {
  schemaRef: string; // Can be an ID (numeric string) or slug
  columns?: string; // Comma-separated list of column names
  filters?: string; // Filter string like "status=active,priority=high" (AND) or "status=active||status=done" (OR of AND groups)
}

export function DynamicSchemaTable({
  schemaRef,
  columns,
  filters,
}: DynamicSchemaTableProps) {
  const navigate = useNavigate();

  const handleCardClick = (card: PartialCard) => {
    navigate(`/app/card/${card.id}`);
  };

  // Parse columns string into array
  const columnsList = columns
    ? columns.split(',').map((c) => c.trim())
    : undefined;

  // Parse filters string into AND/OR groups
  // ("status=active,priority=high||status=done" ->
  //  [{ status: 'active', priority: 'high' }, { status: 'done' }])
  const filterGroups = React.useMemo(() => {
    if (!filters) return undefined;
    return parseFilterGroups(filters);
  }, [filters]);

  return (
    <div className="my-4 border-l-4 border-blue-500 pl-4">
      <SchemaTable
        schemaRef={schemaRef}
        onCardClick={handleCardClick}
        compact={true}
        columns={columnsList}
        filters={filterGroups}
      />
    </div>
  );
}
