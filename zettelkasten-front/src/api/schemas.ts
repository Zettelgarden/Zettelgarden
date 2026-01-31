import { SchemaDefinition } from "../models/Schema";
import { apiClient, getData } from "./client";

function parseSchemaDates(schema: SchemaDefinition): SchemaDefinition {
  return {
    ...schema,
    created_at: new Date(schema.created_at),
    updated_at: new Date(schema.updated_at),
  };
}

export function fetchSchemas(): Promise<SchemaDefinition[]> {
  return getData(apiClient.get<SchemaDefinition[]>("/schemas"))
    .then((schemas) => schemas || [])
    .then((schemas) => schemas.map(parseSchemaDates));
}

export function fetchSchema(id: number): Promise<SchemaDefinition> {
  return getData(apiClient.get<SchemaDefinition>(`/schemas/${id}`))
    .then(parseSchemaDates);
}

// Fetch schema by reference (ID or slug)
// The ref can be a numeric ID (e.g., "123") or a slug (e.g., "book-review")
export function fetchSchemaByRef(ref: string): Promise<SchemaDefinition> {
  return getData(apiClient.get<SchemaDefinition>(`/schemas/${ref}`))
    .then(parseSchemaDates);
}

export interface CreateSchemaParams {
  name: string;
  fields: SchemaDefinition["fields"];
}

export function createSchema(params: CreateSchemaParams): Promise<SchemaDefinition> {
  return getData(apiClient.post<SchemaDefinition>("/schemas", params))
    .then(parseSchemaDates);
}

export interface UpdateSchemaParams {
  name: string;
  fields: SchemaDefinition["fields"];
}

export function updateSchema(id: number, params: UpdateSchemaParams): Promise<SchemaDefinition> {
  return getData(apiClient.put<SchemaDefinition>(`/schemas/${id}`, params))
    .then(parseSchemaDates);
}

export interface DeleteSchemaResponse {
  deleted: boolean;
  warning?: string;
  cards_affected?: number;
}

export function deleteSchema(id: number): Promise<DeleteSchemaResponse> {
  return getData(apiClient.delete<DeleteSchemaResponse>(`/schemas/${id}`));
}
