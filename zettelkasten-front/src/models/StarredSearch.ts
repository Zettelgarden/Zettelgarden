export interface StarredSearch {
    id: number;
    title: string;
    searchTerm: string;
    searchConfig: SearchConfig;
    created_at: Date;
}

export interface SearchConfig {
    useClassicSearch: boolean;
    useFullText: boolean;
    onlyParentCards: boolean;
    onlyEmptyCardId: boolean;
    showEntities: boolean;
    showFacts: boolean;
    showCards: boolean;
    showEmails: boolean;
    showPreview: boolean;
    sortBy: string;
    currentPage: number;
    searchType: string; // "classic" or "typesense"
    rerank: boolean;
    schemaId?: number | null;
}
