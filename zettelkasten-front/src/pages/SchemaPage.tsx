import React, { useState, useEffect, useMemo } from 'react';
import { SchemaDefinition, FieldDefinition } from '../models/Schema';
import { fetchSchemas, deleteSchema } from '../api/schemas';
import { Dialog, Menu } from '@headlessui/react';
import { setDocumentTitle } from '../utils/title';
import { useNavigate } from 'react-router-dom';
import { formatRelativeTime } from '../utils/scheduler';

type SortKey = 'usage' | 'name' | 'updated';

const MAX_FIELD_CHIPS = 8;

export function SchemaPage() {
  const navigate = useNavigate();
  const [schemas, setSchemas] = useState<SchemaDefinition[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [schemaToDelete, setSchemaToDelete] = useState<SchemaDefinition | null>(
    null,
  );
  const [search, setSearch] = useState('');
  const [sortBy, setSortBy] = useState<SortKey>('usage');

  useEffect(() => {
    setDocumentTitle('Schemas');
    loadSchemas();
  }, []);

  const loadSchemas = () => {
    setLoading(true);
    fetchSchemas()
      .then((data) => {
        setSchemas(data);
        setLoading(false);
      })
      .catch((err) => {
        setError('Failed to load schemas');
        setLoading(false);
        console.error('Error fetching schemas:', err);
      });
  };

  const handleDeleteClick = (schema: SchemaDefinition) => {
    setSchemaToDelete(schema);
    setShowDeleteDialog(true);
  };

  const handleConfirmDelete = async () => {
    if (!schemaToDelete) return;

    setShowDeleteDialog(false);
    setIsDeleting(true);

    try {
      const result = await deleteSchema(schemaToDelete.id);
      if (result.warning) {
        console.warn('Schema deletion warning:', result.warning);
      }
      setSchemaToDelete(null);
      loadSchemas();
    } catch (err) {
      setError('Failed to delete schema');
      console.error('Error deleting schema:', err);
    } finally {
      setIsDeleting(false);
    }
  };

  const handleCreateClick = () => {
    navigate('/app/schemas/new');
  };

  const handleEditClick = (schema: SchemaDefinition) => {
    navigate(`/app/schemas/${schema.id}/edit`);
  };

  const handleOpenTable = (schema: SchemaDefinition) => {
    navigate(`/app/schemas/${schema.id}/table`);
  };

  const handleRowKeyDown = (
    e: React.KeyboardEvent,
    schema: SchemaDefinition,
  ) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      handleOpenTable(schema);
    }
  };

  const getFieldTypesSummary = (fields: FieldDefinition[]): string => {
    const typeCount: Record<string, number> = {};
    fields.forEach((field) => {
      typeCount[field.type] = (typeCount[field.type] || 0) + 1;
    });
    return Object.entries(typeCount)
      .map(([type, count]) => `${count} ${type}${count > 1 ? 's' : ''}`)
      .join(', ');
  };

  const visibleSchemas = useMemo(() => {
    const q = search.trim().toLowerCase();
    let list = schemas;
    if (q) {
      list = schemas.filter((s) => {
        const inName = s.name.toLowerCase().includes(q);
        const inFields = s.fields.some((f) => f.name.toLowerCase().includes(q));
        return inName || inFields;
      });
    }

    const sorted = [...list];
    switch (sortBy) {
      case 'name':
        sorted.sort((a, b) => a.name.localeCompare(b.name));
        break;
      case 'updated':
        sorted.sort((a, b) => b.updated_at.getTime() - a.updated_at.getTime());
        break;
      case 'usage':
      default:
        sorted.sort((a, b) => {
          const ca = a.card_count ?? 0;
          const cb = b.card_count ?? 0;
          if (cb !== ca) return cb - ca;
          return a.name.localeCompare(b.name);
        });
        break;
    }
    return sorted;
  }, [schemas, search, sortBy]);

  if (loading) return <SchemaListSkeleton />;
  if (error) return <div className="p-4 text-red-600">{error}</div>;

  return (
    <div className="p-4">
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-baseline gap-2">
          <h1 className="text-xl font-bold text-gray-900">Schemas</h1>
          {schemas.length > 0 && (
            <span className="text-sm text-gray-500">
              {schemas.length} total
            </span>
          )}
        </div>
        <button
          onClick={handleCreateClick}
          className="px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 text-sm font-medium"
        >
          Create Schema
        </button>
      </div>

      {/* Search + sort (only meaningful once schemas exist) */}
      {schemas.length > 0 && (
        <div className="flex flex-wrap items-center gap-2 mb-4">
          <input
            type="text"
            placeholder="Search schemas…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="flex-grow min-w-[12rem] max-w-sm px-3 py-2 text-sm border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
          />
          <label className="flex items-center gap-2 text-sm text-gray-600">
            <span className="sr-only">Sort by</span>
            <select
              value={sortBy}
              onChange={(e) => setSortBy(e.target.value as SortKey)}
              className="px-2 py-2 text-sm border border-gray-300 rounded-md bg-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            >
              <option value="usage">Most used</option>
              <option value="name">Name (A–Z)</option>
              <option value="updated">Recently updated</option>
            </select>
          </label>
        </div>
      )}

      {/* Empty state (also the only place the explanatory blurb lives) */}
      {schemas.length === 0 ? (
        <div className="text-center text-gray-500 mt-8 max-w-md mx-auto">
          <p className="mb-2 font-medium text-gray-700">No schemas yet</p>
          <p className="mb-4">
            Schemas define custom data structures for your cards. Create a
            schema to add structured fields like ratings, dates, or options to
            your notes.
          </p>
          <button
            onClick={handleCreateClick}
            className="px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 text-sm font-medium"
          >
            Create Your First Schema
          </button>
        </div>
      ) : visibleSchemas.length === 0 ? (
        <div className="text-center text-gray-500 mt-8">
          No schemas match “{search}”.
        </div>
      ) : (
        <div className="grid gap-3">
          {visibleSchemas.map((schema) => (
            <SchemaRow
              key={schema.id}
              schema={schema}
              onOpen={handleOpenTable}
              onEdit={handleEditClick}
              onDelete={handleDeleteClick}
              onKeyDown={handleRowKeyDown}
              getFieldTypesSummary={getFieldTypesSummary}
            />
          ))}
        </div>
      )}

      {showDeleteDialog && schemaToDelete && (
        <Dialog
          open={showDeleteDialog}
          onClose={() => {
            setShowDeleteDialog(false);
            setSchemaToDelete(null);
          }}
          className="fixed inset-0 z-50 flex items-center justify-center"
        >
          <div
            className="fixed inset-0 bg-black bg-opacity-30"
            aria-hidden="true"
          />
          <Dialog.Panel className="bg-white p-6 rounded-lg max-w-md mx-auto relative">
            <Dialog.Title className="text-lg font-semibold mb-4">
              Confirm Delete
            </Dialog.Title>
            <div className="mb-4">
              <p className="text-gray-600 mb-2">
                Are you sure you want to delete the schema "
                {schemaToDelete.name}"?
              </p>
            </div>
            <p className="text-red-600 text-sm mb-4">
              This action cannot be undone. Any cards using this schema will no
              longer display their structured data.
            </p>
            <div className="flex justify-end gap-4">
              <button
                onClick={() => {
                  setShowDeleteDialog(false);
                  setSchemaToDelete(null);
                }}
                className="px-4 py-2 text-gray-600 hover:text-gray-800"
              >
                Cancel
              </button>
              <button
                onClick={handleConfirmDelete}
                disabled={isDeleting}
                className="px-4 py-2 bg-red-500 text-white rounded hover:bg-red-600 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {isDeleting ? 'Deleting...' : 'Delete'}
              </button>
            </div>
          </Dialog.Panel>
        </Dialog>
      )}
    </div>
  );
}

