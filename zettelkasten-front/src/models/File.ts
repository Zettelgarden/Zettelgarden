// src/models/File.ts
import { PartialCard } from "./Card";

export interface File {
  id: number;
  name: string;
  filetype: string;
  path: string;
  filename: string;
  size: number;
  created_by: number;
  updated_by: number;
  card_pk: number;
  is_deleted: boolean;
  created_at: string;
  updated_at: string;
  thumbnail_path: string | null;
  card: PartialCard;
}

export interface UploadFileResponse {
  message: string;
  file: File;
}

export interface EditFileMetadataParams {
  name: string;
  card_pk: number;
}
