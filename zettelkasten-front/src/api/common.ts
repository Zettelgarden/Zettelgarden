export interface SchemaApiResponse {
  id: number;
  name: string;
  owner_id: number;
  fields: unknown[];
  created_at: string;
  updated_at: string;
  is_deleted: boolean;
}

export function parseSchemaDates<T extends SchemaApiResponse>(schema: T): T {
  return {
    ...schema,
    created_at: new Date(schema.created_at) as T["created_at"],
    updated_at: new Date(schema.updated_at) as T["updated_at"],
  };
}

export async function checkStatus(response: Response) {
  if (response.status === 401 || response.status === 422) {
    const token = localStorage.getItem("token");
    if (token) {
      localStorage.removeItem("token");
      window.location.reload();
    }
    return;
  }
  // If the response is ok, return the response to continue the promise chain
  if (response.ok) {
    return response;
  }

  // Try to extract error message from response body
  let errorText: string;
  try {
    errorText = await response.text();
  } catch (e) {
    // If we can't read the body, fall back to status code
    errorText = `Request failed with status: ${response.status}`;
  }

  // Throw the error with the extracted message
  throw new Error(errorText || `Request failed with status: ${response.status}`);
}
