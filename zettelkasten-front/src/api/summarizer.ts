import { apiClient, getData } from "./client";

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
  return getData(apiClient.get<SectionAnalysis[]>(`/cards/${cardId}/analysis`));
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
