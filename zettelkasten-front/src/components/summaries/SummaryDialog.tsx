import React from 'react';
import { Modal } from '../ui/Modal';
import { Button } from '../Button';
import { SummarizeJobResponse } from '../../api/summarizer';
import ReactMarkdown from 'react-markdown';

interface SummaryDialogProps {
  summary: SummarizeJobResponse | null;
  isOpen: boolean;
  onClose: () => void;
}

export function SummaryDialog({
  summary,
  isOpen,
  onClose,
}: SummaryDialogProps) {
  return (
    <Modal open={isOpen} onClose={onClose} size="2xl" dialogClassName="z-50">
      <h3 className="text-lg font-medium">Most Recent Summary</h3>
      <div className="mt-4 text-sm text-gray-700 max-h-[50vh] overflow-y-auto pr-2">
        {summary?.result ? (
          <ReactMarkdown>{summary.result}</ReactMarkdown>
        ) : (
          'No summary available.'
        )}
      </div>
      <div className="mt-6 flex justify-end">
        <Button onClick={onClose}>Close</Button>
      </div>
    </Modal>
  );
}