interface SchemaRowProps {
  schema: SchemaDefinition;
  onOpen: (schema: SchemaDefinition) => void;
  onEdit: (schema: SchemaDefinition) => void;
  onDelete: (schema: SchemaDefinition) => void;
  onKeyDown: (e: React.KeyboardEvent, schema: SchemaDefinition) => void;
  getFieldTypesSummary: (fields: FieldDefinition[]) => string;
}

function SchemaRow({
  schema,
  onOpen,
  onEdit,
  onDelete,
  onKeyDown,
  getFieldTypesSummary,
}: SchemaRowProps) {
  const count = schema.card_count ?? 0;
  const shownFields = schema.fields.slice(0, MAX_FIELD_CHIPS);
  const extraFields = schema.fields.length - shownFields.length;

  return (
    <div
      role="button"
      tabIndex={0}
      aria-label={`Open ${schema.name} table view`}
      onClick={() => onOpen(schema)}
      onKeyDown={(e) => onKeyDown(e, schema)}
      className="border rounded-lg p-4 bg-white hover:shadow-md hover:border-blue-300 transition-all cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-500"
    >
      <div className="flex items-start justify-between gap-3">
        <div className="flex-grow min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <h3 className="text-lg font-semibold text-gray-900 truncate">
              {schema.name}
            </h3>
            <span
              className={`px-2 py-0.5 text-xs rounded-full ${
                count > 0
                  ? 'bg-blue-100 text-blue-700'
                  : 'bg-gray-100 text-gray-500'
              }`}
            >
              {count} {count === 1 ? 'card' : 'cards'}
            </span>
          </div>
          <p className="text-sm text-gray-500 mt-1">
            {schema.fields.length} field{schema.fields.length !== 1 ? 's' : ''}{' '}
            · {getFieldTypesSummary(schema.fields)} · Updated{' '}
            {formatRelativeTime(schema.updated_at.toISOString())}
          </p>
        </div>

        {/* Overflow menu: stopPropagation so it doesn't trigger row navigation */}
        <div onClick={(e) => e.stopPropagation()}>
          <Menu as="div" className="relative inline-block text-left">
            <Menu.Button
              className="inline-flex justify-center items-center rounded-md border border-gray-300 px-2 py-1 bg-white text-gray-600 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500"
              aria-label={`Actions for ${schema.name}`}
            >
              ⋯
            </Menu.Button>
            <Menu.Items className="absolute right-0 mt-2 w-44 rounded-md shadow-lg bg-white ring-1 ring-black ring-opacity-5 focus:outline-none z-10">
              <div className="py-1">
                <Menu.Item>
                  {({ active }) => (
                    <button
                      onClick={() => onOpen(schema)}
                      className={`${
                        active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                      } group flex items-center w-full px-3 py-2 text-sm`}
                    >
                      View as Table
                    </button>
                  )}
                </Menu.Item>
                <Menu.Item>
                  {({ active }) => (
                    <button
                      onClick={() => onEdit(schema)}
                      className={`${
                        active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                      } group flex items-center w-full px-3 py-2 text-sm`}
                    >
                      Edit
                    </button>
                  )}
                </Menu.Item>
                <Menu.Item>
                  {({ active }) => (
                    <button
                      onClick={() => onDelete(schema)}
                      className={`${
                        active ? 'bg-gray-100 text-red-600' : 'text-red-600'
                      } group flex items-center w-full px-3 py-2 text-sm`}
                    >
                      Delete
                    </button>
                  )}
                </Menu.Item>
              </div>
            </Menu.Items>
          </Menu>
        </div>
      </div>

      {shownFields.length > 0 && (
        <div className="mt-2 flex flex-wrap gap-2">
          {shownFields.map((field) => (
            <span
              key={field.name}
              className={`px-2 py-1 text-xs rounded-md ${
                field.required
                  ? 'bg-blue-100 text-blue-700 border border-blue-200'
                  : 'bg-gray-100 text-gray-600 border border-gray-200'
              }`}
            >
              {field.name}
              <span className="ml-1 opacity-75">({field.type})</span>
              {field.required && <span className="ml-1">*</span>}
            </span>
          ))}
          {extraFields > 0 && (
            <span className="px-2 py-1 text-xs rounded-md bg-gray-50 text-gray-400 border border-gray-200">
              +{extraFields} more
            </span>
          )}
        </div>
      )}
    </div>
  );
}

function SchemaListSkeleton() {
  return (
    <div className="p-4">
      <div className="flex items-center justify-between mb-4">
        <div className="h-6 w-24 bg-gray-200 rounded animate-pulse" />
        <div className="h-8 w-28 bg-gray-200 rounded animate-pulse" />
      </div>
      <div className="grid gap-3">
        {[0, 1, 2].map((i) => (
          <div key={i} className="border rounded-lg p-4 bg-white">
            <div className="h-5 w-40 bg-gray-200 rounded animate-pulse mb-2" />
            <div className="h-4 w-64 bg-gray-100 rounded animate-pulse mb-3" />
            <div className="flex gap-2">
              <div className="h-6 w-20 bg-gray-100 rounded animate-pulse" />
              <div className="h-6 w-24 bg-gray-100 rounded animate-pulse" />
              <div className="h-6 w-16 bg-gray-100 rounded animate-pulse" />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
