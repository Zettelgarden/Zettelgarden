import React, { useState, useEffect } from "react";
import { SchemaDefinition, FieldDefinition } from "../models/Schema";
import { fetchSchemas, deleteSchema } from "../api/schemas";
import { Dialog } from "@headlessui/react";
import { setDocumentTitle } from "../utils/title";
import { useNavigate } from "react-router-dom";

export function SchemaPage() {
  const navigate = useNavigate();
  const [schemas, setSchemas] = useState<SchemaDefinition[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [schemaToDelete, setSchemaToDelete] = useState<SchemaDefinition | null>(null);

  useEffect(() => {
    setDocumentTitle("Schemas");
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
        setError("Failed to load schemas");
        setLoading(false);
        console.error("Error fetching schemas:", err);
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
        console.warn("Schema deletion warning:", result.warning);
      }
      setSchemaToDelete(null);
      loadSchemas();
    } catch (err) {
      setError("Failed to delete schema");
      console.error("Error deleting schema:", err);
    } finally {
      setIsDeleting(false);
    }
  };

  const handleCreateClick = () => {
    navigate("/app/schemas/new");
  };

  const handleEditClick = (schema: SchemaDefinition) => {
    navigate(`/app/schemas/${schema.id}/edit`);
  };

  const getFieldTypesSummary = (fields: FieldDefinition[]): string => {
    const typeCount: Record<string, number> = {};
    fields.forEach((field) => {
      typeCount[field.type] = (typeCount[field.type] || 0) + 1;
    });
    return Object.entries(typeCount)
      .map(([type, count]) => `${count} ${type}${count > 1 ? "s" : ""}`)
      .join(", ");
  };

  if (loading) return <div className="p-4">Loading schemas...</div>;
  if (error) return <div className="p-4 text-red-600">{error}</div>;

  return (
    <div className="p-4">
      <div className="flex items-center justify-between mb-4">
        <span className="font-bold text-base">Schemas</span>
        <button
          onClick={handleCreateClick}
          className="px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 text-sm font-medium"
        >
          Create Schema
        </button>
      </div>

      <div className="mb-4 text-gray-600 text-sm">
        <p>Schemas define custom data structures for your cards. Create a schema to add structured fields like ratings, dates, or options to your notes.</p>
      </div>

      {schemas.length === 0 && !loading && (
        <div className="text-center text-gray-500 mt-8">
          <p className="mb-4">No schemas found. Create your first schema to get started with structured data.</p>
          <button
            onClick={handleCreateClick}
            className="px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 text-sm font-medium"
          >
            Create Your First Schema
          </button>
        </div>
      )}

      <div className="grid gap-4 mt-4">
        {schemas.map((schema) => (
          <div
            key={schema.id}
            className="border rounded-lg p-4 hover:shadow-md transition-shadow bg-white"
          >
            <div className="flex items-start justify-between">
              <div className="flex-grow">
                <h3 className="text-lg font-semibold text-gray-900">{schema.name}</h3>
                <p className="text-sm text-gray-500 mt-1">
                  {schema.fields.length} field{schema.fields.length !== 1 ? "s" : ""} · {getFieldTypesSummary(schema.fields)}
                </p>
                <div className="mt-2 flex flex-wrap gap-2">
                  {schema.fields.map((field) => (
                    <span
                      key={field.name}
                      className={`px-2 py-1 text-xs rounded-md ${
                        field.required
                          ? "bg-blue-100 text-blue-700 border border-blue-200"
                          : "bg-gray-100 text-gray-600 border border-gray-200"
                      }`}
                    >
                      {field.name}
                      <span className="ml-1 opacity-75">({field.type})</span>
                      {field.required && <span className="ml-1">*</span>}
                    </span>
                  ))}
                </div>
              </div>
              <div className="flex gap-2 ml-4">
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    handleEditClick(schema);
                  }}
                  className="px-3 py-1 text-sm text-gray-600 hover:text-gray-800 hover:bg-gray-100 rounded border border-gray-300"
                >
                  Edit
                </button>
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    navigate(`/app/schemas/${schema.id}/table`);
                  }}
                  className="px-3 py-1 text-sm text-blue-600 hover:text-blue-800 hover:bg-blue-50 rounded border border-blue-300"
                >
                  View as Table
                </button>
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    handleDeleteClick(schema);
                  }}
                  className="px-3 py-1 text-sm text-red-600 hover:text-red-800 hover:bg-red-50 rounded border border-red-300"
                >
                  Delete
                </button>
              </div>
            </div>
          </div>
        ))}
      </div>

      {showDeleteDialog && schemaToDelete && (
        <Dialog
          open={showDeleteDialog}
          onClose={() => {
            setShowDeleteDialog(false);
            setSchemaToDelete(null);
          }}
          className="fixed inset-0 z-50 flex items-center justify-center"
        >
          <div className="fixed inset-0 bg-black bg-opacity-30" aria-hidden="true" />
          <Dialog.Panel className="bg-white p-6 rounded-lg max-w-md mx-auto relative">
            <Dialog.Title className="text-lg font-semibold mb-4">
              Confirm Delete
            </Dialog.Title>
            <div className="mb-4">
              <p className="text-gray-600 mb-2">
                Are you sure you want to delete the schema "{schemaToDelete.name}"?
              </p>
            </div>
            <p className="text-red-600 text-sm mb-4">
              This action cannot be undone. Any cards using this schema will no longer display their structured data.
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
                {isDeleting ? "Deleting..." : "Delete"}
              </button>
            </div>
          </Dialog.Panel>
        </Dialog>
      )}
    </div>
  );
}
