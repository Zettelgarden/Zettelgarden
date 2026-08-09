import {
  type File,
  type FileTag,
  EditFileMetadataParams,
  UploadFileResponse,
  ImportEpubResponse,
} from '../models/File';
import { GenericResponse } from '../models/common';
import { apiClient, getData } from './client';
import { buildURL } from './client';

const BASE_URL = import.meta.env.VITE_URL;

export function uploadFile(
  file: Blob,
  card_pk: number,
  customFilename?: string,
): Promise<UploadFileResponse> {
  const maxSize = 50 * 1024 * 1024; // 50 MB in bytes
  if (file.size > maxSize) {
    return Promise.reject(
      new Error('File size exceeds the maximum limit of 50 MB.'),
    );
  }

  // Create a FormData object and append the file
  let formData = new FormData();

  if (customFilename && file instanceof File) {
    // Create a new File with custom filename but keep the original file's content and type
    const fileExtension = file.name.split('.').pop() || '';
    const newFile = new File([file], `${customFilename}.${fileExtension}`, {
      type: file.type,
    });
    formData.append('file', newFile);
  } else {
    formData.append('file', file);
  }

  formData.append('card_pk', card_pk.toString()); // Append card_pk to the form data

  // Get token and manually handle FormData upload (skip Content-Type for FormData)
  const token = localStorage.getItem('token');
  const url = buildURL(BASE_URL, '/files/upload');

  return fetch(url, {
    method: 'POST',
    headers: {
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      // Don't set Content-Type for FormData - browser sets it with boundary
    },
    body: formData,
  }).then((response) => {
    if (!response.ok) {
      return response.text().then((text) => {
        throw new Error(
          text || `Request failed with status: ${response.status}`,
        );
      });
    }
    return response.json() as Promise<UploadFileResponse>;
  });
}

export function renderFile(fileId: number, filename: string) {
  const token = localStorage.getItem('token');
  const url = buildURL(BASE_URL, `/files/download/${fileId}`);

  return fetch(url, {
    method: 'GET',
    headers: {
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
  })
    .then((response) => {
      if (response.ok) return response.blob();
      throw new Error('Network response was not ok.');
    })
    .then((blob) => {
      // Create a local URL for the blob object
      const localUrl = window.URL.createObjectURL(blob);

      // Create a temporary anchor tag to trigger the download
      const a = document.createElement('a');
      a.href = localUrl;
      a.download = filename || 'downloaded_file';
      document.body.appendChild(a);
      a.click();

      // Clean up by revoking the object URL and removing the temporary anchor tag
      window.URL.revokeObjectURL(localUrl);
      a.remove();
    })
    .catch((error) => console.error('Download error:', error));
}

export function downloadFile(fileId: string) {
  const token = localStorage.getItem('token');
  const url = buildURL(BASE_URL, `/files/download/${fileId}`);

  return fetch(url, {
    method: 'GET',
    headers: {
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
  })
    .then((response) => {
      if (response.ok) return response.blob();
      throw new Error('Network response was not ok.');
    })
    .then((blob) => {
      return window.URL.createObjectURL(blob);
    })
    .catch((error) => console.error('Download error:', error));
}

export function downloadThumbnail(fileId: string): Promise<string | undefined> {
  const token = localStorage.getItem('token');
  const url = buildURL(BASE_URL, `/files/download/${fileId}?thumbnail=true`);

  return fetch(url, {
    method: 'GET',
    headers: {
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
  })
    .then((response) => {
      if (response.ok) return response.blob();
      throw new Error('Network response was not ok.');
    })
    .then((blob) => {
      return window.URL.createObjectURL(blob);
    })
    .catch((error) => {
      console.error('Thumbnail download error:', error);
      return undefined;
    });
}

export interface FilesResponse {
  files: File[];
  page: number;
  per_page: number;
  total: number;
  total_pages: number;
  search?: string;
  storage_used: number;
  max_storage: number;
}

export interface GetAllFilesParams {
  page?: number;
  perPage?: number;
  search?: string;
  filetype?: string;
  unlinked?: boolean;
  sort?: string;
  order?: string;
}

export function getAllFiles(
  page: number = 1,
  perPage: number = 20,
  search: string = '',
  filetype?: string,
  unlinked?: boolean,
  sort?: string,
  order?: string,
  tag?: string,
): Promise<FilesResponse> {
  const params: Record<string, string | number | undefined> = {
    page: page,
    per_page: perPage,
  };

  if (search.trim()) {
    params.search = search.trim();
  }

  if (filetype && filetype.trim()) {
    params.filetype = filetype.trim();
  }

  if (tag && tag.trim()) {
    params.tag = tag.trim();
  }

  if (unlinked) {
    params.unlinked = 'true';
  }

  if (sort && sort.trim()) {
    params.sort = sort.trim();
  }

  if (order && order.trim()) {
    params.order = order.trim();
  }

  return getData(apiClient.get<FilesResponse>('/files', { params }));
}

export function deleteFile(fileId: number): Promise<GenericResponse> {
  return getData(apiClient.delete<GenericResponse>(`/files/${fileId}`));
}

export function editFile(
  fileId: string,
  updateData: EditFileMetadataParams,
): Promise<File> {
  return getData(apiClient.patch<File>(`/files/${fileId}`, updateData));
}

// File tag management functions

export interface CreateFileTagResponse {
  id: number;
  name: string;
}

export function createFileTag(name: string): Promise<CreateFileTagResponse> {
  return getData(
    apiClient.post<CreateFileTagResponse>('/files/tags', { name }),
  );
}

export function getFileTags(): Promise<FileTag[]> {
  return getData(apiClient.get<FileTag[]>('/files/tags'));
}

export function tagFile(fileId: number, tagNames: string[]): Promise<void> {
  return getData(
    apiClient.post(`/files/${fileId}/tags`, { tag_names: tagNames }),
  );
}

export function untagFile(fileId: number, tagName: string): Promise<void> {
  return getData(
    apiClient.delete(`/files/${fileId}/tags/${encodeURIComponent(tagName)}`),
  );
}

// Epub import function
export function importEpub(
  fileId: number,
  cardId?: string,
): Promise<ImportEpubResponse> {
  return getData(
    apiClient.post<ImportEpubResponse>(`/files/${fileId}/import-epub`, {
      card_id: cardId || '',
    }),
  );
}
