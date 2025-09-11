import { checkStatus } from "./common";

const base_url = import.meta.env.VITE_URL;

export interface SummarizeJobResponse {
    id: number;
    status: string;
    result?: string;
}

export interface Argument {
    argument: string;
    importance: number;
}

export interface ThesisEntry {
    thesis: string;
    facts: string[];
    arguments: Argument[];
}

export interface SectionAnalysis {
    section: string;
    theses: ThesisEntry[];
}

export function fetchAnalysisForCard(cardId: number): Promise<SectionAnalysis[]> {
    let token = localStorage.getItem("token");
    return fetch(`${base_url}/cards/${cardId}/analysis`, {
        headers: { Authorization: `Bearer ${token}` },
    })
        .then(checkStatus)
        .then((response) => {
            if (response) {
                return response.json() as Promise<SectionAnalysis[]>;
            } else {
                return Promise.reject(new Error("Response is undefined"));
            }
        });
}

export function fetchSummariesForCard(cardId: number): Promise<SummarizeJobResponse[]> {
    let token = localStorage.getItem("token");
    return fetch(`${base_url}/cards/${cardId}/summaries`, {
        headers: { Authorization: `Bearer ${token}` },
    })
        .then(checkStatus)
        .then((response) => {
            if (response) {
                return response.json() as Promise<SummarizeJobResponse[]>;
            } else {
                return Promise.reject(new Error("Response is undefined"));
            }
        });
}

export function fetchSummarizations(): Promise<SummarizeJobResponse[]> {
    let token = localStorage.getItem("token");
    return fetch(base_url + "/summarizations", {
        headers: { Authorization: `Bearer ${token}` },
    })
        .then(checkStatus)
        .then((response) => {
            if (response) {
                return response.json() as Promise<SummarizeJobResponse[]>;
            } else {
                return Promise.reject(new Error("Response is undefined"));
            }
        });
}

export function createSummarization(text: string): Promise<SummarizeJobResponse> {
    let token = localStorage.getItem("token");
    return fetch(base_url + "/summarize", {
        method: "POST",
        headers: {
            Authorization: `Bearer ${token}`,
            "Content-Type": "application/json",
        },
        body: JSON.stringify({ text }),
    })
        .then(checkStatus)
        .then((response) => {
            if (response) {
                return response.json() as Promise<SummarizeJobResponse>;
            } else {
                return Promise.reject(new Error("Response is undefined"));
            }
        });
}

export function fetchSummarization(id: number): Promise<SummarizeJobResponse> {
    let token = localStorage.getItem("token");
    return fetch(`${base_url}/summarize/${id}`, {
        headers: { Authorization: `Bearer ${token}` },
    })
        .then(checkStatus)
        .then((response) => {
            if (response) {
                return response.json() as Promise<SummarizeJobResponse>;
            } else {
                return Promise.reject(new Error("Response is undefined"));
            }
        });
}
