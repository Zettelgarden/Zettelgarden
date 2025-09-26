// API for fetching facts
import { Fact, FactWithCard } from "../models/Fact";
import { PartialCard } from "../models/Card";

const base_url = import.meta.env.VITE_URL;

export interface FactsResponse {
  facts: FactWithCard[];
  page: number;
  per_page: number;
  total: number;
  total_pages: number;
  search?: string;
}

export async function getAllFacts(page: number = 1, perPage: number = 20, search: string = ""): Promise<FactsResponse> {
  let token = localStorage.getItem("token");

  const params = new URLSearchParams({
    page: page.toString(),
    per_page: perPage.toString(),
  });

  if (search.trim()) {
    params.append("search", search.trim());
  }

  const res = await fetch(`${base_url}/facts?${params.toString()}`, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });
  if (!res.ok) {
    throw new Error("Failed to fetch facts");
  }

  return res.json();
}

export async function getFactCards(factId: number): Promise<PartialCard[]> {
  const token = localStorage.getItem("token");
  const res = await fetch(`${base_url}/facts/${factId}/cards`, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });
  if (!res.ok) {
    throw new Error("Failed to fetch fact cards");
  }
  return res.json();
}

export async function mergeFacts(fact1Id: number, fact2Id: number): Promise<void> {
  const token = localStorage.getItem("token");
  const res = await fetch(`${base_url}/facts/merge`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ fact1_id: fact1Id, fact2_id: fact2Id }),
  });
  if (!res.ok) {
    throw new Error("Failed to merge facts");
  }
}

export async function getSimilarFacts(factId: number, limit = 10): Promise<FactWithCard[]> {
  const token = localStorage.getItem("token");
  const res = await fetch(`${base_url}/facts/${factId}/similar?limit=${limit}`, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });
  if (!res.ok) {
    throw new Error("Failed to fetch similar facts");
  }
  return res.json();
}

export async function getCardFacts(cardId: number): Promise<Fact[]> {
  let token = localStorage.getItem("token");
  const res = await fetch(`${base_url}/cards/${cardId}/facts`, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });
  if (!res.ok) {
    throw new Error("Failed to fetch facts");
  }
  return res.json();
}

export async function getFact(factId: number): Promise<FactWithCard> {
  const token = localStorage.getItem("token");
  const res = await fetch(`${base_url}/facts/${factId}`, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });
  if (!res.ok) {
    throw new Error("Failed to fetch fact");
  }
  return res.json();
}

export async function deleteFact(factId: number): Promise<void> {
  const token = localStorage.getItem("token");
  const res = await fetch(`${base_url}/facts/${factId}`, {
    method: "DELETE",
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });
  if (!res.ok) {
    throw new Error("Failed to delete fact");
  }
}

export async function linkFactToCard(factId: number, cardId: number): Promise<void> {
  const token = localStorage.getItem("token");
  const res = await fetch(`${base_url}/facts/${factId}/cards/${cardId}`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
  });
  if (!res.ok) {
    throw new Error("Failed to link fact to card");
  }
}

export async function updateFact(factId: number, fact: string): Promise<void> {
  const token = localStorage.getItem("token");
  const res = await fetch(`${base_url}/facts/${factId}`, {
    method: "PUT",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ fact }),
  });
  if (!res.ok) {
    throw new Error("Failed to update fact");
  }
}
