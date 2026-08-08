import React from 'react';
import { Dialog } from '@headlessui/react';
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
    <Dialog open={isOpen} onClose={onClose} className="relative z-50">
      <div className="fixed inset-0 bg-black/30" aria-hidden="true" />
      <div className="fixed inset-0 flex items-center justify-center p-4">
        <Dialog.Panel className="w-full max-w-2xl rounded-2xl bg-white p-6 shadow-xl">
          <Dialog.Title className="text-lg font-medium">
            Most Recent Summary
          </Dialog.Title>
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
        </Dialog.Panel>
      </div>
    </Dialog>
  );
}
