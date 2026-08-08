/**
 * Newsletter subscription for the standalone marketing site.
 *
 * The in-app mailing-list API no longer exists (removed with the SaaS chrome,
 * 6er.11/6er.14), so this site POSTs to an optional external endpoint instead.
 * Set VITE_NEWSLETTER_ENDPOINT (e.g. a Buttondown/Formspree-style URL that
 * accepts JSON `{ "email": "..." }`) to wire the form to your list provider.
 * Without it, the form succeeds as a no-op so the site works standalone.
 */

const NEWSLETTER_ENDPOINT: string | undefined =
  import.meta.env.VITE_NEWSLETTER_ENDPOINT;

export async function subscribeToNewsletter(email: string): Promise<void> {
  if (!NEWSLETTER_ENDPOINT) {
    // No list configured: treat as success so the marketing site stands alone.
    return;
  }
  const res = await fetch(NEWSLETTER_ENDPOINT, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email }),
  });
  if (!res.ok) {
    throw new Error(`Newsletter request failed (${res.status})`);
  }
}
