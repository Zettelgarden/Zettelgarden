import React from "react";
import { Card } from "../../models/Card";
import { File } from "../../models/File";
import { FileListItem } from "../files/FileListItem";
import { FileUpload } from "../files/FileUpload";

interface FilesTabProps {
  viewingCard: Card;
  fileUploadRef: React.RefObject<HTMLInputElement>;
  handleDisplayFileOnCardClick: (file: File) => void;
  fileFilterString: string;
  setFileFilterString: (value: string) => void;
  setError: (error: string) => void;
}

export function FilesTab({
  viewingCard,
  fileUploadRef,
  handleDisplayFileOnCardClick,
  fileFilterString,
  setFileFilterString,
  setError,
}: FilesTabProps) {
  function onFileDelete(file_id: number) {}

  return (
    <div>
      <div className="flex p-2">
        <a
          onClick={() => fileUploadRef.current?.click()}
          className="text-blue-600 hover:text-blue-800 cursor-pointer"
        >
          Upload File
        </a>
        <FileUpload ref={fileUploadRef} card={viewingCard} />
      </div>
      {viewingCard.files.length > 0 && (
        <div>
          <ul>
            {viewingCard.files.map((file, index) => (
              <FileListItem
                file={file}
                onDelete={onFileDelete}
                setRefreshFiles={(refresh: boolean) => {}}
                displayFileOnCard={(file: File) => {
                  handleDisplayFileOnCardClick(file);
                }}
                filterString={fileFilterString}
                setFilterString={setFileFilterString}
              />
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}