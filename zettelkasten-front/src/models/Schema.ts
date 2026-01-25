export interface FieldDefinition {
  name: string;
  type: "text" | "number" | "date" | "boolean" | "select" | "multi-select" | "link_to_card";
  required: boolean;
  options?: string[];
}

export interface SchemaDefinition {
  id: number;
  name: string;
  slug: string;
  owner_id: number;
  fields: FieldDefinition[];
  created_at: Date;
  updated_at: Date;
  is_deleted: boolean;
}
