import {
  APIKeyResponse,
  CreateAPIKeyRequest,
  CreateAPIKeyResponse,
  ListAPIKeysResponse,
} from "../models/APIKey";
import { apiClient, getData } from "./client";

export async function listAPIKeys(): Promise<APIKeyResponse[]> {
  const data = await getData(apiClient.get<ListAPIKeysResponse>("/api-keys"));
  return data.api_keys;
}

export async function createAPIKey(
  request: CreateAPIKeyRequest
): Promise<CreateAPIKeyResponse> {
  return getData(apiClient.post<CreateAPIKeyResponse>("/api-keys", request));
}

export async function revokeAPIKey(id: number): Promise<void> {
  return getData(apiClient.delete<void>(`/api-keys/${id}`));
}
