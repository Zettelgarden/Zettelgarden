import React, { useRef, forwardRef, ForwardedRef, useState } from 'react';
import { Card } from '../../models/Card';
import { uploadFile } from '../../api/files';
import { Button } from '../../components/Button';
import { Spinner } from '../ui/Spinner';
import { useUIState } from '../../contexts/UIStateContext';
import { useToast } from '../toast/ToastContext';

interface FileUploadProps {
  card: Card;
  children?: React.ReactNode;
}

export const FileUpload = forwardRef(
  (
    { card, children }: FileUploadProps,
    ref: ForwardedRef<HTMLInputElement>,
  ) => {
    const localFileInputRef = useRef<HTMLInputElement | null>(null);
    const inputRef = (ref ||
      localFileInputRef) as React.RefObject<HTMLInputElement>;
    const { setRefreshFiles } = useUIState();
    const { showToast } = useToast();
    const [isUploading, setIsUploading] = useState(false);
    const [uploadProgress, setUploadProgress] = useState({
      current: 0,
      total: 0,
      filename: '',
    });

    const handleFileSelect = async (
      event: React.ChangeEvent<HTMLInputElement>,
    ) => {
      const files = event.target.files;
      if (files && files.length > 0) {
        setIsUploading(true);
        setUploadProgress({
          current: 0,
          total: files.length,
          filename: files[0].name,
        });

        for (let i = 0; i < files.length; i++) {
          setUploadProgress({
            current: i + 1,
            total: files.length,
            filename: files[i].name,
          });
          try {
            const response = await uploadFile(files[i], card.id);
            if ('error' in response) {
              showToast('error', 'Upload Failed', response['message']);
            } else {
              showToast('success', 'File Uploaded', response['file']['name']);
              setRefreshFiles(true);
            }
          } catch (error) {
            showToast('error', 'Upload Failed', String(error));
          }
        }

        setIsUploading(false);
        setUploadProgress({ current: 0, total: 0, filename: '' });
        // Reset the input so the same file can be selected again
        if (inputRef.current) {
          inputRef.current.value = '';
        }
      }
    };

    const handleButtonClick = () => {
      if (inputRef.current && !isUploading) {
        inputRef.current.click();
      }
    };

    return (
      <div>
        {children && (
          <div
            onClick={handleButtonClick}
            className={isUploading ? 'pointer-events-none opacity-50' : ''}
          >
            {children}
          </div>
        )}
        <input
          type="file"
          ref={inputRef}
          style={{ display: 'none' }}
          onChange={handleFileSelect}
          multiple
          disabled={isUploading}
        />

        {/* Upload in progress overlay */}
        {isUploading && (
          <div className="fixed inset-0 bg-black bg-opacity-30 z-50 flex items-center justify-center">
            <div className="bg-white rounded-lg p-6 shadow-xl min-w-[300px]">
              <div className="flex items-center gap-3 mb-3">
                <Spinner size="md" className="text-blue-600" />
                <span className="text-gray-700 font-medium">
                  Uploading files...
                </span>
              </div>
              {uploadProgress.total > 0 && (
                <div className="space-y-2">
                  <div className="flex justify-between text-sm text-gray-500">
                    <span className="truncate max-w-[200px]">
                      {uploadProgress.filename}
                    </span>
                    <span>
                      {uploadProgress.current} / {uploadProgress.total}
                    </span>
                  </div>
                  <div className="w-full bg-gray-200 rounded-full h-2">
                    <div
                      className="bg-blue-600 h-2 rounded-full transition-all duration-300"
                      style={{
                        width: `${
                          (uploadProgress.current / uploadProgress.total) * 100
                        }%`,
                      }}
                    />
                  </div>
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    );
  },
);
