import React, { useRef, forwardRef, ForwardedRef } from "react";
import { Card } from "../../models/Card";
import { uploadFile } from "../../api/files";
import { Button } from "../../components/Button";
import { useUIState } from "../../contexts/UIStateContext";
import { useToast } from "../toast/ToastContext";

interface FileUploadProps {
  card: Card;
  children?: React.ReactNode;
}

export const FileUpload = forwardRef(({
  card,
  children,
}: FileUploadProps, ref: ForwardedRef<HTMLInputElement>) => {
  const localFileInputRef = useRef<HTMLInputElement | null>(null);
  const inputRef = (ref || localFileInputRef) as React.RefObject<HTMLInputElement>;
  const { setRefreshFiles } = useUIState();
  const { showToast } = useToast();

  const handleFileSelect = async (
    event: React.ChangeEvent<HTMLInputElement>,
  ) => {
    const files = event.target.files;
    if (files && files.length > 0) {
      for (let i = 0; i < files.length; i++) {
        try {
          const response = await uploadFile(files[i], card.id);
          if ("error" in response) {
            showToast("error", "Upload Failed", response["message"]);
          } else {
            showToast("success", "File Uploaded", response["file"]["name"]);
            setRefreshFiles(true);
          }
        } catch (error) {
          showToast("error", "Upload Failed", String(error));
        }
      }
    }
  };

  const handleButtonClick = () => {
    if (inputRef.current) {
      inputRef.current.click();
    }
  };

  return (
    <div>
      {children && <div onClick={handleButtonClick}>{children}</div>}
      <input
        type="file"
        ref={inputRef}
        style={{ display: "none" }}
        onChange={handleFileSelect}
        multiple
      />
    </div>
  );
});
