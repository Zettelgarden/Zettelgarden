import React from 'react';
import { Entity } from '../../models/Card';
import { Link } from 'react-router-dom';
import { CardTag } from '../cards/CardTag';
import {
  Table,
  TableHead,
  TableBody,
  TableRow,
  TableHeader,
  TableCell,
} from '../table/TableComponents';

interface EntityTableProps {
  entities: Entity[];
  selectedEntities: number[];
  selectionMode: boolean;
  onEdit: (entity: Entity, event: React.MouseEvent) => void;
  onClick: (entity: Entity, event: React.MouseEvent) => void;
  getSelectionInfo: (entityId: number) => string | null;
  onSelectAll: (checked: boolean) => void;
}

export function EntityTable({
  entities,
  selectedEntities,
  selectionMode,
  onEdit,
  onClick,
  getSelectionInfo,
  onSelectAll,
}: EntityTableProps) {
  return (
    <div className="overflow-x-auto">
      <Table>
        <TableHead>
          <TableRow>
            {selectionMode && (
              <TableHeader className="w-12">
                <input
                  type="checkbox"
                  checked={
                    selectedEntities.length === entities.length &&
                    entities.length > 0
                  }
                  onChange={(e) => onSelectAll(e.target.checked)}
                  className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                />
              </TableHeader>
            )}
            <TableHeader>Name</TableHeader>
            <TableHeader>Type</TableHeader>
            <TableHeader>Description</TableHeader>
            <TableHeader className="w-20">Cards</TableHeader>
            <TableHeader>Linked Card</TableHeader>
            <TableHeader>Created</TableHeader>
            <TableHeader>Updated</TableHeader>
            <TableHeader className="w-20">Actions</TableHeader>
          </TableRow>
        </TableHead>
        <TableBody>
          {entities.map((entity) => {
            const isSelected = selectedEntities.includes(entity.id);
            const selectionInfo = getSelectionInfo(entity.id);

            return (
              <TableRow
                key={entity.id}
                className={`hover:bg-gray-50 cursor-pointer ${
                  isSelected ? 'bg-blue-50' : ''
                } ${selectionMode ? 'hover:bg-blue-25' : ''}`}
                onClick={(e) => onClick(entity, e)}
              >
                {selectionMode && (
                  <TableCell>
                    <div
                      className={`w-5 h-5 border-2 rounded-md flex items-center justify-center transition-colors
                        ${
                          isSelected
                            ? 'border-blue-500 bg-blue-500'
                            : 'border-gray-300 bg-white hover:border-blue-400'
                        }`}
                    >
                      {isSelected && (
                        <svg
                          className="w-3 h-3 text-white"
                          viewBox="0 0 20 20"
                          fill="currentColor"
                        >
                          <path
                            fillRule="evenodd"
                            d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                            clipRule="evenodd"
                          />
                        </svg>
                      )}
                    </div>
                  </TableCell>
                )}
                <TableCell>
                  <div className="flex flex-col">
                    <span
                      className={`font-semibold ${
                        isSelected ? 'text-blue-800' : 'text-gray-800'
                      }`}
                    >
                      {entity.name}
                    </span>
                    {selectionInfo && (
                      <span
                        className={`text-sm font-medium mt-1 ${
                          selectionInfo === 'Primary'
                            ? 'text-green-600'
                            : 'text-blue-600'
                        }`}
                      >
                        {selectionInfo}
                      </span>
                    )}
                  </div>
                </TableCell>
                <TableCell>
                  <span className="text-sm px-2 py-1 bg-gray-100 rounded-full text-gray-600">
                    {entity.type}
                  </span>
                </TableCell>
                <TableCell>
                  <span className="text-gray-600 text-sm max-w-xs truncate block">
                    {entity.description || '-'}
                  </span>
                </TableCell>
                <TableCell className="text-center">
                  <span className="font-medium">{entity.card_count}</span>
                </TableCell>
                <TableCell>
                  {entity.card ? (
                    <Link
                      to={`/app/card/${entity.card.id}`}
                      onClick={(e) => e.stopPropagation()}
                      className="text-blue-600 hover:text-blue-800"
                    >
                      <CardTag card={entity.card} showTitle={false} />
                    </Link>
                  ) : (
                    <span className="text-gray-400 text-sm">-</span>
                  )}
                </TableCell>
                <TableCell>
                  <span className="text-gray-500 text-sm">
                    {new Date(entity.created_at).toLocaleDateString()}
                  </span>
                </TableCell>
                <TableCell>
                  <span className="text-gray-500 text-sm">
                    {new Date(entity.updated_at).toLocaleDateString()}
                  </span>
                </TableCell>
                <TableCell>
                  <button
                    onClick={(e) => onEdit(entity, e)}
                    className="text-gray-600 hover:text-blue-600 p-1 rounded-full hover:bg-gray-100"
                    title="Edit entity"
                  >
                    <svg
                      xmlns="http://www.w3.org/2000/svg"
                      className="h-4 w-4"
                      viewBox="0 0 20 20"
                      fill="currentColor"
                    >
                      <path d="M13.586 3.586a2 2 0 112.828 2.828l-.793.793-2.828-2.828.793-.793zM11.379 5.793L3 14.172V17h2.828l8.38-8.379-2.83-2.828z" />
                    </svg>
                  </button>
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}
