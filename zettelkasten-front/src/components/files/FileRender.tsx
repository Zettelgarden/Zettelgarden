import React, { useEffect, useState } from 'react';
import { File } from '../../models/File';
import { downloadFile } from '../../api/files';

interface FileRenderProps {
  file: File;
}

export const FileRender = ({ file }: FileRenderProps) => {
  const [fileSrc, setFileSrc] = useState<string>('');
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    if (file.id) {
      setIsLoading(true);
      downloadFile(file.id.toString())
        .then((blobUrl) => {
          if (blobUrl) {
            setFileSrc(blobUrl);
          }
        })
        .catch((error) => {
          console.error('Error fetching file:', error);
        })
        .finally(() => {
          setIsLoading(false);
        });
    }
  }, [file]);

  const isImage =
    file.filetype === 'image/png' ||
    file.filetype === 'image/jpeg' ||
    file.filetype === 'image/jpg' ||
    file.filetype === 'image/gif' ||
    file.filetype === 'image/webp';
  const isPDF = file.filetype === 'application/pdf';

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
          <a
            href={fileSrc}
            download={file.name}
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

          {!isLoading && !isImage && !isPDF && (
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
