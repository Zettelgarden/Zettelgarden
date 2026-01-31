import { apiClient, getData } from "./client";

export function getBillingPortalUrl(): Promise<{ url: string }> {
  return getData(apiClient.get<{ url: string }>("/billing/portal"));
}

export function getStripePublicKey(): Promise<{ key: string }> {
  return getData(apiClient.get<{ key: string }>("/billing/public-key"));
}
