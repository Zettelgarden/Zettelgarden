import { CardTemplate, defaultCardTemplate } from '../models/Card';
import { apiClient, getData } from './client';

function parseTemplateDates(template: CardTemplate): CardTemplate {
  return {
    ...template,
    created_at: new Date(template.created_at),
    updated_at: new Date(template.updated_at),
  };
}

/**
 * Get all templates for the current user
 * @returns A promise that resolves to an array of templates
 */
export function getTemplates(): Promise<CardTemplate[]> {
  return getData(apiClient.get<CardTemplate[]>(`/templates`))
    .then((templates) => templates || [])
    .then((templates) => templates.map(parseTemplateDates));
}

/**
 * Save current card content as a template
 * @param name The display name for the template
 * @param title The title for the card when template is applied
 * @param body The body content for the template
 * @returns A promise that resolves to the created template
 */
export function saveAsTemplate(
  name: string,
  title: string,
  body: string,
): Promise<CardTemplate> {
  return getData(
    apiClient.post<CardTemplate>(`/templates`, { name, title, body }),
  ).then(parseTemplateDates);
}

/**
 * Get a specific template by ID
 * @param id The ID of the template to retrieve
 * @returns A promise that resolves to the template
 */
export function getTemplate(id: number): Promise<CardTemplate> {
  return getData(apiClient.get<CardTemplate>(`/templates/${id}`)).then(
    parseTemplateDates,
  );
}

/**
 * Update an existing template
 * @param id The ID of the template to update
 * @param name The new display name for the template
 * @param title The new title for the template
 * @param body The new body content for the template
 * @returns A promise that resolves to the updated template
 */
export function updateTemplate(
  id: number,
  name: string,
  title: string,
  body: string,
): Promise<CardTemplate> {
  return getData(
    apiClient.put<CardTemplate>(`/templates/${id}`, { name, title, body }),
  ).then(parseTemplateDates);
}

/**
 * Delete a template
 * @param id The ID of the template to delete
 * @returns A promise that resolves when the template is deleted
 */
export function deleteTemplate(id: number): Promise<void> {
  return getData(apiClient.delete<void>(`/templates/${id}`));
}
