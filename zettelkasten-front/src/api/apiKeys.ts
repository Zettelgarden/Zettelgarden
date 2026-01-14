import {
  APIKeyResponse,
  CreateAPIKeyRequest,
  CreateAPIKeyResponse,
  ListAPIKeysResponse,
} from "../models/APIKey";
import { checkStatus } from "./common";

const base_url = import.meta.env.VITE_URL;

export async function listAPIKeys(): Promise<APIKeyResponse[]> {
  let token = localStorage.getItem("token");
  const url = `${base_url}/api-keys`;

  const response = await fetch(url, {
    headers: { Authorization: `Bearer ${token}` },
  });

  const result = await checkStatus(response);
  if (result) {
    const data: ListAPIKeysResponse = await result.json();
    return data.api_keys;
  } else {
    throw new Error("Response is undefined");
  }
}

export async function createAPIKey(
  request: CreateAPIKeyRequest
): Promise<CreateAPIKeyResponse> {
  let token = localStorage.getItem("token");
  const url = `${base_url}/api-keys`;

  const response = await fetch(url, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(request),
  });

  const result = await checkStatus(response);
  if (result) {
    return result.json() as Promise<CreateAPIKeyResponse>;
  } else {
    throw new Error("Response is undefined");
  }
}

export async function revokeAPIKey(id: number): Promise<void> {
  let token = localStorage.getItem("token");
  const url = `${base_url}/api-keys/${id}`;

  const response = await fetch(url, {
    method: "DELETE",
    headers: { Authorization: `Bearer ${token}` },
  });

  await checkStatus(response);
}