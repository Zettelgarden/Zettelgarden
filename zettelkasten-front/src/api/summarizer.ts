import { apiClient, getData } from "./client";

export interface SummarizeJobResponse {
  id: number;
  status: string;
  result?: string;
}

export function fetchSummariesForCard(cardId: number): Promise<SummarizeJobResponse[]> {
  return getData(apiClient.get<SummarizeJobResponse[]>(`/cards/${cardId}/summaries`));
}

export function fetchSummarizations(): Promise<SummarizeJobResponse[]> {
  return getData(apiClient.get<SummarizeJobResponse[]>("/summarizations"));
}

export function createSummarization(text: string): Promise<SummarizeJobResponse> {
  return getData(apiClient.post<SummarizeJobResponse>("/summarize", { text }));
}

export function fetchSummarization(id: number): Promise<SummarizeJobResponse> {
  return getData(apiClient.get<SummarizeJobResponse>(`/summarize/${id}`));
}
