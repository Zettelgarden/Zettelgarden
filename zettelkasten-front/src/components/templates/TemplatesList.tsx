import React, { useState, useEffect } from "react";
import { getTemplates, deleteTemplate, saveAsTemplate } from "../../api/templates";
import { CardTemplate } from "../../models/Card";
import { TemplateVariablesHelp } from "./TemplateVariablesHelp";

interface TemplatesListProps {
  onTemplateDeleted?: () => void;
}

export function TemplatesList({ onTemplateDeleted }: TemplatesListProps) {
  const [templates, setTemplates] = useState<CardTemplate[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [newTemplateTitle, setNewTemplateTitle] = useState("");
  const [newTemplateBody, setNewTemplateBody] = useState("");
  const [creating, setCreating] = useState(false);
  const [viewingTemplate, setViewingTemplate] = useState<CardTemplate | null>(null);

  useEffect(() => {
    fetchTemplates();
  }, []);

  async function fetchTemplates() {
    try {
      setLoading(true);
      const fetchedTemplates = await getTemplates();
      setTemplates(fetchedTemplates);
      setError("");
    } catch (err) {
      setError("Failed to load templates");
      console.error(err);
    } finally {
      setLoading(false);
    }
  }

  async function handleDeleteTemplate(id: number) {
    if (window.confirm("Are you sure you want to delete this template?")) {
      try {
        await deleteTemplate(id);
        setTemplates(templates.filter(template => template.id !== id));
        if (onTemplateDeleted) {
          onTemplateDeleted();
        }
      } catch (err) {
        setError("Failed to delete template");
        console.error(err);
      }
    }
  }

  async function handleCreateTemplate() {
    if (!newTemplateTitle.trim() || !newTemplateBody.trim()) {
      setError("Title and body are required");
      return;
    }

    try {
      setCreating(true);
      const newTemplate = await saveAsTemplate(newTemplateTitle, newTemplateBody);
      setTemplates([...templates, newTemplate]);
      setShowCreateDialog(false);
      setNewTemplateTitle("");
      setNewTemplateBody("");
      setError("");
    } catch (err) {
      setError("Failed to create template");
      console.error(err);
    } finally {
      setCreating(false);
    }
  }

  if (loading) {
    return <div className="text-gray-600">Loading templates...</div>;
  }

  if (error) {
    return <div className="text-red-600">{error}</div>;
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <h3 className="text-lg font-medium">Your Templates</h3>
          <TemplateVariablesHelp />
        </div>
        <button
          onClick={() => setShowCreateDialog(true)}
          className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
        >
          <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
            <path fillRule="evenodd" d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z" clipRule="evenodd" />
          </svg>
          Create Template
        </button>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 rounded-md p-3">
          <div className="text-red-700 text-sm">{error}</div>
        </div>
      )}

      {templates.length === 0 ? (
        <div className="text-center py-8">
          <p className="text-gray-600 mb-2">No templates found.</p>
          <p className="text-sm text-gray-500">Create a template to reuse card structures.</p>
        </div>
      ) : (
        <div className="space-y-2">
          {templates.map((template) => (
            <div key={template.id} className="flex justify-between items-center p-3 border rounded-md bg-white">
              <div
                className="flex-1 cursor-pointer hover:bg-gray-50 -m-3 p-3 rounded-md transition-colors"
                onClick={() => setViewingTemplate(template)}
              >
                <h4 className="font-medium">{template.title}</h4>
                <p className="text-sm text-gray-600">
                  Created: {new Date(template.created_at).toLocaleDateString()}
                </p>
              </div>
              <button
                onClick={() => handleDeleteTemplate(template.id)}
                className="text-red-500 hover:text-red-700 px-2 py-1"
              >
                Delete
              </button>
            </div>
          ))}
        </div>
      )}

      {showCreateDialog && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl p-6 max-w-2xl w-full mx-4">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold">Create Template</h3>
              <TemplateVariablesHelp />
            </div>

            <div className="space-y-4">
              <div>
                <label htmlFor="template-title" className="block text-sm font-medium text-gray-700 mb-1">
                  Title <span className="text-gray-500 text-xs">(will become the card title)</span>
                </label>
                <input
                  id="template-title"
                  type="text"
                  value={newTemplateTitle}
                  onChange={(e) => setNewTemplateTitle(e.target.value)}
                  placeholder="e.g., Meeting Notes, Daily Journal"
                  className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
                />
              </div>

              <div>
                <label htmlFor="template-body" className="block text-sm font-medium text-gray-700 mb-1">
                  Body
                </label>
                <textarea
                  id="template-body"
                  value={newTemplateBody}
                  onChange={(e) => setNewTemplateBody(e.target.value)}
                  placeholder="Template content (use template variables like $date, $time, etc.)"
                  rows={10}
                  className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm font-mono"
                />
              </div>
            </div>

            <div className="flex justify-end gap-3 mt-6">
              <button
                onClick={() => {
                  setShowCreateDialog(false);
                  setNewTemplateTitle("");
                  setNewTemplateBody("");
                  setError("");
                }}
                className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
              >
                Cancel
              </button>
              <button
                onClick={handleCreateTemplate}
                disabled={creating}
                className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 disabled:opacity-50"
              >
                {creating ? "Creating..." : "Create Template"}
              </button>
            </div>
          </div>
        </div>
      )}

      {viewingTemplate && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl p-6 max-w-2xl w-full mx-4 max-h-[80vh] flex flex-col">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold">{viewingTemplate.title}</h3>
              <button
                onClick={() => setViewingTemplate(null)}
                className="text-gray-400 hover:text-gray-600"
              >
                <svg xmlns="http://www.w3.org/2000/svg" className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>

            <div className="flex-1 overflow-y-auto">
              <div className="mb-4">
                <p className="text-sm text-gray-500 mb-2">
                  Created: {new Date(viewingTemplate.created_at).toLocaleDateString()}
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Template Body
                </label>
                <div className="bg-gray-50 rounded-md p-4 border border-gray-200">
                  <pre className="whitespace-pre-wrap font-mono text-sm text-gray-800">
                    {viewingTemplate.body}
                  </pre>
                </div>
              </div>
            </div>

            <div className="flex justify-end gap-3 mt-6 pt-4 border-t">
              <button
                onClick={() => setViewingTemplate(null)}
                className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
              >
                Close
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
