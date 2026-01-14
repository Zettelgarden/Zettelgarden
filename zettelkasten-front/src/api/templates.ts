import { CardTemplate, defaultCardTemplate } from "../models/Card";
import { checkStatus } from "./common";

const base_url = import.meta.env.VITE_URL;

/**
 * Get all templates for the current user
 * @returns A promise that resolves to an array of templates
 */
export function getTemplates(): Promise<CardTemplate[]> {
    let token = localStorage.getItem("token");
    return fetch(`${base_url}/templates`, {
        headers: { Authorization: `Bearer ${token}` },
    })
        .then(checkStatus)
        .then((response) => {
            if (response) {
                return response.json().then((templates: CardTemplate[]) => {
                    if (templates === null) {
                        return [];
                    }
                    return templates.map((template) => ({
                        ...template,
                        created_at: new Date(template.created_at),
                        updated_at: new Date(template.updated_at),
                    }));
                });
            } else {
                return Promise.reject(new Error("Response is undefined"));
            }
        });
}

/**
 * Save current card content as a template
 * @param name The display name for the template
 * @param title The title for the card when template is applied
 * @param body The body content for the template
 * @returns A promise that resolves to the created template
 */
export function saveAsTemplate(name: string, title: string, body: string): Promise<CardTemplate> {
    let token = localStorage.getItem("token");
    return fetch(`${base_url}/templates`, {
        method: "POST",
        headers: {
            Authorization: `Bearer ${token}`,
            "Content-Type": "application/json",
        },
        body: JSON.stringify({ name, title, body }),
    })
        .then(checkStatus)
        .then((response) => {
            if (response) {
                return response.json().then((template: CardTemplate) => ({
                    ...template,
                    created_at: new Date(template.created_at),
                    updated_at: new Date(template.updated_at),
                }));
            } else {
                return Promise.reject(new Error("Response is undefined"));
            }
        });
}

/**
 * Get a specific template by ID
 * @param id The ID of the template to retrieve
 * @returns A promise that resolves to the template
 */
export function getTemplate(id: number): Promise<CardTemplate> {
    let token = localStorage.getItem("token");
    return fetch(`${base_url}/templates/${id}`, {
        headers: { Authorization: `Bearer ${token}` },
    })
        .then(checkStatus)
        .then((response) => {
            if (response) {
                return response.json().then((template: CardTemplate) => ({
                    ...template,
                    created_at: new Date(template.created_at),
                    updated_at: new Date(template.updated_at),
                }));
            } else {
                return Promise.reject(new Error("Response is undefined"));
            }
        });
}

/**
 * Update an existing template
 * @param id The ID of the template to update
 * @param name The new display name for the template
 * @param title The new title for the template
 * @param body The new body content for the template
 * @returns A promise that resolves to the updated template
 */
export function updateTemplate(id: number, name: string, title: string, body: string): Promise<CardTemplate> {
    let token = localStorage.getItem("token");
    return fetch(`${base_url}/templates/${id}`, {
        method: "PUT",
        headers: {
            Authorization: `Bearer ${token}`,
            "Content-Type": "application/json",
        },
        body: JSON.stringify({ name, title, body }),
    })
        .then(checkStatus)
        .then((response) => {
            if (response) {
                return response.json().then((template: CardTemplate) => ({
                    ...template,
                    created_at: new Date(template.created_at),
                    updated_at: new Date(template.updated_at),
                }));
            } else {
                return Promise.reject(new Error("Response is undefined"));
            }
        });
}

/**
 * Delete a template
 * @param id The ID of the template to delete
 * @returns A promise that resolves when the template is deleted
 */
export function deleteTemplate(id: number): Promise<void> {
    let token = localStorage.getItem("token");
    return fetch(`${base_url}/templates/${id}`, {
        method: "DELETE",
        headers: { Authorization: `Bearer ${token}` },
    })
        .then(checkStatus)
        .then(() => { });
}
