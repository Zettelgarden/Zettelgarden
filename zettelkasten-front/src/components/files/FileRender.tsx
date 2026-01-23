import React, { useEffect, useState } from "react";
import { File } from "../../models/File";
import { downloadFile } from "../../api/files";

interface FileRenderProps {
  file: File;
}

export const FileRender = ({ file }: FileRenderProps) => {
  const [imageSrc, setImageSrc] = useState<string>("");
  useEffect(() => {
    if (file.id) {
      downloadFile(file.id.toString())
        .then((blobUrl) => {
          if (blobUrl) {
            setImageSrc(blobUrl);
          }
        })
        .catch((error) => {
          console.error("Error fetching image:", error);
        });
    }
  }, [file]);
  return (
    <div className="fixed inset-0 flex items-center justify-center bg-black/50 z-[1000]">
      <div className="bg-white p-5 rounded-lg shadow-lg w-[90%]">
      {(file.filetype === "image/png" || file.filetype === "image/jpeg" || file.filetype === "image/jpg") && (
          <img src={imageSrc} className="max-w-full h-auto" />
        )}
      </div>
    </div>
  );
};
