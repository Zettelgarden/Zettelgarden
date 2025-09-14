import { StarredSearch, SearchConfig } from "../models/StarredSearch";
import { checkStatus } from "./common";

const base_url = import.meta.env.VITE_URL;

/**
 * Save a search configuration to starred searches
 * @param title The title for the starred search
 * @param searchTerm The search term
 * @param searchConfig The search configuration options
 * @returns A promise that resolves when the search is starred
 */
export function starSearch(
    title: string,
    searchTerm: string,
    searchConfig: SearchConfig
): Promise<void> {
    const url = `${base_url}/searches/star`;
    let token = localStorage.getItem("token");

    return fetch(url, {
        method: "POST",
        headers: {
            Authorization: `Bearer ${token}`,
            "Content-Type": "application/json"
        },
        body: JSON.stringify({
            title,
            search_term: searchTerm,
            search_config: searchConfig
        }),
    })
        .then(checkStatus)
        .then(() => {
            return;
        });
}

/**
 * Remove a starred search
 * @param id The ID of the starred search to remove
 * @returns A promise that resolves when the search is unstarred
 */
export function unstarSearch(id: number): Promise<void> {
    const url = `${base_url}/searches/star/${id}`;
    let token = localStorage.getItem("token");

    return fetch(url, {
        method: "DELETE",
        headers: { Authorization: `Bearer ${token}` },
    })
        .then(checkStatus)
        .then(() => {
            return;
        });
}

/**
 * Get all starred searches for the current user
 * @returns A promise that resolves to an array of starred searches
 */
export function getStarredSearches(): Promise<StarredSearch[]> {
    const url = `${base_url}/searches/starred`;
    let token = localStorage.getItem("token");

    return fetch(url, {
        headers: { Authorization: `Bearer ${token}` },
    })
        .then(checkStatus)
        .then((response) => {
            if (response) {
                return response.json().then((starredSearches: any[]) => {
                    if (starredSearches === null) {
                        return [];
                    }

                    // Transform the response into StarredSearch objects
                    return starredSearches.map((starredSearch) => {
                        return {
                            ...starredSearch,
                            created_at: new Date(starredSearch.created_at),
                        };
                    });
                });
            } else {
                return Promise.reject(new Error("Response is undefined"));
            }
        });
}
