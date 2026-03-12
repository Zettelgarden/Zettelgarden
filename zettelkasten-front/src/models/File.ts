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
  description?: string;
  extracted_text?: string;
  tags?: string[];
  mimetype?: string;
}

export interface FileTag {
  id: number;
  user_id: number;
  name: string;
  file_count?: number;
}

export interface UploadFileResponse {
  message: string;
  file: File;
}

export interface EditFileMetadataParams {
  name: string;
  card_pk: number;
  description?: string;
}

export interface FileUpdateParams {
  name?: string;
  description?: string;
  card_pk?: number;
}

export interface ImportEpubResponse {
  parent_card_id: number;
  child_card_ids: number[];
  metadata: {
    title: string;
    author: string;
    publisher: string;
    year: string;
    description: string;
  };
}
