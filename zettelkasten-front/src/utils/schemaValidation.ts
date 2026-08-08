import { FieldDefinition } from '../models/Schema';

/**
 * Returns the names of required schema fields whose value is missing or empty,
 * mirroring the backend rule (services.ValidateStructuredData, bead s2l):
 * null / undefined / whitespace-only string / empty array all count as empty.
 * Zero numbers and false booleans are legitimate values.
 */
export function getMissingRequiredFields(
  fields: FieldDefinition[],
  data: Record<string, unknown> | null | undefined,
): string[] {
  const missing: string[] = [];
  for (const field of fields) {
    if (!field.required) continue;
    const value = data?.[field.name];
    if (isEmptyValue(value)) {
      missing.push(field.name);
    }
  }
  return missing;
}

export function isEmptyValue(value: unknown): boolean {
  if (value === null || value === undefined) return true;
  if (typeof value === 'string') return value.trim() === '';
  if (Array.isArray(value)) return value.length === 0;
  return false;
}
