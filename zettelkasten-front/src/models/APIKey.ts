export interface CreateAPIKeyRequest {
  name: string;
  description?: string;
}

export interface APIKeyResponse {
  id: number;
  name: string;
  created_at: string;
  last_used_at: string | null;
  revoked_at: string | null;
  is_active: boolean;
  description: string;
}

export interface CreateAPIKeyResponse extends APIKeyResponse {
  key: string;
}

export interface ListAPIKeysResponse {
  api_keys: APIKeyResponse[];
}
