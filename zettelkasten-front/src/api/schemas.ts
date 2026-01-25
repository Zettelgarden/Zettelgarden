import { SchemaDefinition } from "../models/Schema";
import { checkStatus } from "./common";

const base_url = import.meta.env.VITE_URL;

function parseSchemaDates(schema: SchemaDefinition): SchemaDefinition {
  return {
    ...schema,
    created_at: new Date(schema.created_at),
    updated_at: new Date(schema.updated_at),
  };
}

export function fetchSchemas(): Promise<SchemaDefinition[]> {
  const token = localStorage.getItem("token");
  const url = base_url + "/schemas";

  return fetch(url, {
    headers: { Authorization: `Bearer ${token}` },
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json().then((schemas: SchemaDefinition[] | null) => {
          if (!schemas) return [];
          return schemas.map(parseSchemaDates);
        });
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export function fetchSchema(id: number): Promise<SchemaDefinition> {
  const token = localStorage.getItem("token");
  const url = base_url + `/schemas/${id}`;

  return fetch(url, {
    headers: { Authorization: `Bearer ${token}` },
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json().then((schema: SchemaDefinition) => {
          return parseSchemaDates(schema);
        });
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export interface CreateSchemaParams {
  name: string;
  fields: SchemaDefinition["fields"];
}

export function createSchema(params: CreateSchemaParams): Promise<SchemaDefinition> {
  const token = localStorage.getItem("token");
  const url = base_url + "/schemas";

  return fetch(url, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(params),
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json().then((schema: SchemaDefinition) => {
          return parseSchemaDates(schema);
        });
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export interface UpdateSchemaParams {
  name: string;
  fields: SchemaDefinition["fields"];
}

export function updateSchema(id: number, params: UpdateSchemaParams): Promise<SchemaDefinition> {
  const token = localStorage.getItem("token");
  const url = base_url + `/schemas/${id}`;

  return fetch(url, {
    method: "PUT",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(params),
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json().then((schema: SchemaDefinition) => {
          return parseSchemaDates(schema);
        });
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}

export interface DeleteSchemaResponse {
  deleted: boolean;
  warning?: string;
  cards_affected?: number;
}

export function deleteSchema(id: number): Promise<DeleteSchemaResponse> {
  const token = localStorage.getItem("token");
  const url = base_url + `/schemas/${id}`;

  return fetch(url, {
    method: "DELETE",
    headers: {
      Authorization: `Bearer ${token}`,
    },
  })
    .then(checkStatus)
    .then((response) => {
      if (response) {
        return response.json();
      } else {
        return Promise.reject(new Error("Response is undefined"));
      }
    });
}
