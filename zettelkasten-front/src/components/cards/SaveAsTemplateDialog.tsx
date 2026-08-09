import React, { useState } from 'react';
import { saveAsTemplate } from '../../api/templates';
import { Button } from '../Button';
import { Modal } from '../ui/Modal';
import { TemplateVariablesHelp } from '../templates/TemplateVariablesHelp';

interface SaveAsTemplateDialogProps {
  body: string;
  title?: string;
  onClose: () => void;
  onSuccess: (message: string) => void;
}

export function SaveAsTemplateDialog({
  body,
  title: cardTitle = '',
  onClose,
  onSuccess,
}: SaveAsTemplateDialogProps) {
  const [name, setName] = useState('');
  const [title, setTitle] = useState(cardTitle);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState('');

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();

    if (!name.trim()) {
      setError('Please enter a name for the template');
      return;
    }

    setIsSubmitting(true);
    setError('');

    try {
      await saveAsTemplate(name, title, body);
      onSuccess('Template saved successfully');
      onClose();
    } catch (err) {
      setError('Failed to save template');
      setIsSubmitting(false);
    }
  }

  return (
    <Modal open onClose={onClose} size="md" dialogClassName="z-50">
      <h2 className="text-xl font-semibold mb-4">Save as Template</h2>

      {error && (
        <div className="mb-4 p-3 bg-red-50 text-red-700 rounded-md">
          {error}
        </div>
      )}

      <form onSubmit={handleSubmit}>
        <div className="mb-4">
          <label
            htmlFor="template-name"
            className="block text-sm font-medium text-gray-700 mb-1"
          >
            Template Name
          </label>
          <input
            id="template-name"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="block w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
            placeholder="e.g., Daily Journal, Meeting Notes"
            disabled={isSubmitting}
          />
          <p className="text-sm text-gray-500 mt-1">
            Display name shown in template lists
          </p>
        </div>

        <div className="mb-4">
          <label
            htmlFor="template-title"
            className="block text-sm font-medium text-gray-700 mb-1"
          >
            Card Title
          </label>
          <input
            id="template-title"
            type="text"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            className="block w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
            placeholder="Title used when creating new cards"
            disabled={isSubmitting}
          />
          <div className="mt-2">
            <TemplateVariablesHelp />
          </div>
        </div>

        <div className="flex justify-end space-x-3">
          <Button onClick={onClose} variant="outline" disabled={isSubmitting}>
            Cancel
          </Button>
          <button
            type="submit"
            className="px-4 py-3 min-h-[44px] bg-palette-dark text-white font-semibold rounded hover:bg-palette-darkest focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
            disabled={isSubmitting}
          >
            {isSubmitting ? 'Saving...' : 'Save Template'}
          </button>
        </div>
      </form>
    </Modal>
  );
}
