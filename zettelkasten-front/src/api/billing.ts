import { apiClient, getData } from './client';

export function getBillingPortalUrl(): Promise<{ url: string }> {
  return getData(apiClient.get<{ url: string }>('/billing/portal'));
}

export function getStripePublicKey(): Promise<{ key: string }> {
  return getData(apiClient.get<{ key: string }>('/billing/public-key'));
}

export interface BillingStatus {
  enabled: boolean;
}

let billingStatusPromise: Promise<BillingStatus> | null = null;

/**
 * Whether Stripe billing is enabled on this instance (STRIPE_ENABLED on the
 * server). Cached for the page lifetime; if the status request fails we
 * optimistically assume billing is enabled (the pre-existing behavior) rather
 * than locking anyone out.
 */
export function getBillingStatus(): Promise<BillingStatus> {
  if (!billingStatusPromise) {
    billingStatusPromise = getData(
      apiClient.get<BillingStatus>('/billing/status'),
    ).catch(() => ({ enabled: true }));
  }
  return billingStatusPromise;
}
