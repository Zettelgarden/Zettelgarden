import React, { useState } from 'react';
import { Modal } from '../ui/Modal';
import { createArticle } from '../../api/cards';
import { useNavigate } from 'react-router-dom';
import { useToast } from '../toast/ToastContext';

interface AddArticleDialogProps {
  show: boolean;
  onClose: () => void;
}

export function AddArticleDialog({ show, onClose }: AddArticleDialogProps) {
  const [url, setUrl] = useState('');
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();
  const { showToast } = useToast();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!url) return;
    setLoading(true);

    try {
      const newCard = await createArticle(url);

      if (!('error' in newCard)) {
        navigate(`/app/card/${newCard.id}`);
        onClose();
      } else {
        showToast('error', 'Failed to save article card', 'Please try again');
      }
    } catch (error) {
      console.error('Failed to add article:', error);
      showToast(
        'error',
        'Failed to add article',
        'Please check the URL and try again',
      );
    } finally {
      setLoading(false);
      setUrl('');
    }
  };

  return (
    <Modal open={show} onClose={onClose} size="md" dialogClassName="z-[80]">
      <h3 className="text-lg font-medium leading-6 text-gray-900">
        Add Article
      </h3>
      <form onSubmit={handleSubmit} className="mt-4 space-y-4">
        <input
          type="url"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          placeholder="Enter article URL"
          className="w-full border rounded px-3 py-2"
        />
        <div className="flex justify-end space-x-2">
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-3 min-h-[44px] rounded bg-gray-200 hover:bg-gray-300"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={loading}
            className="px-4 py-3 min-h-[44px] rounded bg-blue-500 text-white hover:bg-blue-600 disabled:opacity-50"
          >
            {loading ? 'Adding...' : 'Add'}
          </button>
        </div>
      </form>
    </Modal>
  );
}
