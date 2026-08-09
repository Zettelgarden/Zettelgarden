import React, { useEffect, useState } from 'react';
import ReactMarkdown from 'react-markdown';
import { File } from '../../models/File';
import { downloadFile, renderFile } from '../../api/files';
import { useToast } from '../toast/ToastContext';

interface FileRenderProps {
  file: File;
}

const IMAGE_TYPES = [
  'image/png',
  'image/jpeg',
  'image/jpg',
  'image/gif',
  'image/webp',
];

function isImageType(filetype: string) {
  return IMAGE_TYPES.includes(filetype);
}

function isMarkdown(file: File) {
  return (
    file.filetype === 'text/markdown' ||
    file.filetype === 'text/x-markdown' ||
    /\.(md|markdown)$/i.test(file.name)
  );
}

/**
 * Heuristic: binary blobs read as text contain NUL bytes and/or UTF-8
 * replacement characters; treat those as non-previewable.
 */
function looksLikeText(text: string) {
  return (
    text.length > 0 && !text.includes('\u0000') && !text.includes('\ufffd')
  );
}

export const FileRender = ({ file }: FileRenderProps) => {
  const [fileSrc, setFileSrc] = useState<string>('');
  const [textContent, setTextContent] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const { showToast } = useToast();

  const isImage = isImageType(file.filetype);
  const isPDF = file.filetype === 'application/pdf';

  useEffect(() => {
    if (!file.id) return;
    setIsLoading(true);
    setTextContent(null);
    setFileSrc('');

    // Prefer the already-extracted text (avoids a download for text files).
    if (file.extracted_text) {
      setTextContent(file.extracted_text);
      setIsLoading(false);
      return;
    }

    downloadFile(file.id.toString())
      .then(async (blobUrl) => {
        if (!blobUrl) return;
        // Keep the blob URL so the Download link works for every type.
        setFileSrc(blobUrl);
        if (isImage || isPDF) {
          return;
        }
        // Everything else: try to read the blob as text (markdown, code, …).
        try {
          const res = await fetch(blobUrl);
          const text = await res.text();
          if (looksLikeText(text)) {
            setTextContent(text);
          }
        } catch (error) {
          console.error('Error reading file as text:', error);
        }
      })
      .catch((error) => {
        console.error('Error fetching file:', error);
      })
      .finally(() => {
        setIsLoading(false);
      });
  }, [file.id, file.extracted_text, file.filetype, isImage, isPDF]);

  const handleCopy = async () => {
    if (textContent === null) return;
    try {
      await navigator.clipboard.writeText(textContent);
      showToast('success', 'Copied', 'File text copied to clipboard');
    } catch (error) {
      console.error('Failed to copy text:', error);
      showToast('error', 'Copy Failed', 'Could not copy file text');
    }
  };

  return (
    <div className="fixed inset-0 flex items-center justify-center bg-black/50 z-[1000]">
      <div className="bg-white rounded-lg shadow-lg w-[90%] max-w-4xl max-h-[90vh] overflow-hidden flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-3 border-b border-gray-200">
          <div className="flex items-center gap-2 min-w-0">
            <svg
              className="w-5 h-5 text-gray-400 flex-shrink-0"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
              />
            </svg>
            <span className="text-sm font-medium text-gray-700 truncate">
              {file.name}
            </span>
          </div>
          <div className="flex items-center gap-2 flex-shrink-0">
            {textContent !== null && (
              <button
                onClick={handleCopy}
                className="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm bg-gray-100 hover:bg-gray-200 text-gray-700 rounded-md"
              >
                <svg
                  className="w-4 h-4"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
                  />
                </svg>
                Copy
              </button>
            )}
            <a
              href={fileSrc || undefined}
              download={file.name}
              onClick={
                fileSrc
                  ? undefined
                  : (e) => {
                      e.preventDefault();
                      renderFile(file.id, file.name).catch((error) => {
                        console.error('Error downloading file:', error);
                      });
                    }
              }
              className="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm bg-blue-600 hover:bg-blue-700 text-white rounded-md"
            >
              <svg
                className="w-4 h-4"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
                />
              </svg>
              Download
            </a>
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-auto p-4">
          {isLoading && (
            <div className="flex items-center justify-center h-64">
              <div className="text-gray-500">Loading...</div>
            </div>
          )}

          {!isLoading && isImage && (
            <img
              src={fileSrc}
              alt={file.name}
              className="max-w-full h-auto mx-auto"
            />
          )}

          {!isLoading && isPDF && (
            <iframe
              src={fileSrc}
              title={file.name}
              className="w-full h-[70vh] border border-gray-200 rounded"
            />
          )}

          {!isLoading && textContent !== null && isMarkdown(file) && (
            <div className="prose prose-sm max-w-none">
              <ReactMarkdown>{textContent}</ReactMarkdown>
            </div>
          )}

          {!isLoading && textContent !== null && !isMarkdown(file) && (
            <pre className="whitespace-pre-wrap break-words font-mono text-sm text-gray-800 bg-gray-50 border border-gray-200 rounded-md p-4">
              {textContent}
            </pre>
          )}

          {!isLoading && !isImage && !isPDF && textContent === null && (
            <div className="text-center py-12">
              <svg
                className="mx-auto h-12 w-12 text-gray-400 mb-4"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={1.5}
                  d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
                />
              </svg>
              <p className="text-gray-500 mb-2">
                Preview not available for this file type
              </p>
              <p className="text-sm text-gray-400">{file.filetype}</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
